// Package db provides the GORM PostgreSQL connection and the
// golang-migrate runner used by the API server on startup.
package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	// file source for golang-migrate (file://migrations)
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Connect opens a GORM connection to PostgreSQL using the given DSN
// (key=value form, see backend/config.example.yaml).
func Connect(dsn string, log *slog.Logger) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: slogGormLogger(log),
	})
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	sqlDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("underlying sql db: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetConnMaxIdleTime(5 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return gormDB, nil
}

// Migrate applies all pending SQL migrations from dir (e.g. "migrations"). It
// opens a short-lived dedicated connection because migrate's postgres driver
// closes the *sql.DB it is handed on Close — closing the shared application
// pool here would break every subsequent query. No-op when already current.
func Migrate(dsn, dir string) error {
	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open migrate connection: %w", err)
	}

	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		sqlDB.Close()
		return fmt.Errorf("init migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+dir, "postgres", driver)
	if err != nil {
		sqlDB.Close()
		return fmt.Errorf("init migrate (dir=%s): %w", dir, err)
	}
	defer m.Close() // closes the dedicated sqlDB too

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}

// slogGormLogger adapts slog into GORM's logger interface, warning-level by
// default so slow queries and errors surface without request-level noise.
func slogGormLogger(log *slog.Logger) logger.Interface {
	return logger.New(
		logWriter{log: log},
		logger.Config{
			SlowThreshold:             500 * time.Millisecond,
			LogLevel:                  logger.Warn,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)
}

type logWriter struct{ log *slog.Logger }

func (w logWriter) Printf(format string, args ...any) {
	w.log.Warn(fmt.Sprintf(format, args...))
}
