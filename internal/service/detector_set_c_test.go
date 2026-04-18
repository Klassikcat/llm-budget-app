package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/catalog"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestOverqualifiedModelDetector(t *testing.T) {
	t.Parallel()

	priceCatalog, err := catalog.New(catalog.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	sessions := []domain.SessionSummary{
		fixtureSessionSummary(t, "session-overqualified", domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1", time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC), 6000, 120, 0.01296),
		fixtureSessionSummary(t, "session-benign", domain.ProviderOpenAI, "gpt-4.1-nano", "openai/gpt-4.1-nano", time.Date(2026, 4, 17, 9, 30, 0, 0, time.UTC), 4200, 700, 0.0007),
	}
	usageEntries := []domain.UsageEntry{
		fixtureUsageEntry(t, "entry-overqualified", "session-overqualified", domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1", time.Date(2026, 4, 17, 9, 0, 30, 0, time.UTC), 6000, 120, map[string]string{
			metadataObservedToolCallCountKey: "0",
		}),
		fixtureUsageEntry(t, "entry-benign", "session-benign", domain.ProviderOpenAI, "gpt-4.1-nano", "openai/gpt-4.1-nano", time.Date(2026, 4, 17, 9, 31, 0, 0, time.UTC), 4200, 700, map[string]string{
			metadataObservedToolCallCountKey: "3",
		}),
	}

	insightRepo := &captureInsightRepoC{}
	executor := NewInsightExecutorService(
		[]ports.InsightDetector{NewOverQualifiedModelDetector(priceCatalog)},
		&fixtureSessionRepository{sessions: sessions},
		&fixtureUsageRepository{entries: usageEntries},
		insightRepo,
	)

	result, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Insights) != 1 {
		t.Fatalf("len(result.Insights) = %d, want 1", len(result.Insights))
	}
	if len(insightRepo.insights) != 1 {
		t.Fatalf("len(insightRepo.insights) = %d, want 1 persisted insight", len(insightRepo.insights))
	}

	insight := result.Insights[0]
	if insight.Category != domain.DetectorOverQualifiedModel {
		t.Fatalf("insight.Category = %q, want %q", insight.Category, domain.DetectorOverQualifiedModel)
	}
	if got := insightHashValue(t, insight.Payload.Hashes, "current_model"); got != "gpt-4.1" {
		t.Fatalf("current_model hash = %q, want gpt-4.1", got)
	}
	if got := insightHashValue(t, insight.Payload.Hashes, "recommended_model"); got != "gpt-4.1-nano" {
		t.Fatalf("recommended_model hash = %q, want gpt-4.1-nano", got)
	}
	if got := insightMetricValueTask19(t, insight.Payload.Metrics, "estimated_waste_usd"); got <= 0.01 {
		t.Fatalf("estimated_waste_usd = %v, want > 0.01", got)
	}
	if got := insightMetricValueTask19(t, insight.Payload.Metrics, "output_tokens"); got != 120 {
		t.Fatalf("output_tokens metric = %v, want 120", got)
	}
	if got := insightCountValue(t, insight.Payload.Counts, "trivial_task_hints"); got < 3 {
		t.Fatalf("trivial_task_hints = %d, want at least 3", got)
	}
	assertInsightPayloadPrivacySafe(t, insight.Payload)
}

