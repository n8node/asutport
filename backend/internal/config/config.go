package config

import (
	"fmt"
	"strings"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Environment string `env:"ENVIRONMENT" envDefault:"development"`
	LogLevel    string `env:"LOG_LEVEL" envDefault:"info"`
	Version     string `env:"VERSION" envDefault:"0.1.0"`
	ServerPort  string `env:"SERVER_PORT" envDefault:"8080"`

	PostgresHost     string `env:"POSTGRES_HOST" envDefault:"postgres"`
	PostgresPort     string `env:"POSTGRES_PORT" envDefault:"5432"`
	PostgresDB       string `env:"POSTGRES_DB" envDefault:"asutport"`
	PostgresUser     string `env:"POSTGRES_USER" envDefault:"asutport"`
	PostgresPassword string `env:"POSTGRES_PASSWORD" envDefault:""`
	PostgresSSLMode  string `env:"POSTGRES_SSLMODE" envDefault:"disable"`

	JWTSecret  string `env:"JWT_SECRET" envDefault:""`
	APIKeySalt string `env:"API_KEY_SALT" envDefault:""`

	S3Endpoint     string `env:"S3_ENDPOINT" envDefault:"http://minio:9000"`
	S3Region       string `env:"S3_REGION" envDefault:"us-east-1"`
	S3Bucket       string `env:"S3_BUCKET" envDefault:"asutport"`
	S3AccessKey    string `env:"S3_ACCESS_KEY" envDefault:""`
	S3SecretKey    string `env:"S3_SECRET_KEY" envDefault:""`
	S3UsePathStyle bool   `env:"S3_USE_PATH_STYLE" envDefault:"true"`

	OpenRouterAPIKey       string `env:"OPENROUTER_API_KEY" envDefault:""`
	OpenRouterModelQualify string `env:"OPENROUTER_MODEL_QUALIFY" envDefault:"google/gemini-2.5-flash"`
	OpenRouterModelAnswer  string `env:"OPENROUTER_MODEL_ANSWER" envDefault:"anthropic/claude-sonnet-4"`
	OpenRouterModelVision  string `env:"OPENROUTER_MODEL_VISION" envDefault:"google/gemini-2.5-pro"`
	OpenRouterModelKB      string `env:"OPENROUTER_MODEL_KB" envDefault:"anthropic/claude-sonnet-4"`
	OpenRouterModelEmbed   string `env:"OPENROUTER_MODEL_EMBED" envDefault:"openai/text-embedding-3-large"`

	Domain       string `env:"DOMAIN" envDefault:"asutport.ru"`
	CertbotEmail string `env:"CERTBOT_EMAIL" envDefault:"erman.ai@yandex.ru"`
}

func (c *Config) PublicAppBaseURL() string {
	if strings.EqualFold(strings.TrimSpace(c.Environment), "development") {
		return "http://localhost:3000/app"
	}
	domain := strings.TrimSpace(c.Domain)
	if domain == "" {
		domain = "asutport.ru"
	}
	return "https://" + domain + "/app"
}

func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("parse env: %w", err)
	}
	return &c, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresDB,
		c.PostgresSSLMode,
	)
}

func (c *Config) S3Configured() bool {
	return c.S3Bucket != "" && c.S3AccessKey != "" && c.S3SecretKey != ""
}
