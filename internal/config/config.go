package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Port string `env:"PORT" envDefault:"8080"`
	Env  string `env:"APP_ENV" envDefault:"development"`

	DatabaseURL string `env:"DATABASE_URL,required"`

	JWTAccessSecret  string `env:"JWT_ACCESS_SECRET,required"`
	JWTIssuer        string `env:"JWT_ISSUER" envDefault:"https://api.enmasse.id"`
	JWTServiceSecret string `env:"JWT_SERVICE_SECRET,required"`
	EnmasseURL       string `env:"ENMASSE_INTERNAL_URL"`

	StaffEmail     string        `env:"STAFF_EMAIL,required"`
	StaffPassword  string        `env:"STAFF_PASSWORD,required"`
	StaffJWTSecret string        `env:"STAFF_JWT_SECRET,required"`
	StaffJWTExpiry time.Duration `env:"STAFF_JWT_EXPIRY" envDefault:"8h"`

	CORSOrigins string `env:"CORS_ORIGINS" envDefault:"http://localhost:3000"`

	R2AccountID       string `env:"R2_ACCOUNT_ID"`
	R2AccessKeyID     string `env:"R2_ACCESS_KEY_ID"`
	R2SecretAccessKey string `env:"R2_SECRET_ACCESS_KEY"`
	R2Bucket          string `env:"R2_BUCKET"`
	R2PublicBaseURL   string `env:"R2_PUBLIC_BASE_URL"`

	ConsentHashPepper    string `env:"CONSENT_HASH_PEPPER"`
	LegalDocumentVersion string `env:"LEGAL_DOCUMENT_VERSION" envDefault:"1.0"`
}

func Load() (*Config, error) {
	loadDotEnv()
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("config: %w (copy .env.example to .env in the repo root, then run from that directory)", err)
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if !c.IsProduction() {
		return nil
	}
	if len(c.JWTAccessSecret) < 32 || len(c.JWTServiceSecret) < 32 || len(c.StaffJWTSecret) < 32 {
		return fmt.Errorf("config: JWT secrets must be at least 32 characters in production")
	}
	if strings.Contains(c.JWTAccessSecret, "change-me") || strings.Contains(c.JWTServiceSecret, "change-me") || strings.Contains(c.StaffJWTSecret, "change-me") {
		return fmt.Errorf("config: replace example JWT secrets before production")
	}
	if strings.TrimSpace(c.EnmasseURL) == "" {
		return fmt.Errorf("config: ENMASSE_INTERNAL_URL is required in production")
	}
	if len(c.ConsentHashPepper) < 16 {
		return fmt.Errorf("config: CONSENT_HASH_PEPPER must be at least 16 characters in production")
	}
	return nil
}

func loadDotEnv() {
	dir, err := os.Getwd()
	if err != nil {
		return
	}
	for {
		candidate := filepath.Join(dir, ".env")
		if _, err := os.Stat(candidate); err == nil {
			_ = godotenv.Load(candidate)
			return
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return
		}
		dir = parent
	}
}

func (c *Config) IsProduction() bool {
	return c.Env == "production"
}
