package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

var _ ports.ForecastRepository = (*Store)(nil)

func (s *Store) UpsertForecastSnapshots(ctx context.Context, forecasts []domain.ForecastSnapshot) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(forecasts) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO forecast_snapshots (
				forecast_id, period_start_at, period_end_exclusive, generated_at, actual_spend_usd,
				forecast_spend_usd, budget_limit_usd, projected_overrun_usd, observed_day_count, remaining_day_count,
				created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(forecast_id) DO UPDATE SET
				period_start_at = excluded.period_start_at,
				period_end_exclusive = excluded.period_end_exclusive,
				generated_at = excluded.generated_at,
				actual_spend_usd = excluded.actual_spend_usd,
				forecast_spend_usd = excluded.forecast_spend_usd,
				budget_limit_usd = excluded.budget_limit_usd,
				projected_overrun_usd = excluded.projected_overrun_usd,
				observed_day_count = excluded.observed_day_count,
				remaining_day_count = excluded.remaining_day_count,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, forecast := range forecasts {
			normalized, err := domain.NewForecastSnapshot(forecast)
			if err != nil {
				return err
			}
			if _, err := stmt.ExecContext(
				ctx,
				normalized.ForecastID,
				formatTime(normalized.Period.StartAt),
				formatTime(normalized.Period.EndExclusive),
				formatTime(normalized.GeneratedAt),
				normalized.ActualSpendUSD,
				normalized.ForecastSpendUSD,
				normalized.BudgetLimitUSD,
				normalized.ProjectedOverrunUSD,
				normalized.ObservedDayCount,
				normalized.RemainingDayCount,
				formatTime(normalized.GeneratedAt),
				formatTime(normalized.GeneratedAt),
			); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) ListForecastSnapshots(ctx context.Context, period domain.MonthlyPeriod) ([]domain.ForecastSnapshot, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT forecast_id, period_start_at, period_end_exclusive, generated_at, actual_spend_usd,
		       forecast_spend_usd, budget_limit_usd, projected_overrun_usd, observed_day_count, remaining_day_count
		FROM forecast_snapshots
		WHERE period_start_at = ?
		ORDER BY forecast_id
	`, formatTime(period.StartAt))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	forecasts := make([]domain.ForecastSnapshot, 0)
	for rows.Next() {
		forecast, err := scanForecastSnapshot(rows)
		if err != nil {
			return nil, err
		}
		forecasts = append(forecasts, forecast)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return forecasts, nil
}

func scanForecastSnapshot(scanner interface{ Scan(dest ...any) error }) (domain.ForecastSnapshot, error) {
	var (
		forecastID          string
		periodStartRaw      string
		periodEndRaw        string
		generatedAtRaw      string
		actualSpendUSD      float64
		forecastSpendUSD    float64
		budgetLimitUSD      float64
		projectedOverrunUSD float64
		observedDayCount    int
		remainingDayCount   int
	)

	if err := scanner.Scan(&forecastID, &periodStartRaw, &periodEndRaw, &generatedAtRaw, &actualSpendUSD, &forecastSpendUSD, &budgetLimitUSD, &projectedOverrunUSD, &observedDayCount, &remainingDayCount); err != nil {
		return domain.ForecastSnapshot{}, err
	}

	period, err := parseMonthlyPeriod(periodStartRaw, periodEndRaw)
	if err != nil {
		return domain.ForecastSnapshot{}, err
	}
	generatedAt, err := parseTime(generatedAtRaw)
	if err != nil {
		return domain.ForecastSnapshot{}, err
	}

	forecast, err := domain.NewForecastSnapshot(domain.ForecastSnapshot{
		ForecastID:          forecastID,
		Period:              period,
		GeneratedAt:         generatedAt,
		ActualSpendUSD:      actualSpendUSD,
		ForecastSpendUSD:    forecastSpendUSD,
		BudgetLimitUSD:      budgetLimitUSD,
		ProjectedOverrunUSD: projectedOverrunUSD,
		ObservedDayCount:    observedDayCount,
		RemainingDayCount:   remainingDayCount,
	})
	if err != nil {
		return domain.ForecastSnapshot{}, err
	}

	forecast.ProjectedOverrunUSD = projectedOverrunUSD
	return forecast, nil
}