func TestToolSchemaBloatDetector(t *testing.T) {
	t.Parallel()

	priceCatalog, err := catalog.New(catalog.Options{})
	if err != nil {
		t.Fatalf("catalog.New() error = %v", err)
	}

	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	sessions := []domain.SessionSummary{
		fixtureSessionSummary(t, "session-schema-bloat", domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1", time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC), 24000, 200, 0.0496),
		fixtureSessionSummary(t, "session-small-tooling", domain.ProviderOpenAI, "gpt-4.1-mini", "openai/gpt-4.1-mini", time.Date(2026, 4, 17, 11, 30, 0, 0, time.UTC), 4000, 600, 0.002),
	}
	usageEntries := []domain.UsageEntry{
		fixtureUsageEntry(t, "entry-schema-1", "session-schema-bloat", domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1", time.Date(2026, 4, 17, 11, 0, 10, 0, time.UTC), 8000, 80, map[string]string{
			metadataToolSchemaBytesKey:       "24000",
			metadataToolDefinitionCountKey:   "8",
			metadataToolSchemaOccurrencesKey: "1",
			metadataToolCallCountKey:         "1",
		}),
		fixtureUsageEntry(t, "entry-schema-2", "session-schema-bloat", domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1", time.Date(2026, 4, 17, 11, 0, 40, 0, time.UTC), 8000, 70, map[string]string{
			metadataToolSchemaBytesKey:       "24000",
			metadataToolDefinitionCountKey:   "8",
			metadataToolSchemaOccurrencesKey: "1",
			metadataToolCallCountKey:         "0",
		}),
		fixtureUsageEntry(t, "entry-schema-3", "session-schema-bloat", domain.ProviderOpenAI, "gpt-4.1", "openai/gpt-4.1", time.Date(2026, 4, 17, 11, 1, 10, 0, time.UTC), 8000, 50, map[string]string{
			metadataToolSchemaBytesKey:       "24000",
			metadataToolDefinitionCountKey:   "8",
			metadataToolSchemaOccurrencesKey: "1",
			metadataToolCallCountKey:         "0",
		}),
		fixtureUsageEntry(t, "entry-small-tooling", "session-small-tooling", domain.ProviderOpenAI, "gpt-4.1-mini", "openai/gpt-4.1-mini", time.Date(2026, 4, 17, 11, 30, 15, 0, time.UTC), 4000, 600, map[string]string{
			metadataToolSchemaBytesKey:       "512",
			metadataToolDefinitionCountKey:   "2",
			metadataToolSchemaOccurrencesKey: "1",
			metadataToolCallCountKey:         "2",
		}),
	}

	insightRepo := &captureInsightRepoC{}
	executor := NewInsightExecutorService(
		[]ports.InsightDetector{NewToolSchemaBloatDetector(priceCatalog)},
		&fixtureSessionRepository{sessions: sessions},
		&fixtureUsageRepository{entries: usageEntries},
		insightRepo,
	)

	result, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Insights) != 1 {
		t.Fatalf("len(result.Insights) = %d, want 1", len(result.Insights))
	}
	if len(insightRepo.insights) != 1 {
		t.Fatalf("len(insightRepo.insights) = %d, want 1 persisted insight", len(insightRepo.insights))
	}

	insight := result.Insights[0]
	if insight.Category != domain.DetectorToolSchemaBloat {
		t.Fatalf("insight.Category = %q, want %q", insight.Category, domain.DetectorToolSchemaBloat)
	}
	if got := insightHashValue(t, insight.Payload.Hashes, "current_model"); got != "gpt-4.1" {
		t.Fatalf("current_model hash = %q, want gpt-4.1", got)
	}
	if got := insightCountValue(t, insight.Payload.Counts, "tool_schema_bytes"); got != 72000 {
		t.Fatalf("tool_schema_bytes = %d, want 72000", got)
	}
	if got := insightCountValue(t, insight.Payload.Counts, "tool_definition_count"); got != 24 {
		t.Fatalf("tool_definition_count = %d, want 24", got)
	}
	if got := insightMetricValueTask19(t, insight.Payload.Metrics, "estimated_schema_tokens"); got != 18000 {
		t.Fatalf("estimated_schema_tokens = %v, want 18000", got)
	}
	if got := insightMetricValueTask19(t, insight.Payload.Metrics, "estimated_waste_usd"); got <= 0.03 {
		t.Fatalf("estimated_waste_usd = %v, want > 0.03", got)
	}
	assertInsightPayloadPrivacySafe(t, insight.Payload)
	for _, id := range insight.Payload.UsageEntryIDs {
		if strings.Contains(id, "{") || strings.Contains(id, "text") {
			t.Fatalf("usage entry id %q looks like raw schema content leaked into payload", id)
		}
	}
}

type fixtureSessionRepository struct {
	sessions []domain.SessionSummary
}

func (r *fixtureSessionRepository) UpsertSessions(_ context.Context, sessions []domain.SessionSummary) error {
	r.sessions = append([]domain.SessionSummary(nil), sessions...)
	return nil
}

func (r *fixtureSessionRepository) ListSessions(_ context.Context, _ ports.SessionFilter) ([]domain.SessionSummary, error) {
	return append([]domain.SessionSummary(nil), r.sessions...), nil
}

