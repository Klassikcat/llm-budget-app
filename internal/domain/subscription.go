package domain

import (
	"fmt"
	"strings"
	"time"
)

type MonthlyPeriod struct {
	StartAt      time.Time
	EndExclusive time.Time
}

func NewMonthlyPeriod(anchor time.Time) (MonthlyPeriod, error) {
	anchorUTC, err := NormalizeUTCTimestamp("period_anchor", anchor)
	if err != nil {
		return MonthlyPeriod{}, err
	}

	return NewMonthlyPeriodFromParts(anchorUTC.Year(), anchorUTC.Month())
}

func NewMonthlyPeriodFromParts(year int, month time.Month) (MonthlyPeriod, error) {
	if year < 1 {
		return MonthlyPeriod{}, &ValidationError{
			Code:    ValidationCodeInvalidTimestamp,
			Field:   "year",
			Message: "year must be greater than zero",
		}
	}

	if month < time.January || month > time.December {
		return MonthlyPeriod{}, &ValidationError{
			Code:    ValidationCodeInvalidMonth,
			Field:   "month",
			Message: "month must be between January and December",
		}
	}

	start := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	return MonthlyPeriod{
		StartAt:      start,
		EndExclusive: start.AddDate(0, 1, 0),
	}, nil
}

func (p MonthlyPeriod) Contains(value time.Time) bool {
	if value.IsZero() || p.StartAt.IsZero() || p.EndExclusive.IsZero() {
		return false
	}

	valueUTC := value.UTC()
	return !valueUTC.Before(p.StartAt) && valueUTC.Before(p.EndExclusive)
}

func (p MonthlyPeriod) Next() MonthlyPeriod {
	if p.StartAt.IsZero() || p.EndExclusive.IsZero() {
		return MonthlyPeriod{}
	}

	return MonthlyPeriod{
		StartAt:      p.EndExclusive,
		EndExclusive: p.EndExclusive.AddDate(0, 1, 0),
	}
}

