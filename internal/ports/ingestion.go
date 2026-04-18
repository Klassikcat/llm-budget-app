package ports

import (
	"context"

	"llm-budget-tracker/internal/domain"
)

type IngestionService interface {
	IngestUsageEntries(ctx context.Context, entries []domain.UsageEntry) error
	IngestSubscriptionFees(ctx context.Context, fees []domain.SubscriptionFee) error
	IngestSessionEvents(ctx context.Context, events []SessionEvent) error
}

type InsightDetector interface {
	Category() domain.DetectorCategory
	Detect(ctx context.Context, period domain.MonthlyPeriod, sessions []domain.SessionSummary, usageEntries []domain.UsageEntry) ([]domain.Insight, error)
}
