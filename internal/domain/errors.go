package domain

import (
	"errors"
	"fmt"
)

type ValidationCode string

const (
	ValidationCodeRequired            ValidationCode = "required"
	ValidationCodeInvalidProviderName ValidationCode = "invalid_provider_name"
	ValidationCodeUnsupportedProvider ValidationCode = "unsupported_provider"
	ValidationCodeInvalidBillingMode  ValidationCode = "invalid_billing_mode"
	ValidationCodeInvalidUsageSource  ValidationCode = "invalid_usage_source"
	ValidationCodeUnknownModel        ValidationCode = "unknown_model"
	ValidationCodeInvalidMetadata     ValidationCode = "invalid_metadata"
	ValidationCodeInvalidMonth        ValidationCode = "invalid_month"
	ValidationCodeInvalidThreshold    ValidationCode = "invalid_threshold"
	ValidationCodeInvalidAlertKind    ValidationCode = "invalid_alert_kind"
	ValidationCodeInvalidAlertLevel   ValidationCode = "invalid_alert_level"
	ValidationCodeInvalidDetector     ValidationCode = "invalid_detector"
	ValidationCodeInvalidMetric       ValidationCode = "invalid_metric"
	ValidationCodeInvalidHash         ValidationCode = "invalid_hash"
	ValidationCodeNegativeTokens      ValidationCode = "negative_tokens"
	ValidationCodeNegativeCost        ValidationCode = "negative_cost"
	ValidationCodeInvalidTimestamp    ValidationCode = "invalid_timestamp"
	ValidationCodeInvalidTimeRange    ValidationCode = "invalid_time_range"
)

type ValidationError struct {
	Code    ValidationCode
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Field == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func IsValidationCode(err error, code ValidationCode) bool {
	var validationErr *ValidationError
	if !errors.As(err, &validationErr) {
		return false
	}

	return validationErr.Code == code
}

func requiredError(field string) error {
	return &ValidationError{
		Code:    ValidationCodeRequired,
		Field:   field,
		Message: "value is required",
	}
}
