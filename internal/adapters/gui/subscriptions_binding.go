package gui

import (
	"context"
	"fmt"

	"llm-budget-tracker/internal/service"
)

type subscriptionQuerier interface {
	QuerySubscriptions(ctx context.Context, query service.SubscriptionQuery) (service.SubscriptionListSnapshot, error)
}

type SubscriptionLookupBinding struct {
	queryService subscriptionQuerier
	ctx          context.Context
}

func NewSubscriptionLookupBinding(queryService subscriptionQuerier) *SubscriptionLookupBinding {
	return &SubscriptionLookupBinding{queryService: queryService}
}

func (b *SubscriptionLookupBinding) startup(ctx context.Context) {
	if b == nil {
		return
	}
	b.ctx = ctx
}

func (b *SubscriptionLookupBinding) LoadSubscriptions() (SubscriptionListResponse, error) {
	if b == nil || b.queryService == nil {
		return SubscriptionListResponse{}, fmt.Errorf("subscription query service is not initialized")
	}

	ctx := context.Background()
	if b.ctx != nil {
		ctx = b.ctx
	}

	snapshot, err := b.queryService.QuerySubscriptions(ctx, service.SubscriptionQuery{})
	if err != nil {
		return SubscriptionListResponse{}, err
	}

	return toSubscriptionListResponse(snapshot), nil
}

func toSubscriptionListResponse(snapshot service.SubscriptionListSnapshot) SubscriptionListResponse {
	items := make([]SubscriptionState, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		items = append(items, toSubscriptionState(item))
	}
	return SubscriptionListResponse{Items: items, Empty: snapshot.Empty}
}
