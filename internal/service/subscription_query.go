package service

import (
	"context"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type SubscriptionQuery struct {
	Provider domain.ProviderName
	Active   *bool
}

type SubscriptionListSnapshot struct {
	Items []domain.Subscription
	Empty bool
}

type SubscriptionQueryService struct {
	subscriptionRepo ports.SubscriptionRepository
}

func NewSubscriptionQueryService(subscriptionRepo ports.SubscriptionRepository) *SubscriptionQueryService {
	return &SubscriptionQueryService{subscriptionRepo: subscriptionRepo}
}

func (s *SubscriptionQueryService) QuerySubscriptions(ctx context.Context, query SubscriptionQuery) (SubscriptionListSnapshot, error) {
	if s == nil || s.subscriptionRepo == nil {
		return SubscriptionListSnapshot{}, errSubscriptionRepoRequired
	}

	items, err := s.subscriptionRepo.ListSubscriptions(ctx, ports.SubscriptionFilter{
		Provider: query.Provider,
		Active:   query.Active,
	})
	if err != nil {
		return SubscriptionListSnapshot{}, err
	}

	if items == nil {
		items = make([]domain.Subscription, 0)
	}

	return SubscriptionListSnapshot{Items: items, Empty: len(items) == 0}, nil
}
