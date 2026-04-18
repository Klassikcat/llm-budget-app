package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/config"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestBillingModeFallbackParserAppliesConfiguredModeWhenParserReturnsUnknown(t *testing.T) {
	t.Parallel()

	parser := newBillingModeFallbackParser(staticSessionParser{
		name: "claude_code",
		result: ports.ParseResult{Events: []ports.SessionEvent{{
			EntryID:         "entry-1",
			SessionID:       "session-1",
			OccurredAt:      time.Date(2026, time.April, 19, 10, 0, 0, 0, time.UTC),
			Source:          domain.UsageSourceCLISession,
			Provider:        domain.ProviderClaude,
			BillingModeHint: domain.BillingModeUnknown,
		}}, Warnings: []string{"kept"}},
	}, config.BillingModeSubscription)

	result, err := parser.Parse(context.Background(), ports.ParseInput{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := result.Events[0].BillingModeHint; got != domain.BillingModeSubscription {
		t.Fatalf("BillingModeHint = %q, want %q", got, domain.BillingModeSubscription)
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != "kept" {
		t.Fatalf("Warnings = %#v, want original warnings preserved", result.Warnings)
	}
}

func TestBillingModeFallbackParserPreservesExplicitParserHint(t *testing.T) {
	t.Parallel()

	parser := newBillingModeFallbackParser(staticSessionParser{
		name: "opencode",
		result: ports.ParseResult{Events: []ports.SessionEvent{{
			EntryID:         "entry-1",
			SessionID:       "session-1",
			OccurredAt:      time.Date(2026, time.April, 19, 10, 0, 0, 0, time.UTC),
			Source:          domain.UsageSourceCLISession,
			Provider:        domain.ProviderOpenAI,
			BillingModeHint: domain.BillingModeBYOK,
		}}},
	}, config.BillingModeSubscription)

	result, err := parser.Parse(context.Background(), ports.ParseInput{})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if got := result.Events[0].BillingModeHint; got != domain.BillingModeBYOK {
		t.Fatalf("BillingModeHint = %q, want %q", got, domain.BillingModeBYOK)
	}
}

func TestDomainBillingModeMapsKnownConfigValues(t *testing.T) {
	t.Parallel()

	if got := domainBillingMode(config.BillingModeSubscription); got != domain.BillingModeSubscription {
		t.Fatalf("subscription mode = %q, want %q", got, domain.BillingModeSubscription)
	}
	if got := domainBillingMode(config.BillingModeBYOK); got != domain.BillingModeBYOK {
		t.Fatalf("byok mode = %q, want %q", got, domain.BillingModeBYOK)
	}
	if got := domainBillingMode(config.BillingMode("unexpected")); got != domain.BillingModeUnknown {
		t.Fatalf("unexpected mode = %q, want unknown", got)
	}
}

func TestDefaultWatchTargetsOnlyWrapClaudeAndGeminiParsers(t *testing.T) {
	t.Parallel()

	settings := config.DefaultSettings()
	homeDir := t.TempDir()
	chatDir := filepath.Join(homeDir, ".gemini", "tmp", "project-hash", "chats")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	fixtureBytes, err := os.ReadFile(filepath.Join("..", "adapters", "parsers", "testdata", "gemini", "supported-session.json"))
	if err != nil {
		t.Fatalf("ReadFile() fixture error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(chatDir, "session-supported.json"), fixtureBytes, 0o644); err != nil {
		t.Fatalf("WriteFile() fixture error = %v", err)
	}

	targets := defaultWatchTargets(homeDir, settings, newWarningRecorder())

	wrapped := map[string]bool{}
	for _, target := range targets {
		_, ok := target.Parser.(*billingModeFallbackParser)
		wrapped[target.ID] = ok
	}

	if !wrapped["claude_code"] {
		t.Fatalf("claude_code parser should be wrapped with billing fallback")
	}
	if !wrapped["gemini-cli"] {
		t.Fatalf("gemini-cli parser should be wrapped with billing fallback")
	}
	if wrapped["codex"] {
		t.Fatalf("codex parser should not be wrapped with billing fallback")
	}
	if wrapped["opencode"] {
		t.Fatalf("opencode parser should not be wrapped with billing fallback")
	}
}

type staticSessionParser struct {
	name   string
	result ports.ParseResult
	err    error
}

func (p staticSessionParser) ParserName() string {
	return p.name
}

func (p staticSessionParser) Parse(_ context.Context, _ ports.ParseInput) (ports.ParseResult, error) {
	return p.result, p.err
}
