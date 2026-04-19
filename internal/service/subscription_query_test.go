package service

import (
	"context"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestListSubscriptionPresets(t *testing.T) {
	t.Parallel()

	presets := ListSubscriptionPresets()
	if got := len(presets); got != 9 {
		t.Fatalf("len(ListSubscriptionPresets()) = %d, want 9", got)
	}

	if got := presets[0].Key; got != "chatgpt-plus" {
		t.Fatalf("first preset key = %q, want chatgpt-plus", got)
	}
	if got := presets[len(presets)-1].Key; got != "gemini-ultra" {
		t.Fatalf("last preset key = %q, want gemini-ultra", got)
	}
}

func TestResolveSubscriptionPreset(t *testing.T) {
	t.Parallel()

	preset, err := ResolveSubscriptionPreset("claude-max-20x")
	if err != nil {
		t.Fatalf("ResolveSubscriptionPreset() error = %v", err)
	}

	if preset.Provider != domain.ProviderClaude || preset.DefaultFeeUSD != 200 {
		t.Fatalf("preset = %+v, want claude provider and fee 200", preset)
	}
}

func TestSubscriptionQueryServiceReturnsEmptySlice(t *testing.T) {
	t.Parallel()

	service := NewSubscriptionQueryService(&memorySubscriptionRepository{})
	snapshot, err := service.QuerySubscriptions(context.Background(), SubscriptionQuery{})
	if err != nil {
		t.Fatalf("QuerySubscriptions() error = %v", err)
	}

	if !snapshot.Empty {
		t.Fatal("QuerySubscriptions() Empty = false, want true")
	}
	if snapshot.Items == nil {
		t.Fatal("QuerySubscriptions() Items = nil, want empty slice")
	}
	if got := len(snapshot.Items); got != 0 {
		t.Fatalf("len(QuerySubscriptions().Items) = %d, want 0", got)
	}
}

func TestSubscriptionQueryServiceReturnsOrderedItems(t *testing.T) {
	t.Parallel()

	repo := &querySubscriptionRepository{}
	now := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	first := mustSubscription(t, domain.Subscription{
		SubscriptionID: "openai-chatgpt-plus-2026-04-01",
		Provider:       domain.ProviderOpenAI,
		PlanCode:       "openai-chatgpt-plus",
		PlanName:       "ChatGPT Plus",
		RenewalDay:     1,
		StartsAt:       now,
		FeeUSD:         20,
		IsActive:       true,
	})
	second := mustSubscription(t, domain.Subscription{
		SubscriptionID: "gemini-ultra-2026-04-01",
		Provider:       domain.ProviderGemini,
		PlanCode:       "gemini-gemini-ultra",
		PlanName:       "Gemini Ultra",
		RenewalDay:     1,
		StartsAt:       now,
		FeeUSD:         249.99,
		IsActive:       true,
	})

	if err := repo.UpsertSubscriptions(context.Background(), []domain.Subscription{second, first}); err != nil {
		t.Fatalf("UpsertSubscriptions() error = %v", err)
	}

	service := NewSubscriptionQueryService(repo)
	snapshot, err := service.QuerySubscriptions(context.Background(), SubscriptionQuery{})
	if err != nil {
		t.Fatalf("QuerySubscriptions() error = %v", err)
	}

	if got := len(snapshot.Items); got != 2 {
		t.Fatalf("len(QuerySubscriptions().Items) = %d, want 2", got)
	}
	if got := snapshot.Items[0].Provider; got != domain.ProviderGemini {
		t.Fatalf("first provider = %q, want gemini", got)
	}
	if got := snapshot.Items[1].Provider; got != domain.ProviderOpenAI {
		t.Fatalf("second provider = %q, want openai", got)
	}

	activeOnly := true
	filtered, err := service.QuerySubscriptions(context.Background(), SubscriptionQuery{Provider: domain.ProviderOpenAI, Active: &activeOnly})
	if err != nil {
		t.Fatalf("QuerySubscriptions(filtered) error = %v", err)
	}
	if got := len(filtered.Items); got != 1 {
		t.Fatalf("len(QuerySubscriptions(filtered).Items) = %d, want 1", got)
	}
}

type querySubscriptionRepository struct {
	items []domain.Subscription
}

func (r *querySubscriptionRepository) UpsertSubscriptions(_ context.Context, subscriptions []domain.Subscription) error {
	r.items = append(r.items, subscriptions...)
	return nil
}

func (r *querySubscriptionRepository) ListSubscriptions(_ context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error) {
	items := make([]domain.Subscription, 0, len(r.items))
	for _, item := range r.items {
		if filter.Provider != "" && item.Provider != filter.Provider {
			continue
		}
		if filter.Active != nil && item.IsActive != *filter.Active {
			continue
		}
		items = append(items, item)
	}
	if items == nil {
		items = make([]domain.Subscription, 0)
	}
	return items, nil
}

func (r *querySubscriptionRepository) DeleteSubscription(context.Context, string) error {
	return nil
}

func (r *querySubscriptionRepository) DisableSubscription(context.Context, string, time.Time) error {
	return nil
}

func (r *querySubscriptionRepository) UpsertSubscriptionFees(context.Context, []domain.SubscriptionFee) error {
	return nil
}

func (r *querySubscriptionRepository) ListSubscriptionFees(context.Context, domain.MonthlyPeriod) ([]domain.SubscriptionFee, error) {
	return nil, nil
}
