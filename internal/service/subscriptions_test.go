package service

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestSubscriptionRollupOccursOncePerBillingPeriod(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.March)
	repo := &memorySubscriptionRepository{}
	usageRepo := &memoryUsageRepository{
		entries: []domain.UsageEntry{
			mustUsageEntry(t, "usage-1", domain.UsageSourceManualAPI, domain.ProviderOpenAI, domain.BillingModeDirectAPI, time.Date(2026, time.March, 4, 12, 0, 0, 0, time.UTC), 12.5),
		},
	}
	service := NewSubscriptionService(repo, usageRepo)

	if err := service.SaveSubscriptions(context.Background(), []domain.Subscription{
		mustSubscription(t, domain.Subscription{
			SubscriptionID: "sub-claude",
			Provider:       domain.ProviderClaude,
			PlanCode:       "claude-max",
			PlanName:       "Claude Max",
			RenewalDay:     15,
			StartsAt:       time.Date(2026, time.January, 15, 8, 0, 0, 0, time.UTC),
			FeeUSD:         200,
			IsActive:       true,
		}),
	}); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	firstRollup, err := service.RollupMonthlySpend(context.Background(), period)
	if err != nil {
		t.Fatalf("first RollupMonthlySpend() error = %v", err)
	}
	secondRollup, err := service.RollupMonthlySpend(context.Background(), period)
	if err != nil {
		t.Fatalf("second RollupMonthlySpend() error = %v", err)
	}

	if got := len(firstRollup.SubscriptionFees); got != 1 {
		t.Fatalf("len(firstRollup.SubscriptionFees) = %d, want 1", got)
	}
	if got := len(secondRollup.SubscriptionFees); got != 1 {
		t.Fatalf("len(secondRollup.SubscriptionFees) = %d, want 1", got)
	}
	if got := len(repo.fees); got != 1 {
		t.Fatalf("persisted fee count = %d, want 1", got)
	}

	fee := firstRollup.SubscriptionFees[0]
	if got, want := fee.ChargedAt, time.Date(2026, time.March, 15, 8, 0, 0, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("fee.ChargedAt = %v, want %v", got, want)
	}
	if got, want := firstRollup.SubscriptionSpendUSD, 200.0; got != want {
		t.Fatalf("SubscriptionSpendUSD = %v, want %v", got, want)
	}
	if got, want := firstRollup.VariableSpendUSD, 12.5; got != want {
		t.Fatalf("VariableSpendUSD = %v, want %v", got, want)
	}
	if got, want := firstRollup.TotalSpendUSD, 212.5; got != want {
		t.Fatalf("TotalSpendUSD = %v, want %v", got, want)
	}
}

