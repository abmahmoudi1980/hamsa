// Package config loads backend configuration from a YAML file
// (backend/config.yaml) with environment-variable overrides.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the root configuration structure. Field defaults match
// backend/config.example.yaml; only values present in the loaded file
// (or environment) override them.
type Config struct {
	App     App     `yaml:"app"`
	DB      DB      `yaml:"db"`
	Auth    Auth    `yaml:"auth"`
	SMS     SMS     `yaml:"sms"`
	Payment Payment `yaml:"payment"`
	Storage Storage `yaml:"storage"`
	Push    Push    `yaml:"push"`
}

// App holds application-level settings.
type App struct {
	Env     string `yaml:"env"`
	Addr    string `yaml:"addr"`
	BaseURL string `yaml:"base_url"`
}

// IsDev reports whether the server runs in development mode.
func (a App) IsDev() bool { return a.Env == "dev" }

// DB holds PostgreSQL connection settings.
type DB struct {
	DSN         string `yaml:"dsn"`
	AutoMigrate bool   `yaml:"auto_migrate"`
}

// Auth holds token settings.
type Auth struct {
	JWTSecret  string        `yaml:"jwt_secret"`
	AccessTTL  time.Duration `yaml:"access_ttl"`
	RefreshTTL time.Duration `yaml:"refresh_ttl"`
}

// SMS holds the SMS provider selection and provider-specific settings.
type SMS struct {
	Provider  string `yaml:"provider"` // console | kavenegar
	Kavenegar struct {
		APIKey string `yaml:"api_key"`
		Sender string `yaml:"sender"`
	} `yaml:"kavenegar"`
}

// Payment holds the payment gateway selection and provider-specific settings.
type Payment struct {
	Provider string `yaml:"provider"` // mock | zarinpal
	Zarinpal struct {
		MerchantID string `yaml:"merchant_id"`
		Sandbox    bool   `yaml:"sandbox"`
	} `yaml:"zarinpal"`
}

// Storage holds file-storage settings.
type Storage struct {
	Path string `yaml:"path"`
}

// Push holds push-notification settings.
type Push struct {
	FCMCredentialsFile string `yaml:"fcm_credentials_file"`
}

// Load reads configuration from the file at HAMSA_CONFIG (default
// "config.yaml" relative to the working directory), applies defaults and
// environment overrides, and validates required values.
func Load() (*Config, error) {
	path := os.Getenv("HAMSA_CONFIG")
	if path == "" {
		path = "config.yaml"
	}

	cfg := &Config{
		App: App{Env: "dev", Addr: ":8080", BaseURL: "http://localhost:8080"},
		DB:  DB{AutoMigrate: true},
		Auth: Auth{
			AccessTTL:  15 * time.Minute,
			RefreshTTL: 30 * 24 * time.Hour,
		},
		SMS:     SMS{Provider: "console"},
		Payment: Payment{Provider: "mock"},
		Storage: Storage{Path: "./data/files"},
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found at %s — copy config.example.yaml to config.yaml", path)
		}
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	if err := yaml.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	applyEnvOverrides(cfg)

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// applyEnvOverrides maps selected environment variables onto the config so
// production deployments can avoid secrets on disk.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("HAMSA_APP_ENV"); v != "" {
		cfg.App.Env = v
	}
	if v := os.Getenv("HAMSA_DB_DSN"); v != "" {
		cfg.DB.DSN = v
	}
	if v := os.Getenv("HAMSA_JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v := os.Getenv("HAMSA_SMS_PROVIDER"); v != "" {
		cfg.SMS.Provider = v
	}
	if v := os.Getenv("HAMSA_PAYMENT_PROVIDER"); v != "" {
		cfg.Payment.Provider = v
	}
}

func (c *Config) validate() error {
	if c.App.Env != "dev" && c.App.Env != "production" {
		return fmt.Errorf("app.env must be \"dev\" or \"production\", got %q", c.App.Env)
	}
	if c.App.Addr == "" {
		return fmt.Errorf("app.addr must not be empty")
	}
	if c.DB.DSN == "" {
		return fmt.Errorf("db.dsn must not be empty")
	}
	if c.Auth.JWTSecret == "" {
		return fmt.Errorf("auth.jwt_secret must not be empty")
	}
	if c.Auth.AccessTTL <= 0 || c.Auth.RefreshTTL <= 0 {
		return fmt.Errorf("auth token TTLs must be positive")
	}
	if c.Storage.Path == "" {
		return fmt.Errorf("storage.path must not be empty")
	}
	return nil
}
