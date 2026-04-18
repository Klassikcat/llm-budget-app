package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/config"
)

func TestBootstrapFromPathsCreatesDatabaseAndAppliesMigrations(t *testing.T) {
	t.Helper()

	root := t.TempDir()
	paths := config.Paths{
		ConfigDir:          filepath.Join(root, "config"),
		DataDir:            filepath.Join(root, "data"),
		SettingsFile:       filepath.Join(root, "config", config.SettingsFileName),
		PricesOverrideFile: filepath.Join(root, "config", config.PricesOverrideFileName),
		DatabaseFile:       filepath.Join(root, "data", config.DatabaseFileName),
	}

	store, err := BootstrapFromPaths(context.Background(), paths, Options{
		BusyTimeout:  2 * time.Second,
		Synchronous:  defaultSynchronous,
		Retry:        RetryOptions{MaxAttempts: 4, Backoff: time.Millisecond},
		MigrationsFS: defaultMigrationsFS,
	})
	if err != nil {
		fatalBootstrap(t, err)
	}
	defer store.Close()

	tables := []string{
		"schema_migrations",
		"pricing_catalog_cache",
		"subscriptions",
		"subscription_fees",
		"monthly_budgets",
		"budget_states",
		"forecast_snapshots",
		"alert_events",
		"sessions",
		"usage_entries",
		"insights",
		"watcher_offsets",
		"settings_snapshots",
	}

	for _, table := range tables {
		assertTableExists(t, store.DB(), table)
	}

	assertColumnAbsent(t, store.DB(), "pricing_catalog_cache", "raw_payload")
	assertColumnAbsent(t, store.DB(), "usage_entries", "notes")
	assertColumnAbsent(t, store.DB(), "insights", "summary")

	assertTextPragmaValue(t, store.DB(), "journal_mode", "wal")
	assertIntPragmaValue(t, store.DB(), "synchronous", 1)
	assertIntPragmaValue(t, store.DB(), "busy_timeout", 2000)

	stats := store.DB().Stats()
	if stats.MaxOpenConnections != 1 {
		t.Fatalf("MaxOpenConnections = %d, want 1", stats.MaxOpenConnections)
	}

	if err := ApplyMigrations(context.Background(), store.DB(), defaultMigrationsFS); err != nil {
		t.Fatalf("ApplyMigrations() second run error = %v", err)
	}

	var appliedCount int
	if err := store.DB().QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&appliedCount); err != nil {
		t.Fatalf("scan applied migration count: %v", err)
	}
	if appliedCount != 4 {
		t.Fatalf("applied migration count = %d, want 4", appliedCount)
	}
}

func assertTableExists(t *testing.T, db *sql.DB, table string) {
	t.Helper()

	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
		t.Fatalf("table %q missing: %v", table, err)
	}
}

func assertColumnAbsent(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()

	rows, err := db.Query("PRAGMA table_info(" + table + ");")
	if err != nil {
		t.Fatalf("query table info for %q: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal sql.NullString
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			t.Fatalf("scan table info for %q: %v", table, err)
		}
		if name == column {
			t.Fatalf("column %q unexpectedly present on table %q", column, table)
		}
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info for %q: %v", table, err)
	}
}

func assertTextPragmaValue(t *testing.T, db *sql.DB, pragma string, want string) {
	t.Helper()

	var got string
	if err := db.QueryRow("PRAGMA " + pragma + ";").Scan(&got); err != nil {
		t.Fatalf("query PRAGMA %s: %v", pragma, err)
	}
	if got != want {
		t.Fatalf("PRAGMA %s = %q, want %q", pragma, got, want)
	}
}

func assertIntPragmaValue(t *testing.T, db *sql.DB, pragma string, want int) {
	t.Helper()

	var got int
	if err := db.QueryRow("PRAGMA " + pragma + ";").Scan(&got); err != nil {
		t.Fatalf("query PRAGMA %s: %v", pragma, err)
	}
	if got != want {
		t.Fatalf("PRAGMA %s = %d, want %d", pragma, got, want)
	}
}

func fatalBootstrap(t *testing.T, err error) {
	t.Helper()
	t.Fatalf("BootstrapFromPaths() error = %v", err)
}
