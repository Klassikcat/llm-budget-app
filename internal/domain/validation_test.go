package domain

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestSessionValidation(t *testing.T) {
	t.Run("rejects negative token counts", func(t *testing.T) {
		_, err := NewTokenUsage(-1, 0, 0, 0)
		if err == nil {
			t.Fatal("NewTokenUsage() error = nil, want validation error")
		}
		if !IsValidationCode(err, ValidationCodeNegativeTokens) {
			t.Fatalf("NewTokenUsage() error = %v, want negative token validation", err)
		}
	})

	t.Run("rejects negative costs", func(t *testing.T) {
		_, err := NewCostBreakdown(0, 0, 0, 0, -0.01, 0)
		if err == nil {
			t.Fatal("NewCostBreakdown() error = nil, want validation error")
		}
		if !IsValidationCode(err, ValidationCodeNegativeCost) {
			t.Fatalf("NewCostBreakdown() error = %v, want negative cost validation", err)
		}
	})

	t.Run("rejects invalid billing modes", func(t *testing.T) {
		_, err := ParseBillingMode("enterprise")
		if err == nil {
			t.Fatal("ParseBillingMode() error = nil, want validation error")
		}
		if !IsValidationCode(err, ValidationCodeInvalidBillingMode) {
			t.Fatalf("ParseBillingMode() error = %v, want invalid billing mode validation", err)
		}
	})

	t.Run("rejects malformed provider names", func(t *testing.T) {
		_, err := NewProviderName("OpenAI/")
		if err == nil {
			t.Fatal("NewProviderName() error = nil, want validation error")
		}
		if !IsValidationCode(err, ValidationCodeInvalidProviderName) {
			t.Fatalf("NewProviderName() error = %v, want invalid provider validation", err)
		}
	})

	t.Run("normalizes timestamps to UTC", func(t *testing.T) {
		provider, err := NewProviderName("openai")
		if err != nil {
			t.Fatalf("NewProviderName() error = %v", err)
		}

		start := time.Date(2026, 4, 17, 9, 0, 0, 0, time.FixedZone("KST", 9*60*60))
		end := start.Add(15 * time.Minute)

		summary, err := NewSessionSummary(SessionSummary{
			SessionID:   "sess-1",
			Source:      UsageSourceCLISession,
			Provider:    provider,
			BillingMode: BillingModeBYOK,
			StartedAt:   start,
			EndedAt:     end,
		})
		if err != nil {
			t.Fatalf("NewSessionSummary() error = %v", err)
		}

		if summary.StartedAt.Location() != time.UTC {
			t.Fatalf("summary.StartedAt location = %v, want UTC", summary.StartedAt.Location())
		}
		if summary.EndedAt.Location() != time.UTC {
			t.Fatalf("summary.EndedAt location = %v, want UTC", summary.EndedAt.Location())
		}
		if summary.Duration() != 15*time.Minute {
			t.Fatalf("summary.Duration() = %v, want 15m", summary.Duration())
		}
	})

	t.Run("rejects impossible time ranges", func(t *testing.T) {
		provider, err := NewProviderName("anthropic")
		if err != nil {
			t.Fatalf("NewProviderName() error = %v", err)
		}

		start := time.Date(2026, 4, 17, 0, 0, 0, 0, time.UTC)
		end := start.Add(-time.Minute)

		_, err = NewSessionSummary(SessionSummary{
			SessionID:   "sess-2",
			Source:      UsageSourceCLISession,
			Provider:    provider,
			BillingMode: BillingModeSubscription,
			StartedAt:   start,
			EndedAt:     end,
		})
		if err == nil {
			t.Fatal("NewSessionSummary() error = nil, want validation error")
		}
		if !IsValidationCode(err, ValidationCodeInvalidTimeRange) {
			t.Fatalf("NewSessionSummary() error = %v, want invalid time range validation", err)
		}
	})
}

