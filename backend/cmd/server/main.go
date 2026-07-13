package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/n8node/asutport/internal/config"
	"github.com/n8node/asutport/internal/email"
	"github.com/n8node/asutport/internal/handler"
	"github.com/n8node/asutport/internal/middleware"
	"github.com/n8node/asutport/internal/repository"
	s3store "github.com/n8node/asutport/internal/s3"
	"github.com/n8node/asutport/internal/seed"
	"github.com/n8node/asutport/internal/server"
	"github.com/n8node/asutport/internal/service"
	appmigrations "github.com/n8node/asutport/migrations"
)

func main() {
	seedAdmin := flag.Bool("seed-admin", false, "create platform superadmin user")
	seedEmail := flag.String("email", "", "superadmin email (with --seed-admin)")
	seedPassword := flag.String("password", "", "superadmin password (with --seed-admin)")
	seedName := flag.String("full-name", "ASUTPORT Admin", "superadmin full name (with --seed-admin)")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", slog.Any("err", err))
		os.Exit(1)
	}
	if err := validateSecrets(cfg); err != nil {
		slog.Error("secrets", slog.Any("err", err))
		os.Exit(1)
	}

	logLevel := slog.LevelInfo
	if cfg.LogLevel == "debug" {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	ctx := context.Background()
	pool, err := repository.NewPool(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("postgres", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := runMigrations(cfg.DatabaseURL(), logger); err != nil {
		logger.Error("migrate", slog.Any("err", err))
		os.Exit(1)
	}

	if *seedAdmin {
		if err := seed.Admin(ctx, pool, seed.AdminOptions{
			Email:    *seedEmail,
			Password: *seedPassword,
			FullName: *seedName,
		}, logger); err != nil {
			logger.Error("seed-admin", slog.Any("err", err))
			os.Exit(1)
		}
		return
	}

	s3Client, err := s3store.NewClient(cfg)
	if err != nil {
		logger.Error("s3", slog.Any("err", err))
		os.Exit(1)
	}

	users := repository.NewUserRepo(pool)
	orgs := repository.NewOrgRepo(pool)
	members := repository.NewOrgMemberRepo(pool)
	sessions := repository.NewSessionRepo(pool)
	apiKeys := repository.NewAPIKeyRepo(pool)
	adminSettings := repository.NewAdminSettingsRepo(pool)
	adminUsers := repository.NewAdminUserRepo(pool)
	adminOrgs := repository.NewAdminOrgRepo(pool)
	regVerify := repository.NewRegistrationVerificationRepo(pool)
	emailLoader := email.NewLoader(adminSettings, cfg.JWTSecret)
	authSvc := service.NewAuthService(cfg.JWTSecret, users, members, sessions)

	authH := handler.NewAuthHandler(cfg, users, orgs, members, sessions, regVerify, emailLoader, authSvc)
	orgH := handler.NewOrgHandler(members, orgs)
	adminOrgH := handler.NewAdminOrgHandler(adminOrgs, orgs)
	adminUserH := handler.NewAdminUserHandler(adminUsers)
	keyH := handler.NewAPIKeyHandler(cfg, apiKeys, members)
	adminSettingsH := handler.NewAdminSettingsHandler(cfg, adminSettings)

	authDeps := middleware.AuthDeps{
		Cfg:      cfg,
		Users:    users,
		Sessions: sessions,
		Members:  members,
		Keys:     apiKeys,
	}
	loginRL := middleware.NewLoginRateLimiter(10, time.Minute)

	h := server.New(server.Options{
		Logger: logger,
		Handlers: server.Handlers{
			Health:   handler.NewHealth(cfg.Version, pool, s3Client),
			AuthDeps: authDeps,
			LoginRL:  loginRL,
			Auth: server.AuthHandlers{
				Register:           authH.Register,
				Login:              authH.Login,
				VerifyRegistration: authH.VerifyRegistration,
				Refresh:            authH.Refresh,
				Logout:             authH.Logout,
				Me:                 authH.Me,
				Switch:             authH.SwitchOrg,
			},
			Org: server.OrgHandlers{
				ListMine: orgH.ListMine,
				Current:  orgH.Current,
			},
			AdminOrg: server.AdminOrgHandlers{
				List:         adminOrgH.List,
				Get:          adminOrgH.Get,
				Patch:        adminOrgH.Patch,
				UpdateReview: adminOrgH.UpdateReview,
			},
			AdminUser: server.AdminUserHandlers{
				List:           adminUserH.List,
				Get:            adminUserH.Get,
				PatchActive:    adminUserH.PatchActive,
				RevokeSessions: adminUserH.RevokeSessions,
			},
			APIKey: server.APIKeyHandlers{
				List:   keyH.List,
				Create: keyH.Create,
				Revoke: keyH.Revoke,
			},
			Admin: server.AdminHandlers{
				S3Get:       adminSettingsH.S3Get,
				S3Patch:     adminSettingsH.S3Patch,
				S3Test:      adminSettingsH.S3Test,
				S3CorsHints: adminSettingsH.S3CorsHints,
				SMTPGet:     adminSettingsH.SMTPGet,
				SMTPPatch:   adminSettingsH.SMTPPatch,
				SMTPTest:    adminSettingsH.SMTPTest,
			},
		},
		CORSOrigins: []string{"*"},
	})

	addr := ":" + cfg.ServerPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      120 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("listening", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server", slog.Any("err", err))
			os.Exit(1)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown", slog.Any("err", err))
	}
}

func validateSecrets(c *config.Config) error {
	if len(strings.TrimSpace(c.JWTSecret)) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if strings.TrimSpace(c.APIKeySalt) == "" {
		return fmt.Errorf("API_KEY_SALT must be non-empty")
	}
	return nil
}

func runMigrations(databaseURL string, logger *slog.Logger) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}
	goose.SetBaseFS(appmigrations.FS)
	if err := goose.Up(db, "."); err != nil {
		return err
	}
	logger.Info("migrations applied")
	return nil
}