func TestInactiveSubscriptionExcluded(t *testing.T) {
	period := mustMonthlyPeriod(t, 2026, time.April)
	repo := &memorySubscriptionRepository{}
	usageRepo := &memoryUsageRepository{}
	service := NewSubscriptionService(repo, usageRepo)

	if err := service.SaveSubscriptions(context.Background(), []domain.Subscription{
		mustSubscription(t, domain.Subscription{
			SubscriptionID: "sub-gemini",
			Provider:       domain.ProviderGemini,
			PlanCode:       "gemini-advanced",
			PlanName:       "Gemini Advanced",
			RenewalDay:     10,
			StartsAt:       time.Date(2026, time.January, 10, 9, 0, 0, 0, time.UTC),
			FeeUSD:         19.99,
			IsActive:       true,
		}),
	}); err != nil {
		t.Fatalf("SaveSubscriptions() error = %v", err)
	}

	if err := service.DisableSubscription(context.Background(), "sub-gemini", time.Date(2026, time.March, 20, 18, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("DisableSubscription() error = %v", err)
	}

	marchRollup, err := service.RollupMonthlySpend(context.Background(), mustMonthlyPeriod(t, 2026, time.March))
	if err != nil {
		t.Fatalf("March RollupMonthlySpend() error = %v", err)
	}
	if got := len(marchRollup.SubscriptionFees); got != 1 {
		t.Fatalf("len(marchRollup.SubscriptionFees) = %d, want 1", got)
	}

	aprilRollup, err := service.RollupMonthlySpend(context.Background(), period)
	if err != nil {
		t.Fatalf("April RollupMonthlySpend() error = %v", err)
	}

	if got := len(aprilRollup.SubscriptionFees); got != 0 {
		t.Fatalf("len(aprilRollup.SubscriptionFees) = %d, want 0", got)
	}
	if got := aprilRollup.SubscriptionSpendUSD; got != 0 {
		t.Fatalf("April SubscriptionSpendUSD = %v, want 0", got)
	}
}

func TestSubscriptionCrudLifecycle(t *testing.T) {
	repo := &memorySubscriptionRepository{}
	service := NewSubscriptionService(repo, &memoryUsageRepository{})

	subscription := mustSubscription(t, domain.Subscription{
		SubscriptionID: "sub-openai",
		Provider:       domain.ProviderOpenAI,
		PlanCode:       "chatgpt-plus",
		PlanName:       "ChatGPT Plus",
		RenewalDay:     5,
		StartsAt:       time.Date(2026, time.February, 5, 7, 0, 0, 0, time.UTC),
		FeeUSD:         20,
		IsActive:       true,
	})

	if err := service.SaveSubscriptions(context.Background(), []domain.Subscription{subscription}); err != nil {
		t.Fatalf("initial SaveSubscriptions() error = %v", err)
	}

	subscription.PlanName = "ChatGPT Plus Updated"
	subscription.FeeUSD = 22
	if err := service.SaveSubscriptions(context.Background(), []domain.Subscription{subscription}); err != nil {
		t.Fatalf("update SaveSubscriptions() error = %v", err)
	}

	active := true
	stored, err := service.ListSubscriptions(context.Background(), ports.SubscriptionFilter{Active: &active})
	if err != nil {
		t.Fatalf("ListSubscriptions(active) error = %v", err)
	}
	if got := len(stored); got != 1 {
		t.Fatalf("len(stored active subscriptions) = %d, want 1", got)
	}
	if got, want := stored[0].PlanName, "ChatGPT Plus Updated"; got != want {
		t.Fatalf("stored[0].PlanName = %q, want %q", got, want)
	}
	if got, want := stored[0].FeeUSD, 22.0; got != want {
		t.Fatalf("stored[0].FeeUSD = %v, want %v", got, want)
	}

	disabledAt := time.Date(2026, time.March, 6, 12, 0, 0, 0, time.UTC)
	if err := service.DisableSubscription(context.Background(), subscription.SubscriptionID, disabledAt); err != nil {
		t.Fatalf("DisableSubscription() error = %v", err)
	}

	inactive := false
	stored, err = service.ListSubscriptions(context.Background(), ports.SubscriptionFilter{Active: &inactive})
	if err != nil {
		t.Fatalf("ListSubscriptions(inactive) error = %v", err)
	}
	if got := len(stored); got != 1 {
		t.Fatalf("len(stored inactive subscriptions) = %d, want 1", got)
	}
	if stored[0].EndsAt == nil || !stored[0].EndsAt.Equal(disabledAt) {
		t.Fatalf("stored inactive ends_at = %v, want %v", stored[0].EndsAt, disabledAt)
	}
}

type memorySubscriptionRepository struct {
	subscriptions map[string]domain.Subscription
	fees          map[string]domain.SubscriptionFee
}

func (r *memorySubscriptionRepository) UpsertSubscriptions(_ context.Context, subscriptions []domain.Subscription) error {
	if r.subscriptions == nil {
		r.subscriptions = make(map[string]domain.Subscription)
	}
	for _, subscription := range subscriptions {
		r.subscriptions[subscription.SubscriptionID] = subscription
	}
	return nil
}

func (r *memorySubscriptionRepository) ListSubscriptions(_ context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error) {
	items := make([]domain.Subscription, 0)
	for _, subscription := range r.subscriptions {
		if filter.SubscriptionID != "" && subscription.SubscriptionID != filter.SubscriptionID {
			continue
		}
		if filter.Provider != "" && subscription.Provider != filter.Provider {
			continue
		}
		if filter.PlanCode != "" && subscription.PlanCode != filter.PlanCode {
			continue
		}
		if filter.Active != nil && subscription.IsActive != *filter.Active {
			continue
		}
		if filter.Period != nil && !subscription.OverlapsPeriod(*filter.Period) {
			continue
		}
		items = append(items, subscription)
	}
	return items, nil
}

func (r *memorySubscriptionRepository) DisableSubscription(_ context.Context, subscriptionID string, disabledAt time.Time) error {
	subscription := r.subscriptions[subscriptionID]
	subscription.IsActive = false
	subscription.EndsAt = &disabledAt
	subscription.UpdatedAt = disabledAt
	r.subscriptions[subscriptionID] = subscription
	return nil
}

func (r *memorySubscriptionRepository) UpsertSubscriptionFees(_ context.Context, fees []domain.SubscriptionFee) error {
	if r.fees == nil {
		r.fees = make(map[string]domain.SubscriptionFee)
	}
	for _, fee := range fees {
		r.fees[fee.SubscriptionID+":"+fee.Period.StartAt.Format(time.RFC3339Nano)] = fee
	}
	return nil
}

func (r *memorySubscriptionRepository) ListSubscriptionFees(_ context.Context, period domain.MonthlyPeriod) ([]domain.SubscriptionFee, error) {
	items := make([]domain.SubscriptionFee, 0)
	for _, fee := range r.fees {
		if fee.Period.StartAt.Equal(period.StartAt) {
			items = append(items, fee)
		}
	}
	return items, nil
}

type memoryUsageRepository struct {
	entries []domain.UsageEntry
}

func (r *memoryUsageRepository) UpsertUsageEntries(_ context.Context, entries []domain.UsageEntry) error {
	r.entries = append(r.entries, entries...)
	return nil
}

func (r *memoryUsageRepository) ListUsageEntries(_ context.Context, filter ports.UsageFilter) ([]domain.UsageEntry, error) {
	items := make([]domain.UsageEntry, 0)
	for _, entry := range r.entries {
		if filter.Period != nil && !filter.Period.Contains(entry.OccurredAt) {
			continue
		}
		items = append(items, entry)
	}
	return items, nil
}

func mustSubscription(t *testing.T, subscription domain.Subscription) domain.Subscription {
	t.Helper()

	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	if subscription.CreatedAt.IsZero() {
		subscription.CreatedAt = now
	}
	if subscription.UpdatedAt.IsZero() {
		subscription.UpdatedAt = now
	}

	normalized, err := domain.NewSubscription(subscription)
	if err != nil {
		t.Fatalf("NewSubscription() error = %v", err)
	}

	return normalized
}

func mustMonthlyPeriod(t *testing.T, year int, month time.Month) domain.MonthlyPeriod {
	t.Helper()

	period, err := domain.NewMonthlyPeriodFromParts(year, month)
	if err != nil {
		t.Fatalf("NewMonthlyPeriodFromParts() error = %v", err)
	}

	return period
}

func mustUsageEntry(t *testing.T, id string, source domain.UsageSourceKind, provider domain.ProviderName, billingMode domain.BillingMode, occurredAt time.Time, totalUSD float64) domain.UsageEntry {
	t.Helper()

	breakdown, err := domain.NewCostBreakdown(0, 0, 0, 0, 0, totalUSD)
	if err != nil {
		t.Fatalf("NewCostBreakdown() error = %v", err)
	}
	entry, err := domain.NewUsageEntry(domain.UsageEntry{
		EntryID:       id,
		Source:        source,
		Provider:      provider,
		BillingMode:   billingMode,
		OccurredAt:    occurredAt,
		CostBreakdown: breakdown,
	})
	if err != nil {
		t.Fatalf("NewUsageEntry() error = %v", err)
	}

	return entry
}
