package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

var _ ports.BudgetRepository = (*Store)(nil)

func (s *Store) UpsertMonthlyBudgets(ctx context.Context, budgets []domain.MonthlyBudget) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(budgets) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO monthly_budgets (
				budget_id, name, provider, project_hash, period_start_at, period_end_exclusive,
				limit_usd, currency, thresholds_json, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(budget_id, period_start_at) DO UPDATE SET
				name = excluded.name,
				provider = excluded.provider,
				project_hash = excluded.project_hash,
				period_end_exclusive = excluded.period_end_exclusive,
				limit_usd = excluded.limit_usd,
				currency = excluded.currency,
				thresholds_json = excluded.thresholds_json,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		now := time.Now().UTC()
		for _, budget := range budgets {
			normalized, err := domain.NewMonthlyBudget(budget)
			if err != nil {
				return err
			}
			thresholdsJSON, err := marshalBudgetThresholds(normalized.Thresholds)
			if err != nil {
				return err
			}

			if _, err := stmt.ExecContext(
				ctx,
				normalized.BudgetID,
				nullIfBlank(normalized.Name),
				nullIfBlank(normalized.Provider.String()),
				nullIfBlank(normalized.ProjectHash),
				formatTime(normalized.Period.StartAt),
				formatTime(normalized.Period.EndExclusive),
				normalized.LimitUSD,
				normalized.Currency,
				thresholdsJSON,
				formatTime(now),
				formatTime(now),
			); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) ListMonthlyBudgets(ctx context.Context, filter ports.BudgetFilter) ([]domain.MonthlyBudget, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT budget_id, name, provider, project_hash, period_start_at, period_end_exclusive,
		       limit_usd, currency, thresholds_json
		FROM monthly_budgets
		WHERE 1 = 1
	`)
	args := make([]any, 0, 4)

	if filter.Period != nil {
		query.WriteString(` AND period_start_at = ?`)
		args = append(args, formatTime(filter.Period.StartAt))
	}
	if filter.Provider != "" {
		query.WriteString(` AND provider = ?`)
		args = append(args, filter.Provider.String())
	}
	query.WriteString(` ORDER BY period_start_at, budget_id`)

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	budgets := make([]domain.MonthlyBudget, 0)
	for rows.Next() {
		budget, err := scanMonthlyBudget(rows)
		if err != nil {
			return nil, err
		}
		budgets = append(budgets, budget)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return budgets, nil
}

func (s *Store) UpsertBudgetStates(ctx context.Context, states []domain.BudgetState) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(states) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO budget_states (
				budget_id, period_start_at, period_end_exclusive, current_spend_usd, forecast_spend_usd,
				triggered_thresholds_json, budget_overrun_active, forecast_overrun_active, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(budget_id, period_start_at) DO UPDATE SET
				period_end_exclusive = excluded.period_end_exclusive,
				current_spend_usd = excluded.current_spend_usd,
				forecast_spend_usd = excluded.forecast_spend_usd,
				triggered_thresholds_json = excluded.triggered_thresholds_json,
				budget_overrun_active = excluded.budget_overrun_active,
				forecast_overrun_active = excluded.forecast_overrun_active,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, state := range states {
			normalized, err := domain.NewBudgetState(state)
			if err != nil {
				return err
			}
			thresholdsJSON, err := marshalTriggeredThresholdPercents(normalized.TriggeredThresholdPercents)
			if err != nil {
				return err
			}

			if _, err := stmt.ExecContext(
				ctx,
				normalized.BudgetID,
				formatTime(normalized.Period.StartAt),
				formatTime(normalized.Period.EndExclusive),
				normalized.CurrentSpendUSD,
				normalized.ForecastSpendUSD,
				thresholdsJSON,
				boolToInt(normalized.BudgetOverrunActive),
				boolToInt(normalized.ForecastOverrunActive),
				formatTime(normalized.UpdatedAt),
			); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) GetBudgetState(ctx context.Context, budgetID string, period domain.MonthlyPeriod) (domain.BudgetState, bool, error) {
	if s == nil || s.db == nil {
		return domain.BudgetState{}, false, fmt.Errorf("sqlite store is not initialized")
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT budget_id, period_start_at, period_end_exclusive, current_spend_usd, forecast_spend_usd,
		       triggered_thresholds_json, budget_overrun_active, forecast_overrun_active, updated_at
		FROM budget_states
		WHERE budget_id = ? AND period_start_at = ?
	`, strings.TrimSpace(budgetID), formatTime(period.StartAt))

	state, err := scanBudgetState(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.BudgetState{}, false, nil
		}
		return domain.BudgetState{}, false, err
	}

	return state, true, nil
}

