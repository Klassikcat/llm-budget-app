package domain

import "strings"

type BillingMode string

const (
	BillingModeUnknown      BillingMode = "unknown"
	BillingModeSubscription BillingMode = "subscription"
	BillingModeBYOK         BillingMode = "byok"
	BillingModeDirectAPI    BillingMode = "direct_api"
	BillingModeOpenRouter   BillingMode = "openrouter"
)

type UsageSourceKind string

const (
	UsageSourceSubscription UsageSourceKind = "subscription"
	UsageSourceManualAPI    UsageSourceKind = "manual_api"
	UsageSourceOpenRouter   UsageSourceKind = "openrouter"
	UsageSourceCLISession   UsageSourceKind = "cli_session"
)

func ParseBillingMode(raw string) (BillingMode, error) {
	mode := BillingMode(strings.ToLower(strings.TrimSpace(raw)))
	if !mode.IsValid() {
		return "", &ValidationError{
			Code:    ValidationCodeInvalidBillingMode,
			Field:   "billing_mode",
			Message: "billing mode must be one of unknown, subscription, byok, direct_api, or openrouter",
		}
	}

	return mode, nil
}

func (m BillingMode) IsValid() bool {
	switch m {
	case BillingModeUnknown, BillingModeSubscription, BillingModeBYOK, BillingModeDirectAPI, BillingModeOpenRouter:
		return true
	default:
		return false
	}
}

func ParseUsageSourceKind(raw string) (UsageSourceKind, error) {
	kind := UsageSourceKind(strings.ToLower(strings.TrimSpace(raw)))
	if !kind.IsValid() {
		return "", &ValidationError{
			Code:    ValidationCodeInvalidUsageSource,
			Field:   "source",
			Message: "source must be one of subscription, manual_api, openrouter, or cli_session",
		}
	}

	return kind, nil
}

func (k UsageSourceKind) IsValid() bool {
	switch k {
	case UsageSourceSubscription, UsageSourceManualAPI, UsageSourceOpenRouter, UsageSourceCLISession:
		return true
	default:
		return false
	}
}
