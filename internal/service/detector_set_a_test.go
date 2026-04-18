package service

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
)

func TestDetectorSetAFindings(t *testing.T) {
	store := mustDetectorStore(t, filepath.Join(t.TempDir(), "task-17-detectors.sqlite3"))
	defer store.Close()

	period := mustDetectorMonthlyPeriod(t, time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC))
	sessions, usageEntries := detectorSetAFixtures(t)
	if err := store.UpsertSessions(context.Background(), sessions); err != nil {
		t.Fatalf("UpsertSessions() error = %v", err)
	}
	if err := store.UpsertUsageEntries(context.Background(), usageEntries); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	executor := NewInsightExecutorService(NewDetectorSetA(), store, store, store)
	result, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(result.Insights) != 3 {
		t.Fatalf("len(result.Insights) = %d, want 3", len(result.Insights))
	}

	stored, err := store.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("ListInsights() error = %v", err)
	}
	if len(stored) != 3 {
		t.Fatalf("len(stored) = %d, want 3", len(stored))
	}

	byCategory := map[domain.DetectorCategory]domain.Insight{}
	for _, insight := range stored {
		byCategory[insight.Category] = insight
		assertPrivacySafeInsight(t, insight)
	}

	contextInsight, ok := byCategory[domain.DetectorContextAvalanche]
	if !ok {
		t.Fatal("missing context avalanche insight")
	}
	if contextInsight.Severity != domain.InsightSeverityHigh {
		t.Fatalf("context severity = %q, want high", contextInsight.Severity)
	}
	if !reflect.DeepEqual(contextInsight.Payload.SessionIDs, []string{"session-context"}) {
		t.Fatalf("context session IDs = %#v, want session-context", contextInsight.Payload.SessionIDs)
	}

	cacheInsight, ok := byCategory[domain.DetectorMissedPromptCaching]
	if !ok {
		t.Fatal("missing missed prompt caching insight")
	}
	if cacheInsight.Severity != domain.InsightSeverityMedium {
		t.Fatalf("cache severity = %q, want medium", cacheInsight.Severity)
	}

	planningInsight, ok := byCategory[domain.DetectorPlanningTax]
	if !ok {
		t.Fatal("missing planning tax insight")
	}
	if planningInsight.Severity != domain.InsightSeverityHigh {
		t.Fatalf("planning severity = %q, want high", planningInsight.Severity)
	}
	if got := insightMetricValueTask17(planningInsight.Payload.Metrics, "reasoning_tokens"); got != 9400 {
		t.Fatalf("planning reasoning_tokens = %v, want 9400", got)
	}
}

func TestDetectorSetADeduplicates(t *testing.T) {
	store := mustDetectorStore(t, filepath.Join(t.TempDir(), "task-17-detector-dedup.sqlite3"))
	defer store.Close()

	period := mustDetectorMonthlyPeriod(t, time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC))
	sessions, usageEntries := detectorSetAFixtures(t)
	if err := store.UpsertSessions(context.Background(), sessions); err != nil {
		t.Fatalf("UpsertSessions() error = %v", err)
	}
	if err := store.UpsertUsageEntries(context.Background(), usageEntries); err != nil {
		t.Fatalf("UpsertUsageEntries() error = %v", err)
	}

	executor := NewInsightExecutorService(NewDetectorSetA(), store, store, store)
	first, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	if len(first.Insights) != 3 {
		t.Fatalf("len(first.Insights) = %d, want 3", len(first.Insights))
	}
	storedFirst, err := store.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("ListInsights(first) error = %v", err)
	}

	second, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}
	if len(second.Insights) != 3 {
		t.Fatalf("len(second.Insights) = %d, want 3", len(second.Insights))
	}
	storedSecond, err := store.ListInsights(context.Background(), period)
	if err != nil {
		t.Fatalf("ListInsights(second) error = %v", err)
	}

	if len(storedSecond) != len(storedFirst) {
		t.Fatalf("stored insight count after rerun = %d, want %d", len(storedSecond), len(storedFirst))
	}

	firstIDs := insightIDs(storedFirst)
	secondIDs := insightIDs(storedSecond)
	if !reflect.DeepEqual(firstIDs, secondIDs) {
		t.Fatalf("stored insight IDs changed across reruns: first=%v second=%v", firstIDs, secondIDs)
	}
}

