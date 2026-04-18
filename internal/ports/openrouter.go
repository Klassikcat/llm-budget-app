package ports

import (
	"context"
	"time"

	"llm-budget-tracker/internal/domain"
)

type OpenRouterActivityOptions struct {
	Date       time.Time
	APIKeyHash string
	UserID     string
}

type OpenRouterActivitySource interface {
	FetchUsageEntries(ctx context.Context, options OpenRouterActivityOptions) ([]domain.UsageEntry, error)
}
