package parsers

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestClaudeParserModern(t *testing.T) {
	t.Parallel()

	parser := NewClaudeCodeParser()
	content := fixtureBytes(t, "claude/current/projects/acme-app/sessions/current-session.jsonl")
	path := filepath.Join(string(filepath.Separator), "home", "tester", ".config", "claude", "projects", "acme-app", "sessions", "current-session.jsonl")

	result, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID: "claude-modern",
		Path:     path,
		Content:  content,
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Events) != 2 {
		t.Fatalf("len(result.Events) = %d, want 2", len(result.Events))
	}

	if result.NextOffset != int64(len(content)) {
		t.Fatalf("result.NextOffset = %d, want %d", result.NextOffset, len(content))
	}

	assertContainsWarning(t, result.Warnings, "skipped duplicate row")

	first := result.Events[0]
	if first.Provider != domain.ProviderClaude {
		t.Fatalf("first.Provider = %q, want %q", first.Provider, domain.ProviderClaude)
	}
	if first.Source != domain.UsageSourceCLISession {
		t.Fatalf("first.Source = %q, want %q", first.Source, domain.UsageSourceCLISession)
	}
	if first.BillingModeHint != domain.BillingModeUnknown {
		t.Fatalf("first.BillingModeHint = %q, want %q", first.BillingModeHint, domain.BillingModeUnknown)
	}
	if first.SessionID != "session-current" {
		t.Fatalf("first.SessionID = %q, want %q", first.SessionID, "session-current")
	}
	if first.ProjectName != "acme-app" {
		t.Fatalf("first.ProjectName = %q, want %q", first.ProjectName, "acme-app")
	}
	if first.AgentName != claudeCodeAgentName {
		t.Fatalf("first.AgentName = %q, want %q", first.AgentName, claudeCodeAgentName)
	}
	if first.Tokens.InputTokens != 120 || first.Tokens.OutputTokens != 45 || first.Tokens.CacheWriteTokens != 30 || first.Tokens.CacheReadTokens != 15 {
		t.Fatalf("first.Tokens = %+v, want normalized Claude usage values", first.Tokens)
	}
	if first.CostBreakdown.FlatUSD != 0.12345 || first.CostBreakdown.TotalUSD != 0.12345 {
		t.Fatalf("first.CostBreakdown = %+v, want flat cost 0.12345", first.CostBreakdown)
	}
	if first.PricingRef == nil || first.PricingRef.ModelID != "claude-3-7-sonnet-20250219" {
		t.Fatalf("first.PricingRef = %+v, want Claude model pricing ref", first.PricingRef)
	}
	if first.ObservedToolCall != 1 {
		t.Fatalf("first.ObservedToolCall = %d, want 1", first.ObservedToolCall)
	}
	if first.PrivacySafeTags["location"] != claudeLocationCurrent {
		t.Fatalf("first.PrivacySafeTags[location] = %q, want %q", first.PrivacySafeTags["location"], claudeLocationCurrent)
	}
	if first.PrivacySafeTags["speed"] != "standard" {
		t.Fatalf("first.PrivacySafeTags[speed] = %q, want %q", first.PrivacySafeTags["speed"], "standard")
	}
	if first.PrivacySafeTags["version"] != "1.2.3" {
		t.Fatalf("first.PrivacySafeTags[version] = %q, want %q", first.PrivacySafeTags["version"], "1.2.3")
	}

	second := result.Events[1]
	if second.Tokens.CacheReadTokens != 0 || second.Tokens.CacheWriteTokens != 0 {
		t.Fatalf("second.Tokens = %+v, want zero cache tokens when fields are absent", second.Tokens)
	}
	if second.PrivacySafeTags["speed"] != "fast" {
		t.Fatalf("second.PrivacySafeTags[speed] = %q, want %q", second.PrivacySafeTags["speed"], "fast")
	}
}

func TestClaudeParserLegacy(t *testing.T) {
	t.Parallel()

	parser := NewClaudeCodeParser()
	content := fixtureBytes(t, "claude/legacy/projects/legacy-app/sessions/legacy-session.jsonl")
	path := filepath.Join(string(filepath.Separator), "home", "tester", ".claude", "projects", "legacy-app", "sessions", "legacy-session.jsonl")

	result, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID: "claude-legacy",
		Path:     path,
		Content:  content,
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Events) != 1 {
		t.Fatalf("len(result.Events) = %d, want 1", len(result.Events))
	}

	event := result.Events[0]
	if event.SessionID != "legacy-session" {
		t.Fatalf("event.SessionID = %q, want %q", event.SessionID, "legacy-session")
	}
	if event.ProjectName != "legacy-app" {
		t.Fatalf("event.ProjectName = %q, want %q", event.ProjectName, "legacy-app")
	}
	if event.PrivacySafeTags["location"] != claudeLocationLegacy {
		t.Fatalf("event.PrivacySafeTags[location] = %q, want %q", event.PrivacySafeTags["location"], claudeLocationLegacy)
	}
	if event.CostBreakdown.TotalUSD != 0 {
		t.Fatalf("event.CostBreakdown.TotalUSD = %v, want 0", event.CostBreakdown.TotalUSD)
	}
	if event.Tokens.InputTokens != 40 || event.Tokens.OutputTokens != 18 || event.Tokens.CacheWriteTokens != 8 || event.Tokens.CacheReadTokens != 4 {
		t.Fatalf("event.Tokens = %+v, want normalized legacy token values", event.Tokens)
	}
	if event.PricingRef == nil || event.PricingRef.ModelID != "claude-3-5-haiku-20241022" {
		t.Fatalf("event.PricingRef = %+v, want Claude legacy model pricing ref", event.PricingRef)
	}
	if event.ObservedToolCall != 1 {
		t.Fatalf("event.ObservedToolCall = %d, want 1", event.ObservedToolCall)
	}
}

