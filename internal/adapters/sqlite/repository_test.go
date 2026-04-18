package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestBusyRetry(t *testing.T) {
	t.Run("succeeds after transient write lock", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "busy-retry-success.sqlite3")

		lockHolder := mustBootstrapStore(t, path, Options{
			BusyTimeout: 15 * time.Millisecond,
			Retry:       RetryOptions{MaxAttempts: 4, Backoff: 30 * time.Millisecond},
		})
		defer lockHolder.Close()

		writer := mustBootstrapStore(t, path, Options{
			BusyTimeout: 15 * time.Millisecond,
			Retry:       RetryOptions{MaxAttempts: 6, Backoff: 25 * time.Millisecond},
		})
		defer writer.Close()

		lockedTx, err := lockHolder.DB().BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatalf("BeginTx() lock holder error = %v", err)
		}
		if _, err := lockedTx.Exec(`INSERT INTO watcher_offsets (watcher_key, source_path, byte_offset, updated_at) VALUES (?, ?, ?, ?)`, "holder", "/tmp/holder", 1, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			t.Fatalf("lock holder insert error = %v", err)
		}

		releaseDone := make(chan struct{})
		go func() {
			defer close(releaseDone)
			time.Sleep(90 * time.Millisecond)
			_ = lockedTx.Commit()
		}()

		attempts := 0
		err = writer.WithTx(context.Background(), nil, func(tx *sql.Tx) error {
			attempts++
			_, execErr := tx.Exec(`INSERT INTO watcher_offsets (watcher_key, source_path, byte_offset, updated_at) VALUES (?, ?, ?, ?)`, "writer", "/tmp/writer", 2, time.Now().UTC().Format(time.RFC3339Nano))
			return execErr
		})
		<-releaseDone
		if err != nil {
			t.Fatalf("WithTx() error = %v, want nil", err)
		}
		if attempts < 2 {
			t.Fatalf("attempts = %d, want retry attempts >= 2", attempts)
		}

		var count int
		if err := writer.DB().QueryRow(`SELECT COUNT(*) FROM watcher_offsets WHERE watcher_key = 'writer'`).Scan(&count); err != nil {
			t.Fatalf("count writer row error = %v", err)
		}
		if count != 1 {
			t.Fatalf("writer row count = %d, want 1", count)
		}
	})

	t.Run("returns typed timeout when lock persists", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "busy-retry-timeout.sqlite3")

		lockHolder := mustBootstrapStore(t, path, Options{
			BusyTimeout: 10 * time.Millisecond,
			Retry:       RetryOptions{MaxAttempts: 3, Backoff: 20 * time.Millisecond},
		})
		defer lockHolder.Close()

		writer := mustBootstrapStore(t, path, Options{
			BusyTimeout: 10 * time.Millisecond,
			Retry:       RetryOptions{MaxAttempts: 3, Backoff: 20 * time.Millisecond},
		})
		defer writer.Close()

		lockedTx, err := lockHolder.DB().BeginTx(context.Background(), nil)
		if err != nil {
			t.Fatalf("BeginTx() lock holder error = %v", err)
		}
		defer lockedTx.Rollback()

		if _, err := lockedTx.Exec(`INSERT INTO watcher_offsets (watcher_key, source_path, byte_offset, updated_at) VALUES (?, ?, ?, ?)`, "timeout-holder", "/tmp/timeout-holder", 1, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			t.Fatalf("lock holder insert error = %v", err)
		}

		attempts := 0
		err = writer.WithTx(context.Background(), nil, func(tx *sql.Tx) error {
			attempts++
			_, execErr := tx.Exec(`INSERT INTO watcher_offsets (watcher_key, source_path, byte_offset, updated_at) VALUES (?, ?, ?, ?)`, "timeout-writer", "/tmp/timeout-writer", 2, time.Now().UTC().Format(time.RFC3339Nano))
			return execErr
		})
		if err == nil {
			t.Fatal("WithTx() error = nil, want busy timeout")
		}
		if !IsBusyTimeout(err) {
			t.Fatalf("WithTx() error = %v, want BusyTimeoutError", err)
		}

		var busyErr *BusyTimeoutError
		if !errors.As(err, &busyErr) {
			t.Fatalf("errors.As(%v, BusyTimeoutError) = false", err)
		}
		if busyErr.Attempts != 3 {
			t.Fatalf("busy timeout attempts = %d, want 3", busyErr.Attempts)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}
	})
}

func mustBootstrapStore(t *testing.T, path string, opts Options) *Store {
	t.Helper()

	store, err := Bootstrap(context.Background(), Options{
		Path:         path,
		BusyTimeout:  opts.BusyTimeout,
		Synchronous:  opts.Synchronous,
		MigrationsFS: defaultMigrationsFS,
		Retry:        opts.Retry,
	})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}

	return store
}
