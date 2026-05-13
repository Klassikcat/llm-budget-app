package gui

import (
	"context"
	"fmt"

	"llm-budget-tracker/internal/service"
)

type subscriptionQuerier interface {
	QuerySubscriptions(ctx context.Context, query service.SubscriptionQuery) (service.SubscriptionListSnapshot, error)
}

type subscriptionDeleter interface {
	DeleteSubscription(ctx context.Context, subscriptionID string) error
}

type SubscriptionLookupBinding struct {
	queryService subscriptionQuerier
	manager      subscriptionDeleter
	ctx          context.Context
}

func NewSubscriptionLookupBinding(queryService subscriptionQuerier, managers ...subscriptionDeleter) *SubscriptionLookupBinding {
	binding := &SubscriptionLookupBinding{queryService: queryService}
	if len(managers) > 0 {
		binding.manager = managers[0]
	}
	return binding
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

func (b *SubscriptionLookupBinding) DeleteSubscription(subscriptionID string) (MutationResponse, error) {
	if b == nil || b.manager == nil {
		return MutationResponse{}, fmt.Errorf("subscription manager is not initialized")
	}
	if err := b.manager.DeleteSubscription(b.context(), subscriptionID); err != nil {
		return failedMutationResult(err), nil
	}
	return successMutationResult(), nil
}

func (b *SubscriptionLookupBinding) context() context.Context {
	ctx := context.Background()
	if b != nil && b.ctx != nil {
		ctx = b.ctx
	}
	return ctx
}

func toSubscriptionListResponse(snapshot service.SubscriptionListSnapshot) SubscriptionListResponse {
	items := make([]SubscriptionState, 0, len(snapshot.Items))
	for _, item := range snapshot.Items {
		items = append(items, toSubscriptionState(item))
	}
	return SubscriptionListResponse{Items: items, Empty: snapshot.Empty}
}