func TestClaudeParserPartialLine(t *testing.T) {
	t.Parallel()

	parser := NewClaudeCodeParser()
	base := fixtureBytes(t, "claude/current/projects/acme-app/sessions/partial-base.jsonl")
	tail := fixtureBytes(t, "claude/current/partial-tail.txt")
	tail = bytes.TrimRight(tail, "\n")
	content := append(append([]byte{}, base...), tail...)
	path := filepath.Join(string(filepath.Separator), "home", "tester", ".config", "claude", "projects", "acme-app", "sessions", "partial-session.jsonl")

	result, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID: "claude-partial",
		Path:     path,
		Content:  content,
	})
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	if len(result.Events) != 1 {
		t.Fatalf("len(result.Events) = %d, want 1", len(result.Events))
	}

	if result.NextOffset != int64(len(base)) {
		t.Fatalf("result.NextOffset = %d, want %d", result.NextOffset, len(base))
	}

	assertContainsWarning(t, result.Warnings, "skipped partial trailing line")
	if result.Events[0].SessionID != "partial-session" {
		t.Fatalf("result.Events[0].SessionID = %q, want %q", result.Events[0].SessionID, "partial-session")
	}
}

func TestClaudeParserRotationSafe(t *testing.T) {
	t.Parallel()

	parser := NewClaudeCodeParser()
	content := fixtureBytes(t, "claude/current/projects/acme-app/sessions/rotation-session.jsonl")
	path := filepath.Join(string(filepath.Separator), "home", "tester", ".config", "claude", "projects", "acme-app", "sessions", "rotation-session.jsonl")
	firstLineEnd := int64(bytes.IndexByte(content, '\n') + 1)
	if firstLineEnd <= 0 {
		t.Fatal("rotation fixture missing newline separator")
	}

	initial, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID: "claude-rotation",
		Path:     path,
		Content:  content[:firstLineEnd],
	})
	if err != nil {
		t.Fatalf("initial Parse() error = %v", err)
	}
	if len(initial.Events) != 1 {
		t.Fatalf("len(initial.Events) = %d, want 1", len(initial.Events))
	}
	if initial.NextOffset != firstLineEnd {
		t.Fatalf("initial.NextOffset = %d, want %d", initial.NextOffset, firstLineEnd)
	}

	continued, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID:    "claude-rotation",
		Path:        path,
		Content:     content,
		StartOffset: initial.NextOffset,
	})
	if err != nil {
		t.Fatalf("continued Parse() error = %v", err)
	}
	if len(continued.Events) != 1 {
		t.Fatalf("len(continued.Events) = %d, want 1", len(continued.Events))
	}
	if continued.Events[0].ExternalID != "req-rotate-2" {
		t.Fatalf("continued.Events[0].ExternalID = %q, want %q", continued.Events[0].ExternalID, "req-rotate-2")
	}

	rotated, err := parser.Parse(context.Background(), ports.ParseInput{
		SourceID:    "claude-rotation",
		Path:        path,
		Content:     content[:firstLineEnd],
		StartOffset: continued.NextOffset,
	})
	if err != nil {
		t.Fatalf("rotated Parse() error = %v", err)
	}
	if len(rotated.Events) != 1 {
		t.Fatalf("len(rotated.Events) = %d, want 1", len(rotated.Events))
	}
	if rotated.Events[0].ExternalID != "req-rotate-1" {
		t.Fatalf("rotated.Events[0].ExternalID = %q, want %q", rotated.Events[0].ExternalID, "req-rotate-1")
	}
	assertContainsWarning(t, rotated.Warnings, "start offset exceeds content length; restarting from beginning")
}

func fixtureBytes(t *testing.T, relative string) []byte {
	t.Helper()
	path := fixturePath(t, relative)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	return content
}

func fixturePath(t *testing.T, relative string) string {
	t.Helper()
	path := filepath.Join("testdata", filepath.FromSlash(relative))
	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v", path, err)
	}
	return absPath
}

func assertContainsWarning(t *testing.T, warnings []string, needle string) {
	t.Helper()
	for _, warning := range warnings {
		if strings.Contains(warning, needle) {
			return
		}
	}
	t.Fatalf("warnings %v do not contain %q", warnings, needle)
}