func detectorSetAFixtures(t *testing.T) ([]domain.SessionSummary, []domain.UsageEntry) {
	t.Helper()
	ref := mustPricingRef(t, domain.ProviderOpenAI, "gpt-5-mini", "gpt-5-mini")

	contextEntries := []domain.UsageEntry{
		detectorUsageEntry(t, "ctx-1", "session-context", time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC), 9000, 600, 0.90, 0.06, nil),
	}
	cacheEntries := []domain.UsageEntry{
		detectorUsageEntry(t, "cache-1", "session-cache", time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC), 900, 600, 0.09, 0.06, nil),
		detectorUsageEntry(t, "cache-2", "session-cache", time.Date(2026, 4, 17, 11, 2, 0, 0, time.UTC), 1100, 700, 0.11, 0.07, nil),
		detectorUsageEntry(t, "cache-3", "session-cache", time.Date(2026, 4, 17, 11, 4, 0, 0, time.UTC), 1200, 650, 0.12, 0.065, nil),
		detectorUsageEntry(t, "cache-4", "session-cache", time.Date(2026, 4, 17, 11, 6, 0, 0, time.UTC), 1700, 550, 0.17, 0.055, nil),
	}
	planningEntries := []domain.UsageEntry{
		detectorUsageEntry(t, "plan-1", "session-planning", time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC), 2000, 700, 0.20, 0.07, map[string]string{"opencode_reasoning_tokens": "4700", "observed_tool_call_count": "0"}),
		detectorUsageEntry(t, "plan-2", "session-planning", time.Date(2026, 4, 17, 12, 3, 0, 0, time.UTC), 1800, 500, 0.18, 0.05, map[string]string{"gemini_thought_tokens": "4700", "observed_tool_call_count": "1"}),
	}

	allEntries := append(append(append([]domain.UsageEntry{}, contextEntries...), cacheEntries...), planningEntries...)
	return []domain.SessionSummary{
		detectorSessionSummary(t, "session-context", ref, contextEntries),
		detectorSessionSummary(t, "session-cache", ref, cacheEntries),
		detectorSessionSummary(t, "session-planning", ref, planningEntries),
	}, allEntries
}

func mustDetectorStore(t *testing.T, path string) *sqlite.Store {
	t.Helper()
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: path})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	return store
}

func mustDetectorMonthlyPeriod(t *testing.T, ts time.Time) domain.MonthlyPeriod {
	t.Helper()
	period, err := domain.NewMonthlyPeriod(ts)
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}
	return period
}

func mustPricingRef(t *testing.T, provider domain.ProviderName, modelID, lookup string) *domain.ModelPricingRef {
	t.Helper()
	ref, err := domain.NewModelPricingRef(provider, modelID, lookup)
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	return &ref
}

func detectorUsageEntry(t *testing.T, entryID, sessionID string, occurredAt time.Time, inputTokens, outputTokens int64, inputUSD, outputUSD float64, metadata map[string]string) domain.UsageEntry {
	t.Helper()
	tokens, err := domain.NewTokenUsage(inputTokens, outputTokens, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(inputUSD, outputUSD, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		SessionID:     sessionID,
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeSubscription,
		OccurredAt:    occurredAt,
		ProjectName:   "project-alpha",
		AgentName:     "assistant-runner",
		Metadata:      metadata,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}
	return entry
}

func detectorSessionSummary(t *testing.T, sessionID string, pricingRef *domain.ModelPricingRef, entries []domain.UsageEntry) domain.SessionSummary {
	t.Helper()
	first := entries[0]
	last := entries[len(entries)-1]
	var totalInput, totalOutput int64
	var totalInputUSD, totalOutputUSD float64
	for _, entry := range entries {
		totalInput += entry.Tokens.InputTokens
		totalOutput += entry.Tokens.OutputTokens
		totalInputUSD += entry.CostBreakdown.InputUSD
		totalOutputUSD += entry.CostBreakdown.OutputUSD
	}
	tokens, err := domain.NewTokenUsage(totalInput, totalOutput, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage(summary) error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(totalInputUSD, totalOutputUSD, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown(summary) error = %v", err)
	}
	summary, err := domain.NewSessionSummary(domain.SessionSummary{
		SessionID:     sessionID,
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeSubscription,
		StartedAt:     first.OccurredAt,
		EndedAt:       last.OccurredAt,
		ProjectName:   first.ProjectName,
		AgentName:     first.AgentName,
		PricingRef:    pricingRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}
	return summary
}

func assertPrivacySafeInsight(t *testing.T, insight domain.Insight) {
	t.Helper()
	for _, forbidden := range []string{"prompt", "response", "content", "transcript", "body"} {
		for _, hash := range insight.Payload.Hashes {
			if strings.Contains(strings.ToLower(hash.Value), forbidden) {
				t.Fatalf("insight hash leaked forbidden content token %q: %#v", forbidden, hash)
			}
		}
	}
	if len(insight.Payload.Hashes) == 0 {
		t.Fatalf("insight %s missing hashes", insight.InsightID)
	}
	if len(insight.Payload.Metrics) == 0 {
		t.Fatalf("insight %s missing metrics", insight.InsightID)
	}
	if len(insight.Payload.SessionIDs) != 1 {
		t.Fatalf("insight %s session IDs = %#v, want single session id", insight.InsightID, insight.Payload.SessionIDs)
	}
}

func insightMetricValueTask17(metrics []domain.InsightMetric, key string) float64 {
	for _, metric := range metrics {
		if metric.Key == key {
			return metric.Value
		}
	}
	return 0
}

func insightIDs(insights []domain.Insight) []string {
	ids := make([]string, 0, len(insights))
	for _, insight := range insights {
		ids = append(ids, insight.InsightID)
	}
	return ids
}
