package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
)

func TestDetectorSetBFindings(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	sessions := []domain.SessionSummary{
		mustTask18SessionSummary(t, "session-repeated-reads", time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC), time.Date(2026, 4, 17, 9, 8, 0, 0, time.UTC)),
		mustTask18SessionSummary(t, "session-retry-storm", time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC), time.Date(2026, 4, 17, 10, 6, 0, 0, time.UTC)),
		mustTask18SessionSummary(t, "session-one-off-retry", time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC), time.Date(2026, 4, 17, 11, 2, 0, 0, time.UTC)),
	}

	entries := []domain.UsageEntry{
		mustTask18UsageEntry(t, "read-1", "session-repeated-reads", time.Date(2026, 4, 17, 9, 0, 0, 0, time.UTC), 120, 0.010, map[string]string{"tool_name": "read_file", "target_kind": "file", "file_target_hash": "file:alpha", "action_signature": "read:file:alpha", "status": "success"}),
		mustTask18UsageEntry(t, "read-2", "session-repeated-reads", time.Date(2026, 4, 17, 9, 1, 0, 0, time.UTC), 120, 0.010, map[string]string{"tool_name": "read_file", "target_kind": "file", "file_target_hash": "file:alpha", "action_signature": "read:file:alpha", "status": "success"}),
		mustTask18UsageEntry(t, "read-3", "session-repeated-reads", time.Date(2026, 4, 17, 9, 2, 0, 0, time.UTC), 130, 0.011, map[string]string{"tool_name": "read_file", "target_kind": "file", "file_target_hash": "file:alpha", "action_signature": "read:file:alpha", "status": "success"}),
		mustTask18UsageEntry(t, "read-4", "session-repeated-reads", time.Date(2026, 4, 17, 9, 3, 0, 0, time.UTC), 140, 0.012, map[string]string{"tool_name": "read_file", "target_kind": "file", "file_target_hash": "file:alpha", "action_signature": "read:file:alpha", "status": "success"}),
		mustTask18UsageEntry(t, "read-5", "session-repeated-reads", time.Date(2026, 4, 17, 9, 4, 0, 0, time.UTC), 150, 0.013, map[string]string{"tool_name": "read_file", "target_kind": "file", "file_target_hash": "file:alpha", "action_signature": "read:file:alpha", "status": "success"}),
		mustTask18UsageEntry(t, "read-control", "session-repeated-reads", time.Date(2026, 4, 17, 9, 5, 0, 0, time.UTC), 80, 0.006, map[string]string{"tool_name": "read_file", "target_kind": "file", "file_target_hash": "file:beta", "action_signature": "read:file:beta", "status": "success"}),
		mustTask18UsageEntry(t, "retry-1", "session-retry-storm", time.Date(2026, 4, 17, 10, 0, 0, 0, time.UTC), 90, 0.007, map[string]string{"tool_name": "bash", "operation_hash": "op:lint", "retry_key": "op:lint", "error_hash": "err:timeout", "status": "error_timeout"}),
		mustTask18UsageEntry(t, "retry-2", "session-retry-storm", time.Date(2026, 4, 17, 10, 1, 0, 0, time.UTC), 100, 0.008, map[string]string{"tool_name": "bash", "operation_hash": "op:lint", "retry_key": "op:lint", "error_hash": "err:timeout", "status": "error_timeout"}),
		mustTask18UsageEntry(t, "retry-3", "session-retry-storm", time.Date(2026, 4, 17, 10, 2, 0, 0, time.UTC), 110, 0.009, map[string]string{"tool_name": "bash", "operation_hash": "op:lint", "retry_key": "op:lint", "error_hash": "err:timeout", "status": "error_timeout"}),
		mustTask18UsageEntry(t, "retry-4", "session-retry-storm", time.Date(2026, 4, 17, 10, 3, 0, 0, time.UTC), 120, 0.010, map[string]string{"tool_name": "bash", "operation_hash": "op:lint", "retry_key": "op:lint", "error_hash": "err:timeout", "status": "error_timeout"}),
		mustTask18UsageEntry(t, "retry-success", "session-retry-storm", time.Date(2026, 4, 17, 10, 4, 0, 0, time.UTC), 130, 0.011, map[string]string{"tool_name": "bash", "operation_hash": "op:lint", "retry_key": "op:lint", "status": "success"}),
		mustTask18UsageEntry(t, "one-off-1", "session-one-off-retry", time.Date(2026, 4, 17, 11, 0, 0, 0, time.UTC), 80, 0.006, map[string]string{"tool_name": "bash", "operation_hash": "op:test", "retry_key": "op:test", "error_hash": "err:flake", "status": "error"}),
		mustTask18UsageEntry(t, "one-off-2", "session-one-off-retry", time.Date(2026, 4, 17, 11, 1, 0, 0, time.UTC), 90, 0.007, map[string]string{"tool_name": "bash", "operation_hash": "op:test", "retry_key": "op:test", "status": "success"}),
	}

	usageRepo := &captureUsageEntryRepository{entries: entries}
	sessionRepo := &captureSessionRepository{sessions: sessions}
	insightRepo := &task18CaptureInsightRepository{}
	executor := NewInsightExecutorService(NewDetectorSetB(), sessionRepo, usageRepo, insightRepo)

	result, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if len(result.Insights) != 2 {
		t.Fatalf("len(result.Insights) = %d, want 2", len(result.Insights))
	}
	if len(insightRepo.insights) != 2 {
		t.Fatalf("len(insightRepo.insights) = %d, want 2", len(insightRepo.insights))
	}

	byCategory := make(map[domain.DetectorCategory]domain.Insight, len(result.Insights))
	for _, insight := range result.Insights {
		byCategory[insight.Category] = insight
		assertPrivacySafePayload(t, insight)
	}

	readInsight, ok := byCategory[domain.DetectorRepeatedFileReads]
	if !ok {
		t.Fatal("missing repeated_file_reads insight")
	}
	assertMetricPositive(t, readInsight, "suspected_waste_usd")
	assertMetricPositive(t, readInsight, "suspected_waste_tokens")
	assertMetricPositive(t, readInsight, "suspected_waste_seconds")
	assertCountValue(t, readInsight, "suspected_waste_events", 3)

	retryInsight, ok := byCategory[domain.DetectorRetryAmplification]
	if !ok {
		t.Fatal("missing retry_amplification insight")
	}
	assertMetricPositive(t, retryInsight, "suspected_waste_usd")
	assertMetricPositive(t, retryInsight, "suspected_waste_tokens")
	assertMetricPositive(t, retryInsight, "suspected_waste_seconds")
	assertCountValue(t, retryInsight, "retry_attempts", 3)
	assertCountValue(t, retryInsight, "suspected_waste_events", 3)

	if _, exists := byCategory[domain.DetectorZombieLoops]; exists {
		t.Fatal("unexpected zombie_loops insight for retry fixture with eventual success")
	}
}

