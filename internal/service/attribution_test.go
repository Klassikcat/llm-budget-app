package service

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestSessionAuthModeDetection(t *testing.T) {
	usageRepo := &captureUsageEntryRepository{}
	sessionRepo := &captureSessionRepository{}
	service := NewSessionNormalizerService(usageRepo, sessionRepo, nil)

	openAIRef, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-5-codex", "gpt-5-codex")
	if err != nil {
		t.Fatalf("NewModelPricingRef(openai) error = %v", err)
	}
	miniRef, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-5-mini", "gpt-5-mini")
	if err != nil {
		t.Fatalf("NewModelPricingRef(openai mini) error = %v", err)
	}
	geminiRef, err := domain.NewModelPricingRef(domain.ProviderGemini, "gemini-2.0-flash", "gemini-2.0-flash")
	if err != nil {
		t.Fatalf("NewModelPricingRef(gemini) error = %v", err)
	}

	result, err := service.Normalize(context.Background(), []ports.SessionEvent{
		newSessionEvent(t, "entry-sub-1", "session-sub", domain.ProviderOpenAI, domain.BillingModeSubscription, "alpha-project", "codex", &openAIRef, 1, time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC)),
		newSessionEvent(t, "entry-sub-2", "session-sub", domain.ProviderOpenAI, domain.BillingModeUnknown, "alpha-project", "codex", &openAIRef, 0, time.Date(2026, 4, 17, 9, 1, 0, 0, time.UTC)),
		newSessionEvent(t, "entry-byok-1", "session-byok", domain.ProviderOpenAI, domain.BillingModeBYOK, "beta-project", "codex", &miniRef, 2, time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC)),
		newSessionEvent(t, "entry-byok-2", "session-byok", domain.ProviderOpenAI, domain.BillingModeDirectAPI, "beta-project", "codex", &miniRef, 0, time.Date(2026, 4, 17, 10, 1, 0, 0, time.UTC)),
		newSessionEvent(t, "entry-unknown-1", "session-unknown", domain.ProviderGemini, domain.BillingModeUnknown, "gamma-project", "gemini", &geminiRef, 0, time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}

	if len(result.Sessions) != 3 {
		t.Fatalf("len(result.Sessions) = %d, want 3", len(result.Sessions))
	}
	if len(result.UsageEntries) != 5 {
		t.Fatalf("len(result.UsageEntries) = %d, want 5", len(result.UsageEntries))
	}

	sessionsByID := mapSessionsByID(result.Sessions)
	if got := sessionsByID["session-sub"].BillingMode; got != domain.BillingModeSubscription {
		t.Fatalf("session-sub BillingMode = %q, want subscription", got)
	}
	if got := sessionsByID["session-sub"].ProjectName; got != "alpha-project" {
		t.Fatalf("session-sub ProjectName = %q, want alpha-project", got)
	}
	if got := sessionsByID["session-sub"].AgentName; got != "codex" {
		t.Fatalf("session-sub AgentName = %q, want codex", got)
	}
	if sessionsByID["session-sub"].PricingRef == nil || sessionsByID["session-sub"].PricingRef.ModelID != "gpt-5-codex" {
		t.Fatalf("session-sub PricingRef = %#v, want gpt-5-codex", sessionsByID["session-sub"].PricingRef)
	}

	if got := sessionsByID["session-byok"].BillingMode; got != domain.BillingModeBYOK {
		t.Fatalf("session-byok BillingMode = %q, want byok", got)
	}
	if sessionsByID["session-byok"].PricingRef == nil || sessionsByID["session-byok"].PricingRef.ModelID != "gpt-5-mini" {
		t.Fatalf("session-byok PricingRef = %#v, want gpt-5-mini", sessionsByID["session-byok"].PricingRef)
	}

	if got := sessionsByID["session-unknown"].BillingMode; got != domain.BillingModeUnknown {
		t.Fatalf("session-unknown BillingMode = %q, want unknown", got)
	}

	entriesByID := mapUsageEntriesByID(result.UsageEntries)
	if got := entriesByID["entry-sub-1"].BillingMode; got != domain.BillingModeSubscription {
		t.Fatalf("entry-sub-1 BillingMode = %q, want subscription", got)
	}
	if got := entriesByID["entry-byok-2"].BillingMode; got != domain.BillingModeBYOK {
		t.Fatalf("entry-byok-2 BillingMode = %q, want byok after session-level canonicalization", got)
	}
	if got := entriesByID["entry-byok-1"].Metadata["mcp_tool_call_count"]; got != "2" {
		t.Fatalf("entry-byok-1 Metadata[mcp_tool_call_count] = %q, want 2", got)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("len(result.Warnings) = %d, want 1", len(result.Warnings))
	}
	if result.Warnings[0].Code != AttributionWarningBillingModeMissing {
		t.Fatalf("result.Warnings[0].Code = %q, want billing_mode_missing", result.Warnings[0].Code)
	}

	if len(usageRepo.entries) != 5 {
		t.Fatalf("capture usage repo stored %d entries, want 5", len(usageRepo.entries))
	}
	if len(sessionRepo.sessions) != 3 {
		t.Fatalf("capture session repo stored %d sessions, want 3", len(sessionRepo.sessions))
	}
}

