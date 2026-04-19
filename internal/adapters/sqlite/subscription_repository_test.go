package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func TestSubscriptionRepositoryRoundTripAndOrdering(t *testing.T) {
	store := mustBootstrapStore(t, filepath.Join(t.TempDir(), "subscription-repo.sqlite3"), Options{})
	defer store.Close()

	startsAt := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	gemini := mustSubscriptionForRepository(t, domain.ProviderGemini, "Gemini Ultra", 249.99, startsAt)
	openai := mustSubscriptionForRepository(t, domain.ProviderOpenAI, "ChatGPT Plus", 20, startsAt)

	if err := store.UpsertSubscriptions(context.Background(), []domain.Subscription{openai, gemini}); err != nil {
		t.Fatalf("UpsertSubscriptions() error = %v", err)
	}

	items, err := store.ListSubscriptions(context.Background(), ports.SubscriptionFilter{})
	if err != nil {
		t.Fatalf("ListSubscriptions() error = %v", err)
	}
	if got := len(items); got != 2 {
		t.Fatalf("len(ListSubscriptions()) = %d, want 2", got)
	}
	if got := items[0].Provider; got != domain.ProviderGemini {
		t.Fatalf("items[0].Provider = %q, want gemini", got)
	}
	if got := items[1].Provider; got != domain.ProviderOpenAI {
		t.Fatalf("items[1].Provider = %q, want openai", got)
	}

	openaiOnly, err := store.ListSubscriptions(context.Background(), ports.SubscriptionFilter{Provider: domain.ProviderOpenAI})
	if err != nil {
		t.Fatalf("ListSubscriptions(openai) error = %v", err)
	}
	if got := len(openaiOnly); got != 1 {
		t.Fatalf("len(ListSubscriptions(openai)) = %d, want 1", got)
	}
	if openaiOnly[0].SubscriptionID == "" || openaiOnly[0].PlanCode == "" {
		t.Fatalf("stored subscription = %+v, want generated id and plan code", openaiOnly[0])
	}
}

func mustSubscriptionForRepository(t *testing.T, provider domain.ProviderName, planName string, fee float64, startsAt time.Time) domain.Subscription {
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
