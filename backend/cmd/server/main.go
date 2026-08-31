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

	"github.com/google/uuid"

	"hamsa/internal/announcement"
	"hamsa/internal/audit"
	"hamsa/internal/auth"
	"hamsa/internal/billing"
	"hamsa/internal/building"
	"hamsa/internal/expense"
	"hamsa/internal/maintenance"
	"hamsa/internal/notification"
	"hamsa/internal/payment"
	"hamsa/internal/payment/gateway"
	"hamsa/internal/platform/config"
	"hamsa/internal/platform/db"
	"hamsa/internal/platform/httpx"
	"hamsa/internal/platform/sms"
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
	auditSvc := audit.New(gormDB, log)

	router := httpx.NewRouter(log, cfg.App.Env)
	v1 := router.Group("/api/v1")

	authMW := auth.Authenticate(tokens, users)
	auth.Register(v1.Group("/auth"), &auth.Handler{
		OTP: auth.NewOTPService(
			&auth.GormOTPStore{DB: gormDB},
			sms.New(cfg.SMS.Provider, cfg.SMS.Kavenegar.APIKey, cfg.SMS.Kavenegar.Sender, log),
			auth.RealClock{},
			cfg.App.IsDev(),
		),
		Tokens:  tokens,
		Users:   users,
		Scopes:  auth.NewScopeResolver(gormDB),
		Auditor: auditSvc, // user.login audit entries (FR-038, T025)
	})
	storage.Register(v1.Group("/files", authMW), fileStore, gormDB)
	notification.Register(v1.Group("/me/notifications", authMW), notifSvc)

	// US2: buildings & units registry (manager scope enforced per request).
	buildingRepo := building.NewRepository(gormDB)
	building.Register(v1.Group("", authMW, auth.RequireRole(auth.RoleManager)),
		building.NewService(buildingRepo), auditSvc)

	// US4: billing — periods, cost items, charge engine, invoices. Manager
	// routes enforce the role per handler; /me/invoices serves residents and
	// scopes to their own units via the resolver.
	billingRepo := billing.NewRepository(gormDB)

	// US5: payments & balances — the balance service feeds both the
	// calculation snapshots (prior debt / credit) and the live unit_balances
	// rows recomputed on every ledger event.
	payRepo := payment.NewRepository(gormDB)
	balanceSvc := payment.NewBalanceService(payRepo)
	var payGW gateway.PaymentGateway
	if cfg.Payment.Provider == "zarinpal" {
		payGW = gateway.NewZarinpal(cfg.Payment.Zarinpal.MerchantID, cfg.Payment.Zarinpal.Sandbox)
	} else {
		payGW = gateway.NewMock() // dev default
	}
	paymentSvc := payment.NewPaymentService(payRepo, payGW, balanceSvc, notifSvc, log,
		cfg.App.BaseURL+"/api/v1/payments/callback")

	periodSvc := billing.NewPeriodService(billingRepo, notifSvc, auditSvc)
	periodSvc.SetBalanceRecomputer(balanceSvc) // US5: keep unit_balances current on issue/cancel/adjust
	billing.Register(v1.Group("", authMW),
		billing.NewCalcService(billingRepo, balanceSnapshot{balanceSvc}),
		periodSvc,
		auditSvc,
		auth.NewScopeResolver(gormDB))

	// US5 routes: the gateway callback is public (the gateway cannot present
	// a bearer token); everything else is authenticated.
	payment.Register(v1.Group("", authMW), v1.Group(""), paymentSvc, balanceSvc, auditSvc,
		auth.NewScopeResolver(gormDB))

	// US6: expenses & financial report (manager-only, audited).
	expense.Register(v1.Group("", authMW), expense.NewService(expense.NewRepository(gormDB)), auditSvc)

	// US7: maintenance requests — resident submit/track + manager workflow (notifications per transition, audited).
	maintenance.Register(v1.Group("", authMW), maintenance.NewService(maintenance.NewRepository(gormDB), notifSvc, auditSvc), auditSvc)

	// US8: announcements with audience targeting (manager publish, resident targeted list + read tracking).
	announcement.Register(v1.Group("", authMW), announcement.NewService(announcement.NewRepository(gormDB), notifSvc, auditSvc))

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

// balanceSnapshot adapts the payment balance service to billing's
// BalanceProvider: the unit's outstanding (balance) and available credit
// become the new period's prior_debt / credit_amount snapshots.
type balanceSnapshot struct {
	svc *payment.BalanceService
}

func (b balanceSnapshot) Snapshot(ctx context.Context, unitID uuid.UUID) (billing.Money, billing.Money, error) {
	pd, cr, err := b.svc.Snapshot(ctx, unitID)
	return billing.Money(pd), billing.Money(cr), err
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
