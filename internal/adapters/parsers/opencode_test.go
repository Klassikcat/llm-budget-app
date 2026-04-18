package parsers

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestOpenCodeParserParsesDiscoveredSQLiteSchemaWithPrivacySafeOutputs(t *testing.T) {
	root := writeOpenCodeFixtureTree(t, filepath.Join("testdata", "opencode", "discovered-schema.sql"), filepath.Join("testdata", "opencode", "discovered-auth.json"))
	logPath := filepath.Join(root, "log", "2026-04-17T09-23-47.log")

	parser := NewOpenCodeParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID:    "opencode-fixture",
		Path:        logPath,
		StartOffset: 12,
		ObservedAt:  time.Date(2026, 4, 17, 9, 24, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings = %v, want none", warnings)
	}
	if len(result.Events) != 3 {
		t.Fatalf("len(result.Events) = %d, want 3", len(result.Events))
	}

	dbInfo, err := os.Stat(filepath.Join(root, "opencode.db"))
	if err != nil {
		t.Fatalf("Stat(opencode.db) error = %v", err)
	}
	if result.NextOffset != 12+dbInfo.Size() {
		t.Fatalf("result.NextOffset = %d, want %d", result.NextOffset, 12+dbInfo.Size())
	}

	first := result.Events[0]
	if first.Provider != domain.ProviderOpenRouter {
		t.Fatalf("first.Provider = %q, want %q", first.Provider, domain.ProviderOpenRouter)
	}
	if first.BillingModeHint != domain.BillingModeOpenRouter {
		t.Fatalf("first.BillingModeHint = %q, want %q", first.BillingModeHint, domain.BillingModeOpenRouter)
	}
	if first.AgentName != "explore" {
		t.Fatalf("first.AgentName = %q, want explore", first.AgentName)
	}
	if first.ProjectName != "alpha-project" {
		t.Fatalf("first.ProjectName = %q, want alpha-project", first.ProjectName)
	}
	if first.PricingRef == nil || first.PricingRef.ModelID != "google/gemini-2.5-pro" {
		t.Fatalf("first.PricingRef = %#v, want google/gemini-2.5-pro", first.PricingRef)
	}
	if first.Tokens.InputTokens != 1264 || first.Tokens.OutputTokens != 210 || first.Tokens.CacheReadTokens != 300 || first.Tokens.CacheWriteTokens != 10 {
		t.Fatalf("first.Tokens = %+v, want normalized OpenCode counters", first.Tokens)
	}
	if first.CostBreakdown.TotalUSD != 0.01234 {
		t.Fatalf("first.CostBreakdown.TotalUSD = %v, want 0.01234", first.CostBreakdown.TotalUSD)
	}
	if first.ObservedToolCall != 1 {
		t.Fatalf("first.ObservedToolCall = %d, want 1", first.ObservedToolCall)
	}
	if got := first.PrivacySafeTags["opencode_reasoning_tokens"]; got != "64" {
		t.Fatalf("first reasoning tag = %q, want 64", got)
	}

	second := result.Events[1]
	if second.Provider != domain.ProviderOpenAI {
		t.Fatalf("second.Provider = %q, want %q", second.Provider, domain.ProviderOpenAI)
	}
	if second.BillingModeHint != domain.BillingModeSubscription {
		t.Fatalf("second.BillingModeHint = %q, want %q", second.BillingModeHint, domain.BillingModeSubscription)
	}
	if second.AgentName != "Sisyphus-Junior" {
		t.Fatalf("second.AgentName = %q, want Sisyphus-Junior", second.AgentName)
	}
	if second.CostBreakdown.TotalUSD != 0 {
		t.Fatalf("second.CostBreakdown.TotalUSD = %v, want 0", second.CostBreakdown.TotalUSD)
	}

	third := result.Events[2]
	if third.Provider != domain.ProviderGemini {
		t.Fatalf("third.Provider = %q, want %q", third.Provider, domain.ProviderGemini)
	}
	if third.BillingModeHint != domain.BillingModeBYOK {
		t.Fatalf("third.BillingModeHint = %q, want %q", third.BillingModeHint, domain.BillingModeBYOK)
	}
	if third.PricingRef == nil || third.PricingRef.ModelID != "gemini-2.0-flash" {
		t.Fatalf("third.PricingRef = %#v, want gemini-2.0-flash", third.PricingRef)
	}

	assertNoSentinelLeak(t, result.Events, result.Warnings)
}

