package service_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adapterfsnotify "llm-budget-tracker/internal/adapters/fsnotify"
	"llm-budget-tracker/internal/adapters/parsers"
	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestWatcherImportsNewLinesOnce(t *testing.T) {
	store := mustWatcherStore(t)
	normalizer := service.NewSessionNormalizerService(store, store, nil)

	root := t.TempDir()
	sessionDir := filepath.Join(root, "projects", "acme-app", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", sessionDir, err)
	}

	fixture := watcherFixtureLines(t)
	logPath := filepath.Join(sessionDir, "rotation-session.jsonl")
	if err := os.WriteFile(logPath, []byte(fixture[0]+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", logPath, err)
	}

	startCoordinator(t, context.Background(), normalizer, store, service.NewClaudeWatchTarget(root, parsers.NewClaudeCodeParser()))
	waitForUsageCount(t, store, 1)

	appendLine(t, logPath, fixture[1])
	waitForUsageCount(t, store, 2)
	assertSessionCount(t, store, 1)

	startCoordinator(t, context.Background(), normalizer, store, service.NewClaudeWatchTarget(root, parsers.NewClaudeCodeParser()))
	assertUsageCount(t, store, 2)
	assertSessionCount(t, store, 1)

	checkpoint, err := store.LoadCheckpoint(context.Background(), sourceID(parsers.ClaudeCodeParserName, logPath))
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if checkpoint.Path != logPath {
		t.Fatalf("checkpoint.Path = %q, want %q", checkpoint.Path, logPath)
	}
	if checkpoint.LastMarker != "req-rotate-2" {
		t.Fatalf("checkpoint.LastMarker = %q, want req-rotate-2", checkpoint.LastMarker)
	}
	if checkpoint.Offset <= 0 {
		t.Fatalf("checkpoint.Offset = %d, want > 0", checkpoint.Offset)
	}
}

func TestWatcherHandlesRotationAndTruncation(t *testing.T) {
	store := mustWatcherStore(t)
	normalizer := service.NewSessionNormalizerService(store, store, nil)

	root := t.TempDir()
	sessionDir := filepath.Join(root, "projects", "acme-app", "sessions")
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", sessionDir, err)
	}

	fixture := watcherFixtureLines(t)
	newLine := strings.ReplaceAll(fixture[1], "req-rotate-2", "req-rotate-3")
	newLine = strings.ReplaceAll(newLine, "msg-rotate-2", "msg-rotate-3")
	newLine = strings.ReplaceAll(newLine, "2026-04-16T12:03:00Z", "2026-04-16T12:06:00Z")

	logPath := filepath.Join(sessionDir, "rotation-session.jsonl")
	if err := os.WriteFile(logPath, []byte(fixture[0]+"\n"+fixture[1]+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", logPath, err)
	}

	startCoordinator(t, context.Background(), normalizer, store, service.NewClaudeWatchTarget(root, parsers.NewClaudeCodeParser()))
	waitForUsageCount(t, store, 2)

	beforeRotation, err := store.LoadCheckpoint(context.Background(), sourceID(parsers.ClaudeCodeParserName, logPath))
	if err != nil {
		t.Fatalf("LoadCheckpoint(before rotation) error = %v", err)
	}

	rotatedPath := logPath + ".1"
	if err := os.Rename(logPath, rotatedPath); err != nil {
		t.Fatalf("Rename(%q, %q) error = %v", logPath, rotatedPath, err)
	}
	if err := os.WriteFile(logPath, []byte(fixture[0]+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(rotated %q) error = %v", logPath, err)
	}
	waitForCheckpoint(t, store, sourceID(parsers.ClaudeCodeParserName, logPath), func(checkpoint ports.IngestionCheckpoint) bool {
		return checkpoint.LastMarker == "req-rotate-1" && checkpoint.FileIdentity != beforeRotation.FileIdentity
	})
	waitForUsageCount(t, store, 2)

	if err := os.WriteFile(logPath, []byte(newLine+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(truncated %q) error = %v", logPath, err)
	}
	waitForCheckpoint(t, store, sourceID(parsers.ClaudeCodeParserName, logPath), func(checkpoint ports.IngestionCheckpoint) bool {
		return checkpoint.LastMarker == "req-rotate-3"
	})
	waitForUsageCount(t, store, 3)
	assertSessionCount(t, store, 1)

	afterRotation, err := store.LoadCheckpoint(context.Background(), sourceID(parsers.ClaudeCodeParserName, logPath))
	if err != nil {
		t.Fatalf("LoadCheckpoint(after rotation) error = %v", err)
	}
	if afterRotation.LastMarker != "req-rotate-3" {
		t.Fatalf("checkpoint.LastMarker = %q, want req-rotate-3", afterRotation.LastMarker)
	}
	if afterRotation.FileIdentity == beforeRotation.FileIdentity {
		t.Fatalf("checkpoint.FileIdentity = %q, want a new identity after rotation/truncation", afterRotation.FileIdentity)
	}
	if afterRotation.Offset <= 0 {
		t.Fatalf("checkpoint.Offset = %d, want > 0", afterRotation.Offset)
	}
}

func startCoordinator(t *testing.T, ctx context.Context, normalizer *service.SessionNormalizerService, checkpoints ports.CheckpointRepository, target service.WatchTarget) {
	t.Helper()

	watcher, err := adapterfsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("NewWatcher() error = %v", err)
	}
	coordinator, err := service.NewWatchCoordinator(normalizer, checkpoints, watcher, []service.WatchTarget{target})
	if err != nil {
		t.Fatalf("NewWatchCoordinator() error = %v", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(func() {
		cancel()
		_ = coordinator.Close()
	})

	if err := coordinator.Start(runCtx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
}

func sourceID(parserName, path string) string {
	return parserName + ":" + filepath.Clean(path)
}

func mustWatcherStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "watcher.sqlite3")})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func watcherFixtureLines(t *testing.T) []string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "adapters", "parsers", "testdata", "claude", "current", "projects", "acme-app", "sessions", "rotation-session.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile(rotation fixture) error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != 2 {
		t.Fatalf("rotation fixture line count = %d, want 2", len(lines))
	}
	return lines
}