func TestUsageEntrySupportsRequiredSources(t *testing.T) {
	provider, err := NewProviderName("openai")
	if err != nil {
		t.Fatalf("NewProviderName() error = %v", err)
	}

	pricingRef, err := NewModelPricingRef(provider, "gpt-4.1", "openai/gpt-4.1")
	if err != nil {
		t.Fatalf("NewModelPricingRef() error = %v", err)
	}

	tokens, err := NewTokenUsage(100, 40, 0, 0)
	if err != nil {
		t.Fatalf("NewTokenUsage() error = %v", err)
	}

	costs, err := NewCostBreakdown(0.30, 0.20, 0, 0, 0, 0)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}

	now := time.Date(2026, 4, 17, 12, 0, 0, 0, time.FixedZone("PDT", -7*60*60))

	tests := []struct {
		name  string
		entry UsageEntry
	}{
		{
			name: "subscription",
			entry: UsageEntry{
				EntryID:       "subscription-1",
				Source:        UsageSourceSubscription,
				Provider:      ProviderAnthropic,
				BillingMode:   BillingModeSubscription,
				OccurredAt:    now,
				CostBreakdown: mustCostBreakdown(t, 0, 0, 0, 0, 0, 20),
			},
		},
		{
			name: "manual api",
			entry: UsageEntry{
				EntryID:       "manual-api-1",
				Source:        UsageSourceManualAPI,
				Provider:      provider,
				BillingMode:   BillingModeDirectAPI,
				OccurredAt:    now,
				PricingRef:    &pricingRef,
				Tokens:        tokens,
				CostBreakdown: costs,
			},
		},
		{
			name: "openrouter",
			entry: UsageEntry{
				EntryID:       "openrouter-1",
				Source:        UsageSourceOpenRouter,
				Provider:      ProviderOpenRouter,
				BillingMode:   BillingModeOpenRouter,
				OccurredAt:    now,
				PricingRef:    &pricingRef,
				Tokens:        tokens,
				CostBreakdown: costs,
			},
		},
		{
			name: "cli session",
			entry: UsageEntry{
				EntryID:       "cli-1",
				Source:        UsageSourceCLISession,
				Provider:      ProviderClaude,
				BillingMode:   BillingModeSubscription,
				OccurredAt:    now,
				SessionID:     "session-1",
				Tokens:        tokens,
				CostBreakdown: mustCostBreakdown(t, 0, 0, 0, 0, 0, 0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entry, err := NewUsageEntry(tt.entry)
			if err != nil {
				t.Fatalf("NewUsageEntry() error = %v", err)
			}
			if entry.OccurredAt.Location() != time.UTC {
				t.Fatalf("entry.OccurredAt location = %v, want UTC", entry.OccurredAt.Location())
			}
		})
	}
}

func TestInsightPayloadPrivacy(t *testing.T) {
	period, err := NewMonthlyPeriod(time.Date(2026, 4, 17, 12, 0, 0, 0, time.FixedZone("KST", 9*60*60)))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	hash, err := NewInsightHash("project_hash", "sha256:abc123")
	if err != nil {
		t.Fatalf("NewInsightHash() error = %v", err)
	}

	count, err := NewInsightCount("retry_count", 3)
	if err != nil {
		t.Fatalf("NewInsightCount() error = %v", err)
	}

	metric, err := NewInsightMetric("avoidable_cost", InsightMetricUnitUSD, 2.75)
	if err != nil {
		t.Fatalf("NewInsightMetric() error = %v", err)
	}

	payload, err := NewInsightPayload(
		[]string{"session-1"},
		[]string{"usage-1"},
		[]InsightHash{hash},
		[]InsightCount{count},
		[]InsightMetric{metric},
	)
	if err != nil {
		t.Fatalf("NewInsightPayload() error = %v", err)
	}

	insight, err := NewInsight(Insight{
		InsightID:  "insight-1",
		Category:   DetectorPlanningTax,
		Severity:   InsightSeverityHigh,
		DetectedAt: time.Date(2026, 4, 18, 3, 45, 0, 0, time.FixedZone("PDT", -7*60*60)),
		Period:     period,
		Payload:    payload,
	})
	if err != nil {
		t.Fatalf("NewInsight() error = %v", err)
	}

	encoded, err := json.Marshal(insight.Payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	serialized := string(encoded)
	for _, forbidden := range []string{"prompt", "response", "content", "transcript", "body"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("serialized payload unexpectedly contains %q: %s", forbidden, serialized)
		}
	}

	if !strings.Contains(serialized, "session_ids") || !strings.Contains(serialized, "hashes") || !strings.Contains(serialized, "metrics") {
		t.Fatalf("serialized payload = %s, want IDs, hashes, and metrics only", serialized)
	}
	if insight.DetectedAt.Location() != time.UTC {
		t.Fatalf("insight.DetectedAt location = %v, want UTC", insight.DetectedAt.Location())
	}

	for _, category := range []DetectorCategory{
		DetectorContextAvalanche,
		DetectorRepeatedFileReads,
		DetectorRetryAmplification,
		DetectorOverQualifiedModel,
		DetectorToolSchemaBloat,
		DetectorPlanningTax,
		DetectorZombieLoops,
		DetectorMissedPromptCaching,
	} {
		if !category.IsValid() {
			t.Fatalf("DetectorCategory %q should be valid", category)
		}
	}
}

