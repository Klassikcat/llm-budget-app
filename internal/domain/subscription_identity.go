package domain

import (
	"strings"
	"time"
	"unicode"
)

func GenerateSubscriptionPlanCode(provider ProviderName, planName string) (string, error) {
	validatedProvider, err := NewProviderName(provider.String())
	if err != nil {
		return "", err
	}

	planSlug := slugifySubscriptionSegment(planName)
	if planSlug == "" {
		return "", requiredError("plan_name")
	}

	return validatedProvider.String() + "-" + planSlug, nil
}

func GenerateSubscriptionID(provider ProviderName, planName string, startsAt time.Time) (string, error) {
	planCode, err := GenerateSubscriptionPlanCode(provider, planName)
	if err != nil {
		return "", err
	}

	normalizedStartsAt, err := NormalizeUTCTimestamp("starts_at", startsAt)
	if err != nil {
		return "", err
	}

	return planCode + "-" + normalizedStartsAt.Format("2006-01-02"), nil
}

func slugifySubscriptionSegment(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	var builder strings.Builder
	lastHyphen := false
	for _, r := range strings.ToLower(trimmed) {
		switch {
		case unicode.IsLetter(r) || unicode.IsNumber(r):
			builder.WriteRune(r)
			lastHyphen = false
		case lastHyphen:
			continue
		default:
			builder.WriteByte('-')
			lastHyphen = true
		}
	}

	return strings.Trim(builder.String(), "-")
}