func TestOpenCodeParserSurfacesSchemaDriftButPreservesSupportedRows(t *testing.T) {
	root := writeOpenCodeFixtureTree(t, filepath.Join("testdata", "opencode", "drift-schema.sql"), filepath.Join("testdata", "opencode", "drift-auth.json"))

	parser := NewOpenCodeParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		SourceID:    "opencode-drift-fixture",
		Path:        root,
		StartOffset: 0,
		ObservedAt:  time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v", err)
	}
	if len(result.Events) != 1 {
		t.Fatalf("len(result.Events) = %d, want 1", len(result.Events))
	}
	if len(warnings) != 3 {
		t.Fatalf("len(warnings) = %d, want 3", len(warnings))
	}
	if warnings[0].Code != OpenCodeWarningSchemaDrift {
		t.Fatalf("warnings[0].Code = %q, want %q", warnings[0].Code, OpenCodeWarningSchemaDrift)
	}
	if warnings[0].Variant != "2.0.0" {
		t.Fatalf("warnings[0].Variant = %q, want 2.0.0", warnings[0].Variant)
	}
	if warnings[1].Code != OpenCodeWarningSchemaDrift {
		t.Fatalf("warnings[1].Code = %q, want %q", warnings[1].Code, OpenCodeWarningSchemaDrift)
	}
	if !strings.Contains(warnings[1].Detail, "tokens.cache") {
		t.Fatalf("warnings[1].Detail = %q, want scalar cache drift detail", warnings[1].Detail)
	}
	if warnings[2].Code != OpenCodeWarningMissingProvider {
		t.Fatalf("warnings[2].Code = %q, want %q", warnings[2].Code, OpenCodeWarningMissingProvider)
	}
	if !strings.Contains(warnings[0].String(), "schema_drift") {
		t.Fatalf("warnings[0].String() = %q, want stable typed warning text", warnings[0].String())
	}
	if len(result.Warnings) != 3 || result.Warnings[0] != warnings[0].String() || result.Warnings[1] != warnings[1].String() || result.Warnings[2] != warnings[2].String() {
		t.Fatalf("result.Warnings = %v, want stringified typed warnings", result.Warnings)
	}

	event := result.Events[0]
	if event.Provider != domain.ProviderGemini {
		t.Fatalf("event.Provider = %q, want %q", event.Provider, domain.ProviderGemini)
	}
	if event.BillingModeHint != domain.BillingModeBYOK {
		t.Fatalf("event.BillingModeHint = %q, want %q", event.BillingModeHint, domain.BillingModeBYOK)
	}
	if event.Tokens.CacheReadTokens != 88 || event.Tokens.CacheWriteTokens != 0 {
		t.Fatalf("event.Tokens = %+v, want scalar cache drift mapped into cache read only", event.Tokens)
	}
	if event.CostBreakdown.TotalUSD != 0.0042 {
		t.Fatalf("event.CostBreakdown.TotalUSD = %v, want 0.0042", event.CostBreakdown.TotalUSD)
	}
}

func TestOpenCodeParserMissingDatabaseDegradesToTypedWarning(t *testing.T) {
	root := t.TempDir()

	parser := NewOpenCodeParser()
	result, warnings, err := parser.ParseDetailed(context.Background(), ports.ParseInput{
		Path:        root,
		StartOffset: 33,
	})
	if err != nil {
		t.Fatalf("ParseDetailed() error = %v, want nil", err)
	}
	if len(result.Events) != 0 {
		t.Fatalf("len(result.Events) = %d, want 0", len(result.Events))
	}
	if result.NextOffset != 33 {
		t.Fatalf("result.NextOffset = %d, want 33", result.NextOffset)
	}
	if len(warnings) != 1 {
		t.Fatalf("len(warnings) = %d, want 1", len(warnings))
	}
	if warnings[0].Code != OpenCodeWarningMissingDatabase {
		t.Fatalf("warnings[0].Code = %q, want %q", warnings[0].Code, OpenCodeWarningMissingDatabase)
	}
}

func writeOpenCodeFixtureTree(t *testing.T, schemaPath, authPath string) string {
	t.Helper()

	root := t.TempDir()
	dbPath := filepath.Join(root, "opencode.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open(%q) error = %v", dbPath, err)
	}
	defer db.Close()

	schemaBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", schemaPath, err)
	}
	if _, err := db.Exec(string(schemaBytes)); err != nil {
		t.Fatalf("Exec(%q) error = %v", schemaPath, err)
	}

	authBytes, err := os.ReadFile(authPath)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", authPath, err)
	}
	if err := os.WriteFile(filepath.Join(root, "auth.json"), authBytes, 0o644); err != nil {
		t.Fatalf("WriteFile(auth.json) error = %v", err)
	}

	logDir := filepath.Join(root, "log")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", logDir, err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "2026-04-17T09-23-47.log"), []byte("INFO placeholder log\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(log fixture) error = %v", err)
	}

	return root
}

func assertNoSentinelLeak(t *testing.T, events []ports.SessionEvent, warnings []string) {
	t.Helper()

	const sentinel = "SENSITIVE_TRANSCRIPT_SENTINEL"
	for _, warning := range warnings {
		if strings.Contains(warning, sentinel) {
			t.Fatalf("warning leaked sentinel transcript content: %q", warning)
		}
	}

	for _, event := range events {
		for _, value := range []string{event.EntryID, event.ExternalID, event.SessionID, event.ProjectName, event.AgentName} {
			if strings.Contains(value, sentinel) {
				t.Fatalf("event leaked sentinel transcript content: %+v", event)
			}
		}
		for key, value := range event.PrivacySafeTags {
			if strings.Contains(key, sentinel) || strings.Contains(value, sentinel) {
				t.Fatalf("event.PrivacySafeTags leaked sentinel transcript content: %v", event.PrivacySafeTags)
			}
		}
	}
}
