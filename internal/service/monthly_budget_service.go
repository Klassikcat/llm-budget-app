package service

import (
	"context"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

type MonthlyBudgetService struct {
	budgetRepo ports.BudgetRepository
}

func NewMonthlyBudgetService(budgetRepo ports.BudgetRepository) *MonthlyBudgetService {
	return &MonthlyBudgetService{budgetRepo: budgetRepo}
}

func (s *MonthlyBudgetService) SaveBudgets(ctx context.Context, budgets []domain.MonthlyBudget) error {
	if s == nil || s.budgetRepo == nil {
		return errBudgetRepositoryRequired
	}

	if len(budgets) == 0 {
		return nil
	}

	validated := make([]domain.MonthlyBudget, 0, len(budgets))
	for _, budget := range budgets {
		normalized, err := domain.NewMonthlyBudget(budget)
		if err != nil {
			return err
		}
		validated = append(validated, normalized)
	}

	return s.budgetRepo.UpsertMonthlyBudgets(ctx, validated)
}

func (s *MonthlyBudgetService) ListBudgets(ctx context.Context, filter ports.BudgetFilter) ([]domain.MonthlyBudget, error) {
	if s == nil || s.budgetRepo == nil {
		return nil, errBudgetRepositoryRequired
	}

	return s.budgetRepo.ListMonthlyBudgets(ctx, filter)
}
