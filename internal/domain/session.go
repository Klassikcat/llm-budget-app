package domain

import (
	"strings"
	"time"
)

type SessionSummary struct {
	SessionID     string
	Source        UsageSourceKind
	Provider      ProviderName
	BillingMode   BillingMode
	StartedAt     time.Time
	EndedAt       time.Time
	ProjectName   string
	AgentName     string
	PricingRef    *ModelPricingRef
	Tokens        TokenUsage
	CostBreakdown CostBreakdown
}

func NewSessionSummary(summary SessionSummary) (SessionSummary, error) {
	if strings.TrimSpace(summary.SessionID) == "" {
		return SessionSummary{}, requiredError("session_id")
	}

	if !summary.Source.IsValid() {
		return SessionSummary{}, &ValidationError{
			Code:    ValidationCodeInvalidUsageSource,
			Field:   "source",
			Message: "source must be one of subscription, manual_api, openrouter, or cli_session",
		}
	}

	provider, err := NewProviderName(summary.Provider.String())
	if err != nil {
		return SessionSummary{}, err
	}
	summary.Provider = provider

	if !summary.BillingMode.IsValid() {
		return SessionSummary{}, &ValidationError{
			Code:    ValidationCodeInvalidBillingMode,
			Field:   "billing_mode",
			Message: "billing mode must be one of unknown, subscription, byok, direct_api, or openrouter",
		}
	}

	summary.StartedAt, err = NormalizeUTCTimestamp("started_at", summary.StartedAt)
	if err != nil {
		return SessionSummary{}, err
	}

	summary.EndedAt, err = NormalizeUTCTimestamp("ended_at", summary.EndedAt)
	if err != nil {
		return SessionSummary{}, err
	}

	if summary.EndedAt.Before(summary.StartedAt) {
		return SessionSummary{}, &ValidationError{
			Code:    ValidationCodeInvalidTimeRange,
			Field:   "ended_at",
			Message: "ended_at must be at or after started_at",
		}
	}

	return summary, validateUsageSourceBillingMode(summary.Source, summary.BillingMode)
}

func (s SessionSummary) Duration() time.Duration {
	if s.EndedAt.Before(s.StartedAt) {
		return 0
	}

	return s.EndedAt.Sub(s.StartedAt)
}
