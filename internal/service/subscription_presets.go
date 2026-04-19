package service

import (
	"strings"

	"llm-budget-tracker/internal/domain"
)

type SubscriptionPreset struct {
	Key               string
	Provider          domain.ProviderName
	PlanName          string
	DefaultFeeUSD     float64
	DefaultRenewalDay int
}

var subscriptionPresets = []SubscriptionPreset{
	{Key: "chatgpt-plus", Provider: domain.ProviderOpenAI, PlanName: "ChatGPT Plus", DefaultFeeUSD: 20, DefaultRenewalDay: 1},
	{Key: "chatgpt-pro-5x", Provider: domain.ProviderOpenAI, PlanName: "ChatGPT Pro 5x", DefaultFeeUSD: 100, DefaultRenewalDay: 1},
	{Key: "chatgpt-pro-20x", Provider: domain.ProviderOpenAI, PlanName: "ChatGPT Pro 20x", DefaultFeeUSD: 200, DefaultRenewalDay: 1},
	{Key: "claude-pro", Provider: domain.ProviderClaude, PlanName: "Claude Pro", DefaultFeeUSD: 20, DefaultRenewalDay: 1},
	{Key: "claude-max-5x", Provider: domain.ProviderClaude, PlanName: "Claude Max 5x", DefaultFeeUSD: 100, DefaultRenewalDay: 1},
	{Key: "claude-max-20x", Provider: domain.ProviderClaude, PlanName: "Claude Max 20x", DefaultFeeUSD: 200, DefaultRenewalDay: 1},
	{Key: "gemini-plus", Provider: domain.ProviderGemini, PlanName: "Gemini Plus", DefaultFeeUSD: 7.99, DefaultRenewalDay: 1},
	{Key: "gemini-pro", Provider: domain.ProviderGemini, PlanName: "Gemini Pro", DefaultFeeUSD: 19.99, DefaultRenewalDay: 1},
	{Key: "gemini-ultra", Provider: domain.ProviderGemini, PlanName: "Gemini Ultra", DefaultFeeUSD: 249.99, DefaultRenewalDay: 1},
}

func ListSubscriptionPresets() []SubscriptionPreset {
	items := make([]SubscriptionPreset, len(subscriptionPresets))
	copy(items, subscriptionPresets)
	return items
}

func ResolveSubscriptionPreset(key string) (SubscriptionPreset, error) {
	trimmed := strings.TrimSpace(strings.ToLower(key))
	if trimmed == "" {
		return SubscriptionPreset{}, &domain.ValidationError{
			Code:    domain.ValidationCodeRequired,
			Field:   "preset_key",
			Message: "value is required",
		}
	}

	for _, preset := range subscriptionPresets {
		if preset.Key == trimmed {
			return preset, nil
		}
	}

	return SubscriptionPreset{}, &domain.ValidationError{
		Code:    domain.ValidationCodeInvalidPreset,
		Field:   "preset_key",
		Message: "unknown subscription preset",
	}
}
