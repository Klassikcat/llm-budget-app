package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"llm-budget-tracker/db/migrations"
	"llm-budget-tracker/internal/config"

	_ "modernc.org/sqlite"
)

const (
	defaultJournalMode = "WAL"
	defaultSynchronous = "NORMAL"
)

var defaultMigrationsFS fs.FS = migrations.Files

type Options struct {
	Path         string
	BusyTimeout  time.Duration
	Synchronous  string
	MigrationsFS fs.FS
	Retry        RetryOptions
}

type RetryOptions struct {
	MaxAttempts int
	Backoff     time.Duration
}

type Store struct {
	db    *sql.DB
	retry RetryOptions
}

func Bootstrap(ctx context.Context, opts Options) (*Store, error) {
	resolved, err := opts.withDefaults()
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(resolved.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := sql.Open("sqlite", resolved.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database %q: %w", resolved.Path, err)
	}

	configured := false
	defer func() {
		if !configured {
			_ = db.Close()
		}
	}()

	configureConnectionPool(db)

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping sqlite database %q: %w", resolved.Path, err)
	}

	if err := applyPragmas(ctx, db, resolved); err != nil {
		return nil, err
	}

	if err := ApplyMigrations(ctx, db, resolved.MigrationsFS); err != nil {
		return nil, err
	}

	configured = true

	return &Store{db: db, retry: resolved.Retry}, nil
}

func BootstrapFromPaths(ctx context.Context, paths config.Paths, opts Options) (*Store, error) {
	opts.Path = paths.DatabaseFile
	return Bootstrap(ctx, opts)
}

func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}

	return s.db
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (o Options) withDefaults() (Options, error) {
	if o.Path == "" {
		return Options{}, fmt.Errorf("sqlite database path is required")
	}

	if o.BusyTimeout <= 0 {
		o.BusyTimeout = 5 * time.Second
	}

	if o.Synchronous == "" {
		o.Synchronous = defaultSynchronous
	}

	if o.MigrationsFS == nil {
		o.MigrationsFS = defaultMigrationsFS
	}

	if o.Retry.MaxAttempts <= 0 {
		o.Retry.MaxAttempts = 5
	}

	if o.Retry.Backoff <= 0 {
		o.Retry.Backoff = 25 * time.Millisecond
	}

	return o, nil
}

func configureConnectionPool(db *sql.DB) {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxIdleTime(0)
	db.SetConnMaxLifetime(0)
}

func applyPragmas(ctx context.Context, db *sql.DB, opts Options) error {
	statements := []string{
		fmt.Sprintf("PRAGMA journal_mode = %s;", defaultJournalMode),
		fmt.Sprintf("PRAGMA busy_timeout = %d;", opts.BusyTimeout.Milliseconds()),
		fmt.Sprintf("PRAGMA synchronous = %s;", opts.Synchronous),
		"PRAGMA foreign_keys = ON;",
	}

	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply sqlite pragma %q: %w", statement, err)
		}
	}

	return nil
}
