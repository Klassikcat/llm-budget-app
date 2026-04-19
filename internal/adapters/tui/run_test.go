package tui

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestRunBootstrapOnlyCreatesDatabase(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	dbPath := filepath.Join(root, "data", "llmbudget.sqlite3")

	if err := run(context.Background(), []string{"--bootstrap-only", "--db", dbPath}, os.Stderr); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("Stat(%q) error = %v", dbPath, err)
	}
}
