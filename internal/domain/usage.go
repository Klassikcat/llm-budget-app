package domain

import (
	"fmt"
	"strings"
	"time"
)

type ModelPricingRef struct {
	Provider         ProviderName
	ModelID          string
	PricingLookupKey string
}

func NewModelPricingRef(provider ProviderName, modelID, pricingLookupKey string) (ModelPricingRef, error) {
	if _, err := NewProviderName(provider.String()); err != nil {
		return ModelPricingRef{}, err
	}

	trimmedModelID := strings.TrimSpace(modelID)
	if trimmedModelID == "" {
		return ModelPricingRef{}, requiredError("model_id")
	}

	trimmedLookupKey := strings.TrimSpace(pricingLookupKey)
	if trimmedLookupKey == "" {
		trimmedLookupKey = trimmedModelID
	}

	return ModelPricingRef{
		Provider:         provider,
		ModelID:          trimmedModelID,
		PricingLookupKey: trimmedLookupKey,
	}, nil
}

type TokenUsage struct {
	InputTokens      int64
	OutputTokens     int64
	CacheReadTokens  int64
	CacheWriteTokens int64
	TotalTokens      int64
}

func NewTokenUsage(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens int64) (TokenUsage, error) {
	values := map[string]int64{
		"input_tokens":       inputTokens,
		"output_tokens":      outputTokens,
		"cache_read_tokens":  cacheReadTokens,
		"cache_write_tokens": cacheWriteTokens,
	}

	for field, value := range values {
		if value < 0 {
			return TokenUsage{}, &ValidationError{
				Code:    ValidationCodeNegativeTokens,
				Field:   field,
				Message: "token counts must be non-negative",
			}
		}
	}

	total := inputTokens + outputTokens + cacheReadTokens + cacheWriteTokens

	return TokenUsage{
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		CacheReadTokens:  cacheReadTokens,
		CacheWriteTokens: cacheWriteTokens,
		TotalTokens:      total,
	}, nil
}

type CostBreakdown struct {
	InputUSD      float64
	OutputUSD     float64
	CacheReadUSD  float64
	CacheWriteUSD float64
	ToolUSD       float64
	FlatUSD       float64
	TotalUSD      float64
}

func NewCostBreakdown(inputUSD, outputUSD, cacheReadUSD, cacheWriteUSD, toolUSD, flatUSD float64) (CostBreakdown, error) {
	values := map[string]float64{
		"input_usd":       inputUSD,
		"output_usd":      outputUSD,
		"cache_read_usd":  cacheReadUSD,
		"cache_write_usd": cacheWriteUSD,
		"tool_usd":        toolUSD,
		"flat_usd":        flatUSD,
	}

	for field, value := range values {
		if value < 0 {
			return CostBreakdown{}, &ValidationError{
				Code:    ValidationCodeNegativeCost,
				Field:   field,
				Message: "cost values must be non-negative",
			}
		}
	}

	total := inputUSD + outputUSD + cacheReadUSD + cacheWriteUSD + toolUSD + flatUSD

	return CostBreakdown{
		InputUSD:      inputUSD,
		OutputUSD:     outputUSD,
		CacheReadUSD:  cacheReadUSD,
		CacheWriteUSD: cacheWriteUSD,
		ToolUSD:       toolUSD,
		FlatUSD:       flatUSD,
		TotalUSD:      total,
	}, nil
}

type UsageEntry struct {
	EntryID       string
	Source        UsageSourceKind
	Provider      ProviderName
	BillingMode   BillingMode
	OccurredAt    time.Time
	SessionID     string
	ExternalID    string
	ProjectName   string
	AgentName     string
	Metadata      map[string]string
	PricingRef    *ModelPricingRef
	Tokens        TokenUsage
	CostBreakdown CostBreakdown
}

func NewUsageEntry(entry UsageEntry) (UsageEntry, error) {
	if strings.TrimSpace(entry.EntryID) == "" {
		return UsageEntry{}, requiredError("entry_id")
	}

	if !entry.Source.IsValid() {
		return UsageEntry{}, &ValidationError{
			Code:    ValidationCodeInvalidUsageSource,
			Field:   "source",
			Message: "source must be one of subscription, manual_api, openrouter, or cli_session",
		}
	}

	provider, err := NewProviderName(entry.Provider.String())
	if err != nil {
		return UsageEntry{}, err
	}
	entry.Provider = provider

	if !entry.BillingMode.IsValid() {
		return UsageEntry{}, &ValidationError{
			Code:    ValidationCodeInvalidBillingMode,
			Field:   "billing_mode",
			Message: "billing mode must be one of unknown, subscription, byok, direct_api, or openrouter",
		}
	}

	entry.OccurredAt, err = NormalizeUTCTimestamp("occurred_at", entry.OccurredAt)
	if err != nil {
		return UsageEntry{}, err
	}

	if err := validateUsageSourceBillingMode(entry.Source, entry.BillingMode); err != nil {
		return UsageEntry{}, err
	}

	metadata, err := normalizeUsageMetadata(entry.Metadata)
	if err != nil {
		return UsageEntry{}, err
	}
	entry.Metadata = metadata

	return entry, nil
}

func normalizeUsageMetadata(metadata map[string]string) (map[string]string, error) {
	if len(metadata) == 0 {
		return nil, nil
	}

	normalized := make(map[string]string, len(metadata))
	for key, value := range metadata {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, &ValidationError{
				Code:    ValidationCodeInvalidMetadata,
				Field:   "metadata",
				Message: "metadata keys must be non-empty",
			}
		}

		trimmedValue := strings.TrimSpace(value)
		if trimmedValue == "" {
			return nil, &ValidationError{
				Code:    ValidationCodeInvalidMetadata,
				Field:   fmt.Sprintf("metadata.%s", trimmedKey),
				Message: "metadata values must be non-empty",
			}
		}

		normalized[trimmedKey] = trimmedValue
	}

	return normalized, nil
}

func validateUsageSourceBillingMode(source UsageSourceKind, mode BillingMode) error {
	switch source {
	case UsageSourceSubscription:
		if mode != BillingModeSubscription {
			return &ValidationError{Code: ValidationCodeInvalidBillingMode, Field: "billing_mode", Message: "subscription usage must use subscription billing mode"}
		}
	case UsageSourceManualAPI:
		if mode != BillingModeDirectAPI {
			return &ValidationError{Code: ValidationCodeInvalidBillingMode, Field: "billing_mode", Message: "manual API usage must use direct_api billing mode"}
		}
	case UsageSourceOpenRouter:
		if mode != BillingModeOpenRouter {
			return &ValidationError{Code: ValidationCodeInvalidBillingMode, Field: "billing_mode", Message: "OpenRouter usage must use openrouter billing mode"}
		}
	case UsageSourceCLISession:
		return nil
	default:
		return &ValidationError{Code: ValidationCodeInvalidUsageSource, Field: "source", Message: "unsupported usage source"}
	}

	return nil
}