func TestZombieLoopAvoidsFalsePositive(t *testing.T) {
	period, err := domain.NewMonthlyPeriod(time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	session := mustTask18SessionSummary(t, "session-progressing-loop", time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC), time.Date(2026, 4, 17, 12, 7, 0, 0, time.UTC))
	entries := []domain.UsageEntry{
		mustTask18UsageEntry(t, "loop-1", session.SessionID, time.Date(2026, 4, 17, 12, 0, 0, 0, time.UTC), 100, 0.008, map[string]string{"tool_name": "planner", "loop_signature": "loop:plan", "progress_marker": "step-1", "status": "success"}),
		mustTask18UsageEntry(t, "loop-2", session.SessionID, time.Date(2026, 4, 17, 12, 1, 0, 0, time.UTC), 100, 0.008, map[string]string{"tool_name": "planner", "loop_signature": "loop:plan", "progress_marker": "step-2", "status": "success"}),
		mustTask18UsageEntry(t, "loop-3", session.SessionID, time.Date(2026, 4, 17, 12, 2, 0, 0, time.UTC), 100, 0.008, map[string]string{"tool_name": "planner", "loop_signature": "loop:plan", "progress_marker": "step-3", "status": "success"}),
		mustTask18UsageEntry(t, "loop-4", session.SessionID, time.Date(2026, 4, 17, 12, 3, 0, 0, time.UTC), 100, 0.008, map[string]string{"tool_name": "planner", "loop_signature": "loop:plan", "progress_marker": "step-4", "status": "success"}),
		mustTask18UsageEntry(t, "loop-5", session.SessionID, time.Date(2026, 4, 17, 12, 4, 0, 0, time.UTC), 100, 0.008, map[string]string{"tool_name": "planner", "loop_signature": "loop:plan", "progress_marker": "step-5", "status": "success"}),
	}

	usageRepo := &captureUsageEntryRepository{entries: entries}
	sessionRepo := &captureSessionRepository{sessions: []domain.SessionSummary{session}}
	insightRepo := &task18CaptureInsightRepository{}
	executor := NewInsightExecutorService(NewDetectorSetB(), sessionRepo, usageRepo, insightRepo)

	result, err := executor.Execute(context.Background(), period)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	for _, insight := range result.Insights {
		if insight.Category == domain.DetectorZombieLoops {
			t.Fatalf("unexpected zombie_loops insight: %+v", insight)
		}
	}
}

