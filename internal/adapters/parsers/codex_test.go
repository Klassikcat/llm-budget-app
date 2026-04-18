package parsers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestCodexParserNormalizesUsageAndBillingHints(t *testing.T) {
	t.Helper()

	fixturePath := filepath.Join("testdata", "codex", "session-normalized.jsonl")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", fixturePath, err)
	}

	parser := NewCodexParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID:    "codex-fixture",
		Path:        filepath.Join("/home/tester/.codex/sessions/2026/04/17", "rollout-2026-04-17T09-30-00-sample.jsonl"),
		Content:     content,
		StartOffset: 0,
		ObservedAt:  time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(result.Events) != 2 {
		t.Fatalf("len(result.Events) = %d, want 2", len(result.Events))
	}
	if result.NextOffset != int64(len(content)) {
		t.Fatalf("result.NextOffset = %d, want %d", result.NextOffset, len(content))
	}

	first := result.Events[0]
	if first.Provider != domain.ProviderCodex {
		t.Fatalf("first.Provider = %q, want %q", first.Provider, domain.ProviderCodex)
	}
	if first.BillingModeHint != domain.BillingModeSubscription {
		t.Fatalf("first.BillingModeHint = %q, want %q", first.BillingModeHint, domain.BillingModeSubscription)
	}
	if first.ProjectName != "sample-project" {
		t.Fatalf("first.ProjectName = %q, want sample-project", first.ProjectName)
	}
	if first.SessionID != "sess_codex_demo_1" {
		t.Fatalf("first.SessionID = %q, want sess_codex_demo_1", first.SessionID)
	}
	if first.PricingRef == nil || first.PricingRef.ModelID != "gpt-5-codex" {
		t.Fatalf("first.PricingRef = %#v, want model gpt-5-codex", first.PricingRef)
	}
	if first.Tokens.InputTokens != 1200 || first.Tokens.OutputTokens != 250 || first.Tokens.CacheReadTokens != 500 || first.Tokens.CacheWriteTokens != 100 {
		t.Fatalf("first.Tokens = %+v, want normalized token counters", first.Tokens)
	}
	if first.ObservedToolCall != 1 {
		t.Fatalf("first.ObservedToolCall = %d, want 1", first.ObservedToolCall)
	}
	if got := first.PrivacySafeTags["codex_usage_shape"]; got != "payload.usage" {
		t.Fatalf("first codex_usage_shape = %q, want payload.usage", got)
	}

	second := result.Events[1]
	if second.BillingModeHint != domain.BillingModeBYOK {
		t.Fatalf("second.BillingModeHint = %q, want %q", second.BillingModeHint, domain.BillingModeBYOK)
	}
	if second.PricingRef == nil || second.PricingRef.ModelID != "gpt-5-mini" {
		t.Fatalf("second.PricingRef = %#v, want model gpt-5-mini", second.PricingRef)
	}
	if second.Tokens.InputTokens != 300 || second.Tokens.OutputTokens != 80 || second.Tokens.CacheReadTokens != 40 {
		t.Fatalf("second.Tokens = %+v, want normalized nested usage counters", second.Tokens)
	}
	if second.CostBreakdown.TotalUSD != 0.0125 {
		t.Fatalf("second.CostBreakdown.TotalUSD = %v, want 0.0125", second.CostBreakdown.TotalUSD)
	}
	if got := second.PrivacySafeTags["codex_usage_shape"]; got != "payload.item.usage" {
		t.Fatalf("second codex_usage_shape = %q, want payload.item.usage", got)
	}
	if !strings.Contains(second.OccurredAt.Format(time.RFC3339), "T09:31:15Z") {
		t.Fatalf("second.OccurredAt = %s, want UTC 2026-04-17T09:31:15Z", second.OccurredAt.Format(time.RFC3339Nano))
	}
}

func TestCodexParserUnknownVariantProducesTypedWarning(t *testing.T) {
	t.Helper()

	fixturePath := filepath.Join("testdata", "codex", "session-unknown-variant.jsonl")
	content, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", fixturePath, err)
	}

	parser := NewCodexParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID:    "codex-unknown-fixture",
		Path:        filepath.Join("/home/tester/.codex/sessions/2026/04/17", "rollout-2026-04-17T10-00-00-unknown.jsonl"),
		Content:     content,
		StartOffset: 25,
		ObservedAt:  time.Date(2026, 4, 17, 10, 5, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("len(result.Events) = %d, want 1", len(result.Events))
	}
	if result.NextOffset != int64(25+len(content)) {
		t.Fatalf("result.NextOffset = %d, want %d", result.NextOffset, 25+len(content))
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}

	warning := warnings[0]
	if warning.Code != CodexWarningUnsupportedVariant {
		t.Fatalf("warning.Code = %q, want %q", warning.Code, CodexWarningUnsupportedVariant)
	}
	if warning.Variant != "mystery_event" {
		t.Fatalf("warning.Variant = %q, want mystery_event", warning.Variant)
	}
	if warning.Line != 2 {
		t.Fatalf("warning.Line = %d, want 2", warning.Line)
	}
	if !strings.Contains(warning.String(), "unsupported_variant") || !strings.Contains(warning.String(), "mystery_event") {
		t.Fatalf("warning.String() = %q, want stable typed warning text", warning.String())
	}
	if len(result.Warnings) != 1 || result.Warnings[0] != warning.String() {
		t.Fatalf("result.Warnings = %v, want stringified typed warning", result.Warnings)
	}
	if result.Events[0].BillingModeHint != domain.BillingModeSubscription {
		t.Fatalf("result.Events[0].BillingModeHint = %q, want subscription preserved", result.Events[0].BillingModeHint)
	}
}
