package service

import (
	"context"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type CostCalculatorService struct {
	catalog ports.PriceCatalog
}

func NewCostCalculatorService(catalog ports.PriceCatalog) *CostCalculatorService {
	return &CostCalculatorService{catalog: catalog}
}

func (s *CostCalculatorService) CalculateUsageEntry(ctx context.Context, entry domain.UsageEntry) (domain.UsageEntry, error) {
	validated, err := domain.NewUsageEntry(entry)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	if validated.PricingRef == nil {
		return validated, nil
	}

	if s == nil || s.catalog == nil {
		return domain.UsageEntry{}, errPriceCatalogRequired
	}

	price, err := s.catalog.LookupModelPrice(ctx, *validated.PricingRef, validated.OccurredAt)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	validated.CostBreakdown, err = price.Calculate(validated.Tokens, 0)
	if err != nil {
		return domain.UsageEntry{}, err
	}

	return domain.NewUsageEntry(validated)
}

func (s *CostCalculatorService) CalculateSessionSummary(ctx context.Context, summary domain.SessionSummary, toolInvocations int64) (domain.SessionSummary, error) {
	validated, err := domain.NewSessionSummary(summary)
	if err != nil {
		return domain.SessionSummary{}, err
	}

	if validated.PricingRef == nil {
		return validated, nil
	}

	if s == nil || s.catalog == nil {
		return domain.SessionSummary{}, errPriceCatalogRequired
	}

	price, err := s.catalog.LookupModelPrice(ctx, *validated.PricingRef, validated.StartedAt)
	if err != nil {
		return domain.SessionSummary{}, err
	}

	validated.CostBreakdown, err = price.Calculate(validated.Tokens, toolInvocations)
	if err != nil {
		return domain.SessionSummary{}, err
	}

	return domain.NewSessionSummary(validated)
}
