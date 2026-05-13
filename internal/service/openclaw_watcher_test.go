package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/parsers"
	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
	"llm-budget-tracker/internal/service"
)

func TestOpenClawWatcherEventIngestsJSONLFixture(t *testing.T) {
	store := mustOpenClawWatcherStore(t)
	normalizer := service.NewSessionNormalizerService(store, store, nil)
	root := t.TempDir()
	watcher := newOpenClawStubWatcher()

	coordinator, err := service.NewWatchCoordinator(normalizer, store, watcher, []service.WatchTarget{
		service.NewOpenClawWatchTarget(root, parsers.NewOpenClawParser()),
	})
	if err != nil {
		t.Fatalf("NewWatchCoordinator() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() {
		cancel()
		_ = coordinator.Close()
	})
	if err := coordinator.Start(ctx); err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	usagePath := filepath.Join(root, "usage.jsonl")
	if err := os.WriteFile(usagePath, []byte(openClawWatcherJSONLFixture()), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", usagePath, err)
	}
	watcher.events <- service.FileWatchEvent{Name: usagePath, Op: service.FileWatchCreate | service.FileWatchWrite}

	entries := waitForOpenClawUsageCount(t, store, 1)
	entry := entries[0]
	if entry.Provider != domain.ProviderOpenAI {
		t.Fatalf("Provider = %q, want %q", entry.Provider, domain.ProviderOpenAI)
	}
	if entry.SessionID != "openclaw-watch-session" {
		t.Fatalf("SessionID = %q, want openclaw-watch-session", entry.SessionID)
	}
	if entry.ExternalID != "openclaw-watch-request" {
		t.Fatalf("ExternalID = %q, want openclaw-watch-request", entry.ExternalID)
	}
	if entry.Tokens.InputTokens != 1000 || entry.Tokens.OutputTokens != 200 {
		t.Fatalf("Tokens = %+v, want input=1000 output=200", entry.Tokens)
	}
}

type openClawStubWatcher struct {
	events chan service.FileWatchEvent
	errors chan error
}

func newOpenClawStubWatcher() *openClawStubWatcher {
	return &openClawStubWatcher{
		events: make(chan service.FileWatchEvent, 4),
		errors: make(chan error),
	}
}

func (w *openClawStubWatcher) Add(string) error { return nil }

func (w *openClawStubWatcher) Close() error {
	close(w.events)
	close(w.errors)
	return nil
}

func (w *openClawStubWatcher) Events() <-chan service.FileWatchEvent { return w.events }

func (w *openClawStubWatcher) Errors() <-chan error { return w.errors }

func mustOpenClawWatcherStore(t *testing.T) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "openclaw-watcher.sqlite3")})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func waitForOpenClawUsageCount(t *testing.T, repo ports.UsageEntryRepository, want int) []domain.UsageEntry {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := repo.ListUsageEntries(context.Background(), ports.UsageFilter{})
		if err == nil && len(entries) == want {
			return entries
		}
		time.Sleep(25 * time.Millisecond)
	}
	entries, err := repo.ListUsageEntries(context.Background(), ports.UsageFilter{})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	t.Fatalf("usage entry count = %d, want %d", len(entries), want)
	return nil
}

func openClawWatcherJSONLFixture() string {
	return `{"timestamp":"2026-05-13T10:00:00Z","request_id":"openclaw-watch-request","session_id":"openclaw-watch-session","provider":"openai","model":"gpt-4.1","usage":{"input_tokens":1000,"output_tokens":200,"cache_read_input_tokens":0,"cache_creation_input_tokens":0},"cost_usd":0.42,"project":"watcher-fixture","billing_mode":"byok"}` + "\n"
}
