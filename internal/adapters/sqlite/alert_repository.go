package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

var _ ports.AlertRepository = (*Store)(nil)

func (s *Store) UpsertAlerts(ctx context.Context, alerts []domain.AlertEvent) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(alerts) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO alert_events (
				alert_id, kind, severity, triggered_at, period_start_at, period_end_exclusive,
				budget_id, forecast_id, insight_id, detector_category, current_spend_usd, limit_usd,
				threshold_percent, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(alert_id) DO UPDATE SET
				kind = excluded.kind,
				severity = excluded.severity,
				triggered_at = excluded.triggered_at,
				period_start_at = excluded.period_start_at,
				period_end_exclusive = excluded.period_end_exclusive,
				budget_id = excluded.budget_id,
				forecast_id = excluded.forecast_id,
				insight_id = excluded.insight_id,
				detector_category = excluded.detector_category,
				current_spend_usd = excluded.current_spend_usd,
				limit_usd = excluded.limit_usd,
				threshold_percent = excluded.threshold_percent,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, alert := range alerts {
			normalized, err := domain.NewAlertEvent(alert)
			if err != nil {
				return err
			}
			if _, err := stmt.ExecContext(
				ctx,
				normalized.AlertID,
				string(normalized.Kind),
				string(normalized.Severity),
				formatTime(normalized.TriggeredAt),
				formatTime(normalized.Period.StartAt),
				formatTime(normalized.Period.EndExclusive),
				nullIfBlank(normalized.BudgetID),
				nullIfBlank(normalized.ForecastID),
				nullIfBlank(normalized.InsightID),
				nullIfBlank(string(normalized.DetectorCategory)),
				normalized.CurrentSpendUSD,
				normalized.LimitUSD,
				normalized.ThresholdPercent,
				formatTime(normalized.TriggeredAt),
				formatTime(normalized.TriggeredAt),
			); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) ListAlerts(ctx context.Context, filter ports.AlertFilter) ([]domain.AlertEvent, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT alert_id, kind, severity, triggered_at, period_start_at, period_end_exclusive,
		       budget_id, forecast_id, insight_id, detector_category, current_spend_usd, limit_usd,
		       threshold_percent
		FROM alert_events
		WHERE 1 = 1
	`)
	args := make([]any, 0, 4)

	if filter.Period != nil {
		query.WriteString(` AND period_start_at = ?`)
		args = append(args, formatTime(filter.Period.StartAt))
	}
	if strings.TrimSpace(filter.BudgetID) != "" {
		query.WriteString(` AND budget_id = ?`)
		args = append(args, strings.TrimSpace(filter.BudgetID))
	}
	if filter.Kind != "" {
		query.WriteString(` AND kind = ?`)
		args = append(args, string(filter.Kind))
	}
	query.WriteString(` ORDER BY triggered_at ASC, alert_id ASC`)

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	alerts := make([]domain.AlertEvent, 0)
	for rows.Next() {
		alert, err := scanAlertEvent(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return alerts, nil
}

func scanAlertEvent(scanner interface{ Scan(dest ...any) error }) (domain.AlertEvent, error) {
	var (
		alertID          string
		kindRaw          string
		severityRaw      string
		triggeredAtRaw   string
		periodStartRaw   string
		periodEndRaw     string
		budgetID         sql.NullString
		forecastID       sql.NullString
		insightID        sql.NullString
		detectorCategory sql.NullString
		currentSpendUSD  float64
		limitUSD         float64
		thresholdPercent float64
	)

	if err := scanner.Scan(&alertID, &kindRaw, &severityRaw, &triggeredAtRaw, &periodStartRaw, &periodEndRaw, &budgetID, &forecastID, &insightID, &detectorCategory, &currentSpendUSD, &limitUSD, &thresholdPercent); err != nil {
		return domain.AlertEvent{}, err
	}

	period, err := parseMonthlyPeriod(periodStartRaw, periodEndRaw)
	if err != nil {
		return domain.AlertEvent{}, err
	}
	triggeredAt, err := parseTime(triggeredAtRaw)
	if err != nil {
		return domain.AlertEvent{}, err
	}

	alert, err := domain.NewAlertEvent(domain.AlertEvent{
		AlertID:          alertID,
		Kind:             domain.AlertKind(kindRaw),
		Severity:         domain.AlertSeverity(severityRaw),
		TriggeredAt:      triggeredAt,
		Period:           period,
		BudgetID:         budgetID.String,
		ForecastID:       forecastID.String,
		InsightID:        insightID.String,
		DetectorCategory: domain.DetectorCategory(detectorCategory.String),
		CurrentSpendUSD:  currentSpendUSD,
		LimitUSD:         limitUSD,
		ThresholdPercent: thresholdPercent,
	})
	if err != nil {
		return domain.AlertEvent{}, err
	}

	return alert, nil
}