type task18CaptureInsightRepository struct {
	insights []domain.Insight
}

func (r *task18CaptureInsightRepository) UpsertInsights(_ context.Context, insights []domain.Insight) error {
	r.insights = append([]domain.Insight(nil), insights...)
	return nil
}

func (r *task18CaptureInsightRepository) ListInsights(_ context.Context, _ domain.MonthlyPeriod) ([]domain.Insight, error) {
	return append([]domain.Insight(nil), r.insights...), nil
}

func mustTask18SessionSummary(t *testing.T, sessionID string, startedAt, endedAt time.Time) domain.SessionSummary {
	t.Helper()
	pricingRef, err := domain.NewModelPricingRef(domain.ProviderOpenAI, "gpt-5-mini", "gpt-5-mini")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}
	tokens, err := domain.NewTokenUsage(1000, 200, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(0.020, 0.010, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	summary, err := domain.NewSessionSummary(domain.SessionSummary{
		SessionID:     sessionID,
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeBYOK,
		StartedAt:     startedAt,
		EndedAt:       endedAt,
		ProjectName:   "task-18-project",
		AgentName:     "codex",
		PricingRef:    &pricingRef,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewSessionSummary() error = %v", err)
	}
	return summary
}

func mustTask18UsageEntry(t *testing.T, entryID, sessionID string, occurredAt time.Time, totalTokens int64, totalUSD float64, metadata map[string]string) domain.UsageEntry {
	t.Helper()
	tokens, err := domain.NewTokenUsage(totalTokens, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}
	costs, err := domain.NewCostBreakdown(totalUSD, 0, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       entryID,
		Source:        domain.UsageSourceCLISession,
		Provider:      domain.ProviderOpenAI,
		BillingMode:   domain.BillingModeBYOK,
		OccurredAt:    occurredAt,
		SessionID:     sessionID,
		ExternalID:    entryID,
		ProjectName:   "task-18-project",
		AgentName:     "codex",
		Metadata:      metadata,
		Tokens:        tokens,
		CostBreakdown: costs,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}
	return entry
}

func assertPrivacySafePayload(t *testing.T, insight domain.Insight) {
	t.Helper()
	encoded, err := json.Marshal(insight.Payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	serialized := string(encoded)
	for _, forbidden := range []string{"prompt", "response", "content", "transcript", "body"} {
		if containsAny(strings.ToLower(serialized), forbidden) {
			t.Fatalf("payload leaked forbidden field %q: %s", forbidden, serialized)
		}
	}
}

func assertMetricPositive(t *testing.T, insight domain.Insight, key string) {
	t.Helper()
	for _, metric := range insight.Payload.Metrics {
		if metric.Key == key {
			if metric.Value <= 0 {
				t.Fatalf("metric %q = %f, want > 0", key, metric.Value)
			}
			return
		}
	}
	t.Fatalf("missing metric %q", key)
}

func assertCountValue(t *testing.T, insight domain.Insight, key string, want int64) {
	t.Helper()
	for _, count := range insight.Payload.Counts {
		if count.Key == key {
			if count.Value != want {
				t.Fatalf("count %q = %d, want %d", key, count.Value, want)
			}
			return
		}
	}
	t.Fatalf("missing count %q", key)
}
