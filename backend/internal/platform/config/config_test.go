package config

import (
	"os"
	"path/filepath"
	"strings"
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

// minimal dev config; smsir fields injected per test.
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

func TestLoad_SmsIrProviderRequiresTemplate(t *testing.T) {
	t.Run("missing template_id fails", func(t *testing.T) {
		writeConfig(t, baseConfig+`
sms:
  provider: smsir
  smsir:
    api_key: "test-key"
`)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "template_id") {
			t.Fatalf("expected template_id validation error, got %v", err)
		}
	})

	t.Run("missing api_key fails", func(t *testing.T) {
		writeConfig(t, baseConfig+`
sms:
  provider: smsir
  smsir:
    template_id: 123456
`)
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "api_key") {
			t.Fatalf("expected api_key validation error, got %v", err)
		}
	})

	t.Run("complete smsir config loads", func(t *testing.T) {
		writeConfig(t, baseConfig+`
sms:
  provider: smsir
  smsir:
    api_key: "test-key"
    template_id: 123456
    param_name: "Code"
`)
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.SMS.SMSIR.TemplateID != 123456 || cfg.SMS.SMSIR.ParamName != "Code" {
			t.Fatalf("smsir settings not parsed: %+v", cfg.SMS.SMSIR)
		}
	})

	t.Run("env overrides", func(t *testing.T) {
		writeConfig(t, baseConfig+`
sms:
  provider: console
`)
		t.Setenv("HAMSA_SMS_PROVIDER", "smsir")
		t.Setenv("HAMSA_SMS_SMSIR_API_KEY", "env-key")
		t.Setenv("HAMSA_SMS_SMSIR_TEMPLATE_ID", "123456")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if cfg.SMS.Provider != "smsir" || cfg.SMS.SMSIR.APIKey != "env-key" || cfg.SMS.SMSIR.TemplateID != 123456 {
			t.Fatalf("env overrides not applied: %+v", cfg.SMS)
		}
	})

	t.Run("invalid template id env fails", func(t *testing.T) {
		writeConfig(t, baseConfig+`
sms:
  provider: console
`)
		t.Setenv("HAMSA_SMS_SMSIR_TEMPLATE_ID", "abc")
		if _, err := Load(); err == nil || !strings.Contains(err.Error(), "HAMSA_SMS_SMSIR_TEMPLATE_ID") {
			t.Fatalf("expected invalid env error, got %v", err)
		}
	})
}
