// Package main is the standalone migration runner: applies all pending SQL
// migrations from ./migrations against the configured database and exits.
// The API server applies migrations itself only when db.auto_migrate is set;
// deployments that manage schema changes out-of-band run this instead.
package main

import (
	"log/slog"
	"os"

	"hamsa/internal/platform/config"
	"hamsa/internal/platform/db"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}
	if err := db.Migrate(cfg.DB.DSN, "migrations"); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied", "dsn_host_db", cfg.DB.DSN)
}