type fixtureUsageRepository struct {
	entries []domain.UsageEntry
}

func (r *fixtureUsageRepository) UpsertUsageEntries(_ context.Context, entries []domain.UsageEntry) error {
	r.entries = append([]domain.UsageEntry(nil), entries...)
	return nil
}

func (r *fixtureUsageRepository) ListUsageEntries(_ context.Context, _ ports.UsageFilter) ([]domain.UsageEntry, error) {
	return append([]domain.UsageEntry(nil), r.entries...), nil
}

type captureInsightRepoC struct {
	insights []domain.Insight
}

func (r *captureInsightRepoC) UpsertInsights(_ context.Context, insights []domain.Insight) error {
	r.insights = append([]domain.Insight(nil), insights...)
	return nil
}

func (r *captureInsightRepoC) ListInsights(_ context.Context, _ domain.MonthlyPeriod) ([]domain.Insight, error) {
	return append([]domain.Insight(nil), r.insights...), nil
}

func fixtureSessionSummary(t *testing.T, sessionID string, provider domain.ProviderName, modelID, lookupKey string, startedAt time.Time, inputTokens, outputTokens int64, totalUSD float64) domain.SessionSummary {
	t.Helper()

	pricingRef, err := domain.NewModelPricingRef(provider, modelID, lookupKey)
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	tokens, err := domain.NewTokenUsage(inputTokens, outputTokens, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(0, 0, 0, 0, 0, totalUSD)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	summary, err := domain.NewSessionSummary(domain.SessionSummary{
		SessionID:     sessionID,
		Source:        domain.UsageSourceCLISession,
		Provider:      provider,
		BillingMode:   domain.BillingModeBYOK,
		StartedAt:     startedAt,
		EndedAt:       startedAt.Add(90 * time.Second),
		ProjectName:   "fixture-project",
		AgentName:     "fixture-agent",
		PricingRef:    &pricingRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}
	return summary
}

func fixtureUsageEntry(t *testing.T, entryID, sessionID string, provider domain.ProviderName, modelID, lookupKey string, occurredAt time.Time, inputTokens, outputTokens int64, metadata map[string]string) domain.UsageEntry {
	t.Helper()

	pricingRef, err := domain.NewModelPricingRef(provider, modelID, lookupKey)
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	tokens, err := domain.NewTokenUsage(inputTokens, outputTokens, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(0, 0, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		SessionID:     sessionID,
		Source:        domain.UsageSourceCLISession,
		Provider:      provider,
		BillingMode:   domain.BillingModeBYOK,
		OccurredAt:    occurredAt,
		ExternalID:    entryID,
		ProjectName:   "fixture-project",
		AgentName:     "fixture-agent",
		Metadata:      metadata,
		PricingRef:    &pricingRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}
	return entry
}

func insightHashValue(t *testing.T, hashes []domain.InsightHash, kind string) string {
	t.Helper()
	for _, hash := range hashes {
		if hash.Kind == kind {
			return hash.Value
		}
	}
	t.Fatalf("missing insight hash %q", kind)
	return ""
}

func insightCountValue(t *testing.T, counts []domain.InsightCount, key string) int64 {
	t.Helper()
	for _, count := range counts {
		if count.Key == key {
			return count.Value
		}
	}
	t.Fatalf("missing insight count %q", key)
	return 0
}

func insightMetricValueTask19(t *testing.T, metrics []domain.InsightMetric, key string) float64 {
	t.Helper()
	for _, metric := range metrics {
		if metric.Key == key {
			return metric.Value
		}
	}
	t.Fatalf("missing insight metric %q", key)
	return 0
}

func assertInsightPayloadPrivacySafe(t *testing.T, payload domain.InsightPayload) {
	t.Helper()
	const sentinel = "SENSITIVE_PROMPT_SENTINEL"

	for _, id := range append(append([]string{}, payload.SessionIDs...), payload.UsageEntryIDs...) {
		if strings.Contains(id, sentinel) {
			t.Fatalf("payload leaked sensitive identifier %q", id)
		}
	}
	for _, hash := range payload.Hashes {
		if strings.Contains(hash.Value, sentinel) {
			t.Fatalf("payload leaked sensitive hash value %+v", hash)
		}
	}
}
