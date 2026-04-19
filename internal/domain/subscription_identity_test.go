package domain

import (
	"testing"
	"time"
)

func TestGenerateSubscriptionPlanCode(t *testing.T) {
	t.Parallel()

	got, err := GenerateSubscriptionPlanCode(ProviderOpenAI, "ChatGPT Pro 20x")
	if err != nil {
		t.Fatalf("GenerateSubscriptionPlanCode() error = %v", err)
	}

	if want := "openai-chatgpt-pro-20x"; got != want {
		t.Fatalf("GenerateSubscriptionPlanCode() = %q, want %q", got, want)
	}
}

func TestGenerateSubscriptionIDDeterministicByNormalizedDate(t *testing.T) {
	t.Parallel()

	startsAt := time.Date(2026, time.April, 1, 0, 0, 0, 0, time.UTC)
	got, err := GenerateSubscriptionID(ProviderClaude, "Claude Max 5x", startsAt)
	if err != nil {
		t.Fatalf("GenerateSubscriptionID() error = %v", err)
	}

	if want := "claude-claude-max-5x-2026-04-01"; got != want {
		t.Fatalf("GenerateSubscriptionID() = %q, want %q", got, want)
	}

	again, err := GenerateSubscriptionID(ProviderClaude, "Claude Max 5x", startsAt)
	if err != nil {
		t.Fatalf("GenerateSubscriptionID() second call error = %v", err)
	}
	if again != got {
		t.Fatalf("GenerateSubscriptionID() deterministic mismatch = %q, want %q", again, got)
	}
}
