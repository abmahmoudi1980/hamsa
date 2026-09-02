package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HAMSA_CONFIG", path)
	return path
}

const baseConfig = `
app:
  env: dev
  addr: ":8080"
db:
  dsn: "host=localhost dbname=test sslmode=disable"
auth:
  jwt_secret: "test-secret-long-enough-for-jwt"
  access_ttl: 15m
  refresh_ttl: 720h
storage:
  path: "./data/files"
`

func TestLoad_MissingFileFails(t *testing.T) {
	t.Setenv("HAMSA_CONFIG", filepath.Join(t.TempDir(), "absent.yaml"))
	if _, err := Load(); err == nil {
		t.Fatal("expected error for missing config file")
	}
}

func TestLoad_EnvOverrides(t *testing.T) {
	writeConfig(t, baseConfig)
	t.Setenv("HAMSA_APP_ENV", "production")
	t.Setenv("HAMSA_DB_DSN", "host=other dbname=prod sslmode=require")
	t.Setenv("HAMSA_JWT_SECRET", "env-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.App.Env != "production" || cfg.DB.DSN != "host=other dbname=prod sslmode=require" || cfg.Auth.JWTSecret != "env-secret" {
		t.Fatalf("env overrides not applied: %+v", cfg)
	}
}

func TestLoad_InvalidEnvRejected(t *testing.T) {
	writeConfig(t, baseConfig)
	t.Setenv("HAMSA_APP_ENV", "staging")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid app.env")
	}
}
