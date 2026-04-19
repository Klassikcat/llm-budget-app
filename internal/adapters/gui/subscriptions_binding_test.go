package gui

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/adapters/sqlite"
	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/service"
)

func TestSubscriptionLookupBindingEmptyState(t *testing.T) {
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "subscriptions-binding-empty.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	binding := newTestSubscriptionLookupBinding(store)
	response, err := binding.LoadSubscriptions()
	if err != nil {
		t.Fatalf("LoadSubscriptions() error = %v", err)
	}

	if !response.Empty {
		t.Fatal("LoadSubscriptions() empty = false, want true")
	}
	if response.Items == nil {
		t.Fatal("LoadSubscriptions() items = nil, want empty slice")
	}
	if got := len(response.Items); got != 0 {
		t.Fatalf("len(LoadSubscriptions().Items) = %d, want 0", got)
	}
}

func TestSubscriptionLookupBindingPopulatedState(t *testing.T) {
	store, err := sqlite.Bootstrap(context.Background(), sqlite.Options{Path: filepath.Join(t.TempDir(), "subscriptions-binding.sqlite3")})
	if err != nil {
		t.Fatalf("sqlite.Bootstrap() error = %v", err)
	}
	defer store.Close()

	ctx := context.Background()
	startsAt := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	if err := store.UpsertSubscriptions(ctx, []domain.Subscription{
		mustSubscriptionForGUIBinding(t, domain.ProviderGemini, "Gemini Ultra", 249.99, startsAt),
		mustSubscriptionForGUIBinding(t, domain.ProviderOpenAI, "ChatGPT Plus", 20, startsAt),
	}); err != nil {
		t.Fatalf("UpsertSubscriptions() error = %v", err)
	}

	binding := newTestSubscriptionLookupBinding(store)
	response, err := binding.LoadSubscriptions()
	if err != nil {
		t.Fatalf("LoadSubscriptions() error = %v", err)
	}

	if response.Empty {
		t.Fatal("LoadSubscriptions() empty = true, want false")
	}
	if got := len(response.Items); got != 2 {
		t.Fatalf("len(LoadSubscriptions().Items) = %d, want 2", got)
	}
	if got := response.Items[0]; got.Provider != "gemini" || got.PlanName != "Gemini Ultra" || got.StartsAt != "2026-04-01" {
		t.Fatalf("response.Items[0] = %+v, want gemini ultra starting 2026-04-01", got)
	}
	if got := response.Items[1]; got.Provider != "openai" || got.PlanName != "ChatGPT Plus" {
		t.Fatalf("response.Items[1] = %+v, want openai chatgpt plus", got)
	}
}

func newTestSubscriptionLookupBinding(store *sqlite.Store) *SubscriptionLookupBinding {
	binding := NewSubscriptionLookupBinding(service.NewSubscriptionQueryService(store))
	binding.startup(context.Background())
	return binding
}

func mustSubscriptionForGUIBinding(t *testing.T, provider domain.ProviderName, planName string, fee float64, startsAt time.Time) domain.Subscription {
	t.Helper()
	planCode, err := domain.GenerateSubscriptionPlanCode(provider, planName)
	if err != nil {
		t.Fatalf("GenerateSubscriptionPlanCode() error = %v", err)
	}
	subscriptionID, err := domain.GenerateSubscriptionID(provider, planName, startsAt)
	if err != nil {
		t.Fatalf("GenerateSubscriptionID() error = %v", err)
	}
	subscription, err := domain.NewSubscription(domain.Subscription{
		SubscriptionID: subscriptionID,
		Provider:       provider,
		PlanCode:       planCode,
		PlanName:       planName,
		RenewalDay:     1,
		StartsAt:       startsAt,
		FeeUSD:         fee,
		IsActive:       true,
		CreatedAt:      startsAt,
		UpdatedAt:      startsAt,
	})
	if err != nil {
		t.Fatalf("NewSubscription() error = %v", err)
	}
	return subscription
}
