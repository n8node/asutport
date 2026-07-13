package seed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/n8node/asutport/internal/auth"
	"github.com/n8node/asutport/internal/repository"
)

type AdminOptions struct {
	Email    string
	Password string
	FullName string
}

func Admin(ctx context.Context, pool *pgxpool.Pool, opts AdminOptions, logger *slog.Logger) error {
	opts.Email = strings.TrimSpace(strings.ToLower(opts.Email))
	opts.FullName = strings.TrimSpace(opts.FullName)
	if opts.Email == "" || opts.Password == "" {
		return fmt.Errorf("email and password are required")
	}
	if opts.FullName == "" {
		opts.FullName = "ASUTPORT Admin"
	}
	hash, err := auth.HashPassword(opts.Password)
	if err != nil {
		return err
	}

	users := repository.NewUserRepo(pool)
	orgs := repository.NewOrgRepo(pool)
	members := repository.NewOrgMemberRepo(pool)

	if _, err := users.GetByEmail(ctx, opts.Email); err == nil {
		return fmt.Errorf("user %s already exists", opts.Email)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return err
	}

	u, err := users.Create(ctx, opts.Email, hash, opts.FullName)
	if err != nil {
		return err
	}

	platformOrg, err := orgs.GetBySlug(ctx, "platform")
	if errors.Is(err, repository.ErrNotFound) {
		platformOrg, err = orgs.Create(ctx, "ASUTPORT Platform", "manufacturer", "platform")
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if _, err := members.Create(ctx, platformOrg.ID, u.ID, "superadmin"); err != nil {
		if !errors.Is(err, repository.ErrConflict) {
			return err
		}
	}
	logger.Info("seed admin complete",
		slog.String("email", u.Email),
		slog.String("org_id", platformOrg.ID.String()),
		slog.String("slug", platformOrg.Slug),
	)
	return nil
}