type Subscription struct {
	SubscriptionID string
	Provider       ProviderName
	PlanCode       string
	PlanName       string
	RenewalDay     int
	StartsAt       time.Time
	EndsAt         *time.Time
	FeeUSD         float64
	IsActive       bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func NewSubscription(subscription Subscription) (Subscription, error) {
	if strings.TrimSpace(subscription.SubscriptionID) == "" {
		return Subscription{}, requiredError("subscription_id")
	}

	provider, err := NewProviderName(subscription.Provider.String())
	if err != nil {
		return Subscription{}, err
	}
	subscription.Provider = provider

	subscription.PlanCode = strings.TrimSpace(subscription.PlanCode)
	if subscription.PlanCode == "" {
		return Subscription{}, requiredError("plan_code")
	}

	subscription.PlanName = strings.TrimSpace(subscription.PlanName)
	if subscription.PlanName == "" {
		return Subscription{}, requiredError("plan_name")
	}

	if subscription.RenewalDay < 1 || subscription.RenewalDay > 31 {
		return Subscription{}, &ValidationError{
			Code:    ValidationCodeInvalidTimestamp,
			Field:   "renewal_day",
			Message: "renewal day must be between 1 and 31",
		}
	}

	subscription.StartsAt, err = NormalizeUTCTimestamp("starts_at", subscription.StartsAt)
	if err != nil {
		return Subscription{}, err
	}

	if subscription.EndsAt != nil {
		normalizedEndsAt, err := NormalizeUTCTimestamp("ends_at", *subscription.EndsAt)
		if err != nil {
			return Subscription{}, err
		}
		subscription.EndsAt = &normalizedEndsAt
		if subscription.EndsAt.Before(subscription.StartsAt) {
			return Subscription{}, &ValidationError{
				Code:    ValidationCodeInvalidTimeRange,
				Field:   "ends_at",
				Message: "ends_at must be at or after starts_at",
			}
		}
	}

	if !subscription.IsActive && subscription.EndsAt == nil {
		return Subscription{}, &ValidationError{
			Code:    ValidationCodeInvalidTimeRange,
			Field:   "ends_at",
			Message: "inactive subscriptions must include an ends_at timestamp",
		}
	}

	if subscription.FeeUSD < 0 {
		return Subscription{}, &ValidationError{
			Code:    ValidationCodeNegativeCost,
			Field:   "fee_usd",
			Message: "subscription fee must be non-negative",
		}
	}

	subscription.CreatedAt, err = NormalizeUTCTimestamp("created_at", subscription.CreatedAt)
	if err != nil {
		return Subscription{}, err
	}

	subscription.UpdatedAt, err = NormalizeUTCTimestamp("updated_at", subscription.UpdatedAt)
	if err != nil {
		return Subscription{}, err
	}

	if subscription.UpdatedAt.Before(subscription.CreatedAt) {
		return Subscription{}, &ValidationError{
			Code:    ValidationCodeInvalidTimeRange,
			Field:   "updated_at",
			Message: "updated_at must be at or after created_at",
		}
	}

	return subscription, nil
}

func (s Subscription) FeeForPeriod(period MonthlyPeriod) (SubscriptionFee, bool, error) {
	validated, err := NewSubscription(s)
	if err != nil {
		return SubscriptionFee{}, false, err
	}

	chargedAt, ok := validated.chargeAt(period)
	if !ok {
		return SubscriptionFee{}, false, nil
	}

	fee, err := NewSubscriptionFee(SubscriptionFee{
		SubscriptionID: validated.SubscriptionID,
		Provider:       validated.Provider,
		PlanCode:       validated.PlanCode,
		ChargedAt:      chargedAt,
		Period:         period,
		FeeUSD:         validated.FeeUSD,
	})
	if err != nil {
		return SubscriptionFee{}, false, err
	}

	return fee, true, nil
}

func (s Subscription) chargeAt(period MonthlyPeriod) (time.Time, bool) {
	if period.StartAt.IsZero() || period.EndExclusive.IsZero() {
		return time.Time{}, false
	}

	lastDay := period.EndExclusive.AddDate(0, 0, -1).Day()
	day := s.RenewalDay
	if day > lastDay {
		day = lastDay
	}

	chargedAt := time.Date(
		period.StartAt.Year(),
		period.StartAt.Month(),
		day,
		s.StartsAt.Hour(),
		s.StartsAt.Minute(),
		s.StartsAt.Second(),
		s.StartsAt.Nanosecond(),
		time.UTC,
	)

	if chargedAt.Before(s.StartsAt) {
		return time.Time{}, false
	}

	if s.EndsAt != nil && !chargedAt.Before(*s.EndsAt) {
		return time.Time{}, false
	}

	return chargedAt, true
}

func (s Subscription) OverlapsPeriod(period MonthlyPeriod) bool {
	if period.StartAt.IsZero() || period.EndExclusive.IsZero() {
		return false
	}

	if !s.StartsAt.Before(period.EndExclusive) {
		return false
	}

	if s.EndsAt == nil {
		return true
	}

	return s.EndsAt.After(period.StartAt)
}

func (s Subscription) PeriodKey(period MonthlyPeriod) string {
	return fmt.Sprintf("%s:%s", s.SubscriptionID, period.StartAt.Format(time.RFC3339Nano))
}

type SubscriptionFee struct {
	SubscriptionID string
	Provider       ProviderName
	PlanCode       string
	ChargedAt      time.Time
	Period         MonthlyPeriod
	FeeUSD         float64
}

func NewSubscriptionFee(fee SubscriptionFee) (SubscriptionFee, error) {
	if strings.TrimSpace(fee.SubscriptionID) == "" {
		return SubscriptionFee{}, requiredError("subscription_id")
	}

	provider, err := NewProviderName(fee.Provider.String())
	if err != nil {
		return SubscriptionFee{}, err
	}
	fee.Provider = provider

	fee.PlanCode = strings.TrimSpace(fee.PlanCode)
	if fee.PlanCode == "" {
		return SubscriptionFee{}, requiredError("plan_code")
	}

	fee.ChargedAt, err = NormalizeUTCTimestamp("charged_at", fee.ChargedAt)
	if err != nil {
		return SubscriptionFee{}, err
	}

	if fee.FeeUSD < 0 {
		return SubscriptionFee{}, &ValidationError{
			Code:    ValidationCodeNegativeCost,
			Field:   "fee_usd",
			Message: "subscription fee must be non-negative",
		}
	}

	if fee.Period.StartAt.IsZero() || fee.Period.EndExclusive.IsZero() {
		fee.Period, err = NewMonthlyPeriod(fee.ChargedAt)
		if err != nil {
			return SubscriptionFee{}, err
		}
	}

	if !fee.Period.Contains(fee.ChargedAt) {
		return SubscriptionFee{}, &ValidationError{
			Code:    ValidationCodeInvalidTimeRange,
			Field:   "charged_at",
			Message: "charged_at must fall within the subscription monthly period",
		}
	}

	return fee, nil
}