func TestBudgetPeriodRollover(t *testing.T) {
	period, err := NewMonthlyPeriod(time.Date(2026, 1, 31, 23, 30, 0, 0, time.FixedZone("PST", -8*60*60)))
	if err != nil {
		t.Fatalf("NewMonthlyPeriod() error = %v", err)
	}

	if got, want := period.StartAt, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("period.StartAt = %v, want %v", got, want)
	}
	if got, want := period.EndExclusive, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("period.EndExclusive = %v, want %v", got, want)
	}

	warningThreshold, err := NewBudgetThreshold(AlertSeverityWarning, 0.8)
	if err != nil {
		t.Fatalf("NewBudgetThreshold() error = %v", err)
	}
	criticalThreshold, err := NewBudgetThreshold(AlertSeverityCritical, 0.95)
	if err != nil {
		t.Fatalf("NewBudgetThreshold() error = %v", err)
	}

	budget, err := NewMonthlyBudget(MonthlyBudget{
		BudgetID:   "budget-1",
		Name:       "main",
		Period:     period,
		LimitUSD:   100,
		Thresholds: []BudgetThreshold{criticalThreshold, warningThreshold},
	})
	if err != nil {
		t.Fatalf("NewMonthlyBudget() error = %v", err)
	}

	status, err := budget.EvaluateSpend(110)
	if err != nil {
		t.Fatalf("EvaluateSpend() error = %v", err)
	}
	if !status.IsOverrun {
		t.Fatal("EvaluateSpend() IsOverrun = false, want true")
	}
	if len(status.TriggeredThresholds) != 2 {
		t.Fatalf("len(status.TriggeredThresholds) = %d, want 2", len(status.TriggeredThresholds))
	}

	subscription, err := NewSubscriptionFee(SubscriptionFee{
		SubscriptionID: "sub-1",
		Provider:       ProviderClaude,
		PlanCode:       "claude-max",
		ChargedAt:      time.Date(2026, 2, 28, 23, 30, 0, 0, time.FixedZone("PST", -8*60*60)),
		FeeUSD:         200,
	})
	if err != nil {
		t.Fatalf("NewSubscriptionFee() error = %v", err)
	}
	if got, want := subscription.Period.StartAt, time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("subscription period start = %v, want %v", got, want)
	}
	if got, want := subscription.Period.EndExclusive, time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("subscription period end = %v, want %v", got, want)
	}

	alert, err := NewAlertEvent(AlertEvent{
		AlertID:         "alert-1",
		Kind:            AlertKindBudgetOverrun,
		Severity:        AlertSeverityCritical,
		TriggeredAt:     time.Date(2026, 3, 1, 0, 15, 0, 0, time.FixedZone("PST", -8*60*60)),
		Period:          subscription.Period,
		BudgetID:        budget.BudgetID,
		CurrentSpendUSD: 110,
		LimitUSD:        100,
	})
	if err != nil {
		t.Fatalf("NewAlertEvent() error = %v", err)
	}
	if alert.TriggeredAt.Location() != time.UTC {
		t.Fatalf("alert.TriggeredAt location = %v, want UTC", alert.TriggeredAt.Location())
	}

	next := period.Next()
	if got, want := next.StartAt, period.EndExclusive; !got.Equal(want) {
		t.Fatalf("next.StartAt = %v, want %v", got, want)
	}
}

func mustCostBreakdown(t *testing.T, inputUSD, outputUSD, cacheReadUSD, cacheWriteUSD, toolUSD, flatUSD float64) CostBreakdown {
	t.Helper()

	breakdown, err := NewCostBreakdown(inputUSD, outputUSD, cacheReadUSD, cacheWriteUSD, toolUSD, flatUSD)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}

	return breakdown
}