func TestAttributionAmbiguityWarning(t *testing.T) {
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "task-16-attribution.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	service := NewSessionNormalizerService(store, store, nil)
	ref, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-5-codex", "gpt-5-codex")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	result, err := service.Normalize(context.Background(), []ports.SessionEvent{
		newSessionEvent(t, "entry-ambiguous-1", "session-ambiguous", domain.ProviderOpenAI, domain.BillingModeSubscription, "delta-project", "codex", &ref, 1, time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC)),
		newSessionEvent(t, "entry-ambiguous-2", "session-ambiguous", domain.ProviderOpenAI, domain.BillingModeBYOK, "delta-project", "codex", &ref, 0, time.Date(2026, 4, 17, 12, 1, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if len(result.Warnings) != 1 {
		t.Fatalf("len(result.Warnings) = %d, want 1", len(result.Warnings))
	}
	warning := result.Warnings[0]
	if warning.Code != AttributionWarningBillingModeConflict {
		t.Fatalf("warning.Code = %q, want billing_mode_conflict", warning.Code)
	}
	if !strings.Contains(warning.String(), "subscription,byok") && !strings.Contains(warning.String(), "byok,subscription") {
		t.Fatalf("warning.String() = %q, want conflicting mode detail", warning.String())
	}

	sessions, err := store.ListSessions(context.Background(), ports.SessionFilter{SessionID: "session-ambiguous"})
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("len(sessions) = %d, want 1", len(sessions))
	}
	if got := sessions[0].BillingMode; got != domain.BillingModeUnknown {
		t.Fatalf("stored session BillingMode = %q, want unknown", got)
	}

	entries, err := store.ListUsageEntries(context.Background(), ports.UsageFilter{SessionID: "session-ambiguous"})
	if err != nil {
		t.Fatalf("ListUsageEntries() error = %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(entries) = %d, want 2", len(entries))
	}
	if got := entries[0].Metadata["mcp_tool_call_count"]; got != "1" {
		t.Fatalf("entries[0].Metadata[mcp_tool_call_count] = %q, want 1", got)
	}
	for _, entry := range entries {
		if entry.BillingMode != domain.BillingModeUnknown {
			t.Fatalf("entry %s BillingMode = %q, want unknown", entry.EntryID, entry.BillingMode)
		}
		for key, value := range entry.Metadata {
			if strings.Contains(key, "SENSITIVE_TRANSCRIPT_SENTINEL") || strings.Contains(value, "SENSITIVE_TRANSCRIPT_SENTINEL") {
				t.Fatalf("entry metadata leaked transcript sentinel: %v", entry.Metadata)
			}
		}
	}
}

type captureUsageEntryRepository struct {
	entries []domain.UsageEntry
}

func (r *captureUsageEntryRepository) UpsertUsageEntries(_ context.Context, entries []domain.UsageEntry) error {
	r.entries = append([]domain.UsageEntry(nil), entries...)
	return nil
}

func (r *captureUsageEntryRepository) ListUsageEntries(_ context.Context, _ ports.UsageFilter) ([]domain.UsageEntry, error) {
	return append([]domain.UsageEntry(nil), r.entries...), nil
}

type captureSessionRepository struct {
	sessions []domain.SessionSummary
}

func (r *captureSessionRepository) UpsertSessions(_ context.Context, sessions []domain.SessionSummary) error {
	r.sessions = append([]domain.SessionSummary(nil), sessions...)
	return nil
}

func (r *captureSessionRepository) ListSessions(_ context.Context, _ ports.SessionFilter) ([]domain.SessionSummary, error) {
	return append([]domain.SessionSummary(nil), r.sessions...), nil
}

func newSessionEvent(t *testing.T, entryID, sessionID string, provider domain.ProviderName, billingMode domain.BillingMode, projectName, agentName string, pricingRef *domain.ModelPricingRef, observedToolCall int64, occurredAt time.Time) ports.SessionEvent {
	t.Helper()
	tokens, err := domain.NewTokenUsage(120, 30, 10, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(0.0012, 0.0008, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	return ports.SessionEvent{
		EntryID:         entryID,
		ExternalID:      entryID,
		SessionID:       sessionID,
		OccurredAt:      occurredAt,
		Source:          domain.UsageSourceCLISession,
		Provider:        provider,
		BillingModeHint: billingMode,
		ProjectName:     projectName,
		AgentName:       agentName,
		PricingRef:      pricingRef,
		Tokens:          tokens,
		CostBreakdown:   costs,
		PrivacySafeTags: map[string]string{
			"parser":              "task-16-test",
			"safe_tag":            "structured-only",
			"transcript_redacted": "content-redacted",
		},
		ObservedToolCall: observedToolCall,
	}
}

func mapSessionsByID(summaries []domain.SessionSummary) map[string]domain.SessionSummary {
	result := make(map[string]domain.SessionSummary, len(summaries))
	for _, summary := range summaries {
		result[summary.SessionID] = summary
	}
	return result
}

func mapUsageEntriesByID(entries []domain.UsageEntry) map[string]domain.UsageEntry {
	result := make(map[string]domain.UsageEntry, len(entries))
	for _, entry := range entries {
		result[entry.EntryID] = entry
	}
	return result
}