func appendLine(t *testing.T, path, line string) {
	t.Helper()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("OpenFile(%q) error = %v", path, err)
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		t.Fatalf("WriteString(%q) error = %v", path, err)
	}
}

func waitForUsageCount(t *testing.T, repo ports.UsageEntryRepository, want int) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := repo.ListUsageEntries(context.Background(), ports.UsageFilter{})
		if err == nil && len(entries) == want {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	entries, err := repo.ListUsageEntries(context.Background(), ports.UsageFilter{})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	t.Fatalf("usage entry count = %d, want %d", len(entries), want)
}

func waitForCheckpoint(t *testing.T, repo ports.CheckpointRepository, source string, predicate func(ports.IngestionCheckpoint) bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		checkpoint, err := repo.LoadCheckpoint(context.Background(), source)
		if err == nil && predicate(checkpoint) {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}
	checkpoint, err := repo.LoadCheckpoint(context.Background(), source)
	if err != nil {
		t.Fatalf("LoadCheckpoint(%q) error = %v", source, err)
	}
	t.Fatalf("checkpoint = %+v, predicate not satisfied", checkpoint)
}

func assertUsageCount(t *testing.T, repo ports.UsageEntryRepository, want int) {
	t.Helper()
	entries, err := repo.ListUsageEntries(context.Background(), ports.UsageFilter{})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != want {
		t.Fatalf("usage entry count = %d, want %d", len(entries), want)
	}
}

func assertSessionCount(t *testing.T, repo ports.SessionRepository, want int) {
	t.Helper()
	sessions, err := repo.ListSessions(context.Background(), ports.SessionFilter{})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != want {
		t.Fatalf("session count = %d, want %d", len(sessions), want)
	}
}
