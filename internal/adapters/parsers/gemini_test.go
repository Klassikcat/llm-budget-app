package parsers

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestProbeGeminiCLIPathReportsInactiveForMissingDirectory(t *testing.T) {
	missingRoot := filepath.Join(t.TempDir(), ".gemini")

	status := ProbeGeminiCLIPath(missingRoot)

	if status.Provider != domain.ProviderGemini {
		t.Fatalf("Provider = %q, want %q", status.Provider, domain.ProviderGemini)
	}
	if status.State != GeminiStateInactive {
		t.Fatalf("State = %q, want %q", status.State, GeminiStateInactive)
	}
	if status.Message == "" {
		t.Fatal("Message is empty")
	}
}

func TestProbeGeminiCLIPathReportsSupportedForChatSessionFixtureTree(t *testing.T) {
	root := t.TempDir()
	chatDir := filepath.Join(root, "tmp", "project-hash", "chats")
	if err := os.MkdirAll(chatDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	fixtureBytes, err := os.ReadFile(filepath.Join("testdata", "gemini", "supported-session.json"))
	if err != nil {
		t.Fatalf("ReadFile() fixture error = %v", err)
	}

	fixturePath := filepath.Join(chatDir, "session-2026-01-23T08-50-45548acb.json")
	if err := os.WriteFile(fixturePath, fixtureBytes, 0o644); err != nil {
		t.Fatalf("WriteFile() fixture error = %v", err)
	}

	status := ProbeGeminiCLIPath(root)

	if status.State != GeminiStateSupported {
		t.Fatalf("State = %q, want %q", status.State, GeminiStateSupported)
	}
}

func TestGeminiCLIParserParsesSupportedSessionFixture(t *testing.T) {
	fixtureBytes, err := os.ReadFile(filepath.Join("testdata", "gemini", "supported-session.json"))
	if err != nil {
		t.Fatalf("ReadFile() fixture error = %v", err)
	}

	parser := NewGeminiCLIParser()
	result, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID:    "gemini-fixture",
		Path:        filepath.Join("tmp", "project-hash", "chats", "session-supported.json"),
		Content:     fixtureBytes,
		StartOffset: 0,
		ObservedAt:  time.Date(2026, time.January, 23, 8, 51, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Events) != 1 {
		t.Fatalf("len(Events) = %d, want 1", len(result.Events))
	}

	event := result.Events[0]
	if event.Provider != domain.ProviderGemini {
		t.Fatalf("Provider = %q, want %q", event.Provider, domain.ProviderGemini)
	}
	if event.Source != domain.UsageSourceCLISession {
		t.Fatalf("Source = %q, want %q", event.Source, domain.UsageSourceCLISession)
	}
	if event.BillingModeHint != domain.BillingModeUnknown {
		t.Fatalf("BillingModeHint = %q, want %q", event.BillingModeHint, domain.BillingModeUnknown)
	}
	if event.SessionID != "45548acb-a712-4740-a867-cdc9f3ce994f" {
		t.Fatalf("SessionID = %q", event.SessionID)
	}
	if event.ExternalID != "df40687d-2eed-4c9f-8fc7-a0944809dbf5" {
		t.Fatalf("ExternalID = %q", event.ExternalID)
	}
	if event.AgentName != geminiParserName {
		t.Fatalf("AgentName = %q, want %q", event.AgentName, geminiParserName)
	}
	if event.PricingRef == nil {
		t.Fatal("PricingRef is nil")
	}
	if event.PricingRef.ModelID != "gemini-3-flash-preview" {
		t.Fatalf("PricingRef.ModelID = %q", event.PricingRef.ModelID)
	}
	if event.Tokens.InputTokens != 24312 {
		t.Fatalf("InputTokens = %d, want 24312", event.Tokens.InputTokens)
	}
	if event.Tokens.OutputTokens != 193 {
		t.Fatalf("OutputTokens = %d, want 193", event.Tokens.OutputTokens)
	}
	if event.Tokens.CacheReadTokens != 0 {
		t.Fatalf("CacheReadTokens = %d, want 0", event.Tokens.CacheReadTokens)
	}
	if event.Tokens.TotalTokens != 24505 {
		t.Fatalf("TotalTokens = %d, want 24505", event.Tokens.TotalTokens)
	}
	if got := event.PrivacySafeTags["project_hash"]; got != "a2ae60f67948c12452528f9fec6b31af78312838da2295436abc1edf0b79b314" {
		t.Fatalf("project_hash tag = %q", got)
	}
	if got := event.PrivacySafeTags["gemini_thought_tokens"]; got != "111" {
		t.Fatalf("gemini_thought_tokens tag = %q, want 111", got)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("len(Warnings) = %d, want 1", len(result.Warnings))
	}
	if result.NextOffset != int64(len(fixtureBytes)) {
		t.Fatalf("NextOffset = %d, want %d", result.NextOffset, len(fixtureBytes))
	}
}

func TestGeminiCLIParserReturnsTypedUnsupportedErrorForLogsJSON(t *testing.T) {
	fixtureBytes, err := os.ReadFile(filepath.Join("testdata", "gemini", "unsupported-logs.json"))
	if err != nil {
		t.Fatalf("ReadFile() fixture error = %v", err)
	}

	parser := NewGeminiCLIParser()
	result, err := parser.Parse(context.Background(), ports.ParseInput{
		Path:        filepath.Join("tmp", "project-hash", "logs.json"),
		Content:     fixtureBytes,
		StartOffset: 99,
	})
	if err == nil {
		t.Fatal("Parse() error = nil, want typed unsupported error")
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(Events) = %d, want 0", len(result.Events))
	}
	if result.NextOffset != 99 {
		t.Fatalf("NextOffset = %d, want 99", result.NextOffset)
	}
	if !IsGeminiState(err, GeminiStateUnsupported) {
		t.Fatalf("IsGeminiState(err, unsupported) = false; err = %v", err)
	}

	var unsupportedErr *GeminiUnsupportedError
	if !errors.As(err, &unsupportedErr) {
		t.Fatalf("errors.As(%v, *GeminiUnsupportedError) = false", err)
	}
	if unsupportedErr.Status.Provider != domain.ProviderGemini {
		t.Fatalf("Provider = %q, want %q", unsupportedErr.Status.Provider, domain.ProviderGemini)
	}
	if unsupportedErr.Status.Message == "" {
		t.Fatal("unsupported status message is empty")
	}
}

func TestGeminiCLIParserDoesNotPanicOnEmptyContent(t *testing.T) {
	parser := NewGeminiCLIParser()

	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("Parse() panicked: %v", recovered)
		}
	}()

	result, err := parser.Parse(context.Background(), ports.ParseInput{
		Path:        filepath.Join("tmp", "project-hash", "chats", "session-empty.json"),
		StartOffset: 7,
	})
	if err == nil {
		t.Fatal("Parse() error = nil, want typed unsupported error")
	}
	if result.NextOffset != 7 {
		t.Fatalf("NextOffset = %d, want 7", result.NextOffset)
	}
	if !IsGeminiState(err, GeminiStateUnsupported) {
		t.Fatalf("IsGeminiState(err, unsupported) = false; err = %v", err)
	}
}