func scanMonthlyBudget(scanner interface{ Scan(dest ...any) error }) (domain.MonthlyBudget, error) {
	var (
		budgetID       string
		name           sql.NullString
		providerRaw    sql.NullString
		projectHash    sql.NullString
		periodStartRaw string
		periodEndRaw   string
		limitUSD       float64
		currency       string
		thresholdsJSON string
	)

	if err := scanner.Scan(&budgetID, &name, &providerRaw, &projectHash, &periodStartRaw, &periodEndRaw, &limitUSD, &currency, &thresholdsJSON); err != nil {
		return domain.MonthlyBudget{}, err
	}

	period, err := parseMonthlyPeriod(periodStartRaw, periodEndRaw)
	if err != nil {
		return domain.MonthlyBudget{}, err
	}
	thresholds, err := unmarshalBudgetThresholds(thresholdsJSON)
	if err != nil {
		return domain.MonthlyBudget{}, err
	}

	var provider domain.ProviderName
	if providerRaw.Valid && strings.TrimSpace(providerRaw.String) != "" {
		provider, err = domain.NewProviderName(providerRaw.String)
		if err != nil {
			return domain.MonthlyBudget{}, err
		}
	}

	return domain.NewMonthlyBudget(domain.MonthlyBudget{
		BudgetID:    budgetID,
		Name:        name.String,
		Provider:    provider,
		ProjectHash: projectHash.String,
		Period:      period,
		LimitUSD:    limitUSD,
		Currency:    currency,
		Thresholds:  thresholds,
	})
}

func scanBudgetState(scanner interface{ Scan(dest ...any) error }) (domain.BudgetState, error) {
	var (
		budgetID              string
		periodStartRaw        string
		periodEndRaw          string
		currentSpendUSD       float64
		forecastSpendUSD      float64
		thresholdsJSON        string
		budgetOverrunActive   int
		forecastOverrunActive int
		updatedAtRaw          string
	)

	if err := scanner.Scan(&budgetID, &periodStartRaw, &periodEndRaw, &currentSpendUSD, &forecastSpendUSD, &thresholdsJSON, &budgetOverrunActive, &forecastOverrunActive, &updatedAtRaw); err != nil {
		return domain.BudgetState{}, err
	}

	period, err := parseMonthlyPeriod(periodStartRaw, periodEndRaw)
	if err != nil {
		return domain.BudgetState{}, err
	}
	updatedAt, err := parseTime(updatedAtRaw)
	if err != nil {
		return domain.BudgetState{}, err
	}
	thresholds, err := unmarshalTriggeredThresholdPercents(thresholdsJSON)
	if err != nil {
		return domain.BudgetState{}, err
	}

	return domain.NewBudgetState(domain.BudgetState{
		BudgetID:                   budgetID,
		Period:                     period,
		CurrentSpendUSD:            currentSpendUSD,
		ForecastSpendUSD:           forecastSpendUSD,
		TriggeredThresholdPercents: thresholds,
		BudgetOverrunActive:        budgetOverrunActive != 0,
		ForecastOverrunActive:      forecastOverrunActive != 0,
		UpdatedAt:                  updatedAt,
	})
}

func marshalBudgetThresholds(thresholds []domain.BudgetThreshold) (string, error) {
	encoded, err := json.Marshal(thresholds)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func unmarshalBudgetThresholds(raw string) ([]domain.BudgetThreshold, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var thresholds []domain.BudgetThreshold
	if err := json.Unmarshal([]byte(raw), &thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}

func marshalTriggeredThresholdPercents(thresholds []float64) (string, error) {
	encoded, err := json.Marshal(thresholds)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func unmarshalTriggeredThresholdPercents(raw string) ([]float64, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var thresholds []float64
	if err := json.Unmarshal([]byte(raw), &thresholds); err != nil {
		return nil, err
	}
	return thresholds, nil
}

func parseMonthlyPeriod(startRaw, endRaw string) (domain.MonthlyPeriod, error) {
	startAt, err := parseTime(startRaw)
	if err != nil {
		return domain.MonthlyPeriod{}, err
	}
	endExclusive, err := parseTime(endRaw)
	if err != nil {
		return domain.MonthlyPeriod{}, err
	}
	return domain.MonthlyPeriod{StartAt: startAt, EndExclusive: endExclusive}, nil
}
