// Hamsa — Building Management MVP (P0) API server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"hamsa/internal/auth"
	"hamsa/internal/notification"
	"hamsa/internal/platform/config"
	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
	"hamsa/internal/platform/storage"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	log := newLogger(cfg.App.Env)
	slog.SetDefault(log)

	// Database connection (fail fast if PostgreSQL is unreachable) and the
	// handle later handed to repositories/services.
	gormDB, err := db.Connect(cfg.DB.DSN, log)
	if err != nil {
		log.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	sqlDB, err := gormDB.DB()
	if err != nil {
		log.Error("failed to access database handle", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if cfg.DB.AutoMigrate {
		if err := db.Migrate(cfg.DB.DSN, "migrations"); err != nil {
			log.Error("failed to run migrations", "error", err)
			os.Exit(1)
		}
	}

	// Auth services (identity only — role/scope resolved per request).
	users := auth.NewRepository(gormDB)
	tokens := auth.NewTokenService(
		[]byte(cfg.Auth.JWTSecret),
		cfg.Auth.AccessTTL,
		cfg.Auth.RefreshTTL,
		&auth.GormRefreshStore{DB: gormDB},
		auth.RealClock{},
	)

	// Platform + cross-cutting services.
	fileStore := storage.New(cfg.Storage.Path, 0)
	notifSvc := notification.NewService(gormDB, notification.NoopNotifier{})

	router := httpx.NewRouter(log, cfg.App.Env)
	v1 := router.Group("/api/v1")

	authMW := auth.Authenticate(tokens, users)
	storage.Register(v1.Group("/files", authMW), fileStore, gormDB)
	notification.Register(v1.Group("/me/notifications", authMW), notifSvc)

	srv := &http.Server{
		Addr:              cfg.App.Addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Info("hamsa api listening", "env", cfg.App.Env, "addr", cfg.App.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown on SIGINT/SIGTERM.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("graceful shutdown failed", "error", err)
	}
	log.Info("hamsa api stopped")
}

// newLogger returns a structured logger: human-readable text in dev,
// machine-readable JSON in production.
func newLogger(env string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: slog.LevelDebug}
	if env == "production" {
		opts.Level = slog.LevelInfo
		return slog.New(slog.NewJSONHandler(os.Stdout, opts))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts))
}
