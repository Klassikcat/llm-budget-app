package ports

import (
	"context"

	"llm-budget-tracker/internal/domain"
)

type AlertNotifier interface {
	NotifyAlert(ctx context.Context, alert domain.AlertEvent) error
}
