package service

import (
	"context"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type SubscriptionMonthlyRollup struct {
	Period               domain.MonthlyPeriod
	UsageEntries         []domain.UsageEntry
	SubscriptionFees     []domain.SubscriptionFee
	VariableSpendUSD     float64
	SubscriptionSpendUSD float64
	TotalSpendUSD        float64
}

type SubscriptionService struct {
	subscriptionRepo ports.SubscriptionRepository
	usageRepo        ports.UsageEntryRepository
}

func NewSubscriptionService(subscriptionRepo ports.SubscriptionRepository, usageRepo ports.UsageEntryRepository) *SubscriptionService {
	return &SubscriptionService{subscriptionRepo: subscriptionRepo, usageRepo: usageRepo}
}

func (s *SubscriptionService) SaveSubscriptions(ctx context.Context, subscriptions []domain.Subscription) error {
	if s == nil || s.subscriptionRepo == nil {
		return errSubscriptionRepoRequired
	}

	if len(subscriptions) == 0 {
		return nil
	}

	now := time.Now().UTC()
	validated := make([]domain.Subscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		if subscription.CreatedAt.IsZero() {
			subscription.CreatedAt = now
		}
		subscription.UpdatedAt = now

		normalized, err := domain.NewSubscription(subscription)
		if err != nil {
			return err
		}
		validated = append(validated, normalized)
	}

	return s.subscriptionRepo.UpsertSubscriptions(ctx, validated)
}

func (s *SubscriptionService) ListSubscriptions(ctx context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error) {
	if s == nil || s.subscriptionRepo == nil {
		return nil, errSubscriptionRepoRequired
	}

	return s.subscriptionRepo.ListSubscriptions(ctx, filter)
}

func (s *SubscriptionService) DeleteSubscription(ctx context.Context, subscriptionID string) error {
	if s == nil || s.subscriptionRepo == nil {
		return errSubscriptionRepoRequired
	}

	if strings.TrimSpace(subscriptionID) == "" {
		return errSubscriptionIDRequired
	}

	return s.subscriptionRepo.DeleteSubscription(ctx, subscriptionID)
}

func (s *SubscriptionService) DisableSubscription(ctx context.Context, subscriptionID string, disabledAt time.Time) error {
	if s == nil || s.subscriptionRepo == nil {
		return errSubscriptionRepoRequired
	}

	if strings.TrimSpace(subscriptionID) == "" {
		return errSubscriptionIDRequired
	}

	return s.subscriptionRepo.DisableSubscription(ctx, subscriptionID, disabledAt)
}

func (s *SubscriptionService) RollupMonthlySpend(ctx context.Context, period domain.MonthlyPeriod) (SubscriptionMonthlyRollup, error) {
	if s == nil || s.subscriptionRepo == nil {
		return SubscriptionMonthlyRollup{}, errSubscriptionRepoRequired
	}
	if s.usageRepo == nil {
		return SubscriptionMonthlyRollup{}, errUsageEntryRepositoryRequired
	}

	usageEntries, err := s.usageRepo.ListUsageEntries(ctx, ports.UsageFilter{Period: &period})
	if err != nil {
		return SubscriptionMonthlyRollup{}, err
	}

	active := true
	subscriptions, err := s.subscriptionRepo.ListSubscriptions(ctx, ports.SubscriptionFilter{Period: &period, Active: &active})
	if err != nil {
		return SubscriptionMonthlyRollup{}, err
	}

	fees := make([]domain.SubscriptionFee, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		fee, ok, err := subscription.FeeForPeriod(period)
		if err != nil {
			return SubscriptionMonthlyRollup{}, err
		}
		if !ok {
			continue
		}
		fees = append(fees, fee)
	}

	if err := s.subscriptionRepo.UpsertSubscriptionFees(ctx, fees); err != nil {
		return SubscriptionMonthlyRollup{}, err
	}

	persistedFees, err := s.subscriptionRepo.ListSubscriptionFees(ctx, period)
	if err != nil {
		return SubscriptionMonthlyRollup{}, err
	}
	persistedFees = filterSubscriptionFeesByActiveSubscriptions(persistedFees, subscriptions)

	variableSpend := sumVariableUsageSpend(usageEntries)
	subscriptionSpend := sumSubscriptionSpend(persistedFees)

	return SubscriptionMonthlyRollup{
		Period:               period,
		UsageEntries:         usageEntries,
		SubscriptionFees:     persistedFees,
		VariableSpendUSD:     variableSpend,
		SubscriptionSpendUSD: subscriptionSpend,
		TotalSpendUSD:        variableSpend + subscriptionSpend,
	}, nil
}

func filterSubscriptionFeesByActiveSubscriptions(fees []domain.SubscriptionFee, subscriptions []domain.Subscription) []domain.SubscriptionFee {
	if len(fees) == 0 || len(subscriptions) == 0 {
		return make([]domain.SubscriptionFee, 0)
	}

	activeSubscriptionIDs := make(map[string]struct{}, len(subscriptions))
	for _, subscription := range subscriptions {
		activeSubscriptionIDs[subscription.SubscriptionID] = struct{}{}
	}

	filtered := make([]domain.SubscriptionFee, 0, len(fees))
	for _, fee := range fees {
		if _, ok := activeSubscriptionIDs[fee.SubscriptionID]; !ok {
			continue
		}
		filtered = append(filtered, fee)
	}

	return filtered
}
