package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

var _ ports.InsightRepository = (*Store)(nil)

func (s *Store) UpsertInsights(ctx context.Context, insights []domain.Insight) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(insights) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO insights (
				insight_id,
				category,
				severity,
				detected_at,
				period_start_at,
				period_end_exclusive,
				payload_json,
				created_at,
				updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(insight_id) DO UPDATE SET
				category = excluded.category,
				severity = excluded.severity,
				detected_at = excluded.detected_at,
				period_start_at = excluded.period_start_at,
				period_end_exclusive = excluded.period_end_exclusive,
				payload_json = excluded.payload_json,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, insight := range insights {
			validated, err := domain.NewInsight(insight)
			if err != nil {
				return err
			}

			payloadJSON, err := json.Marshal(validated.Payload)
			if err != nil {
				return fmt.Errorf("marshal insight payload %s: %w", validated.InsightID, err)
			}

			now := time.Now().UTC().Format(time.RFC3339Nano)
			if _, err := stmt.ExecContext(ctx,
				validated.InsightID,
				string(validated.Category),
				string(validated.Severity),
				validated.DetectedAt.Format(time.RFC3339Nano),
				validated.Period.StartAt.Format(time.RFC3339Nano),
				validated.Period.EndExclusive.Format(time.RFC3339Nano),
				string(payloadJSON),
				now,
				now,
			); err != nil {
				return fmt.Errorf("upsert insight %s: %w", validated.InsightID, err)
			}
		}

		return nil
	})
}

func (s *Store) ListInsights(ctx context.Context, period domain.MonthlyPeriod) ([]domain.Insight, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT insight_id, category, severity, detected_at, period_start_at, period_end_exclusive, payload_json
		FROM insights
		WHERE period_start_at = ? AND period_end_exclusive = ?
		ORDER BY detected_at ASC, insight_id ASC
	`, period.StartAt.Format(time.RFC3339Nano), period.EndExclusive.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("list insights: %w", err)
	}
	defer rows.Close()

	insights := make([]domain.Insight, 0)
	for rows.Next() {
		insight, err := scanInsight(rows)
		if err != nil {
			return nil, err
		}
		insights = append(insights, insight)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate insights: %w", err)
	}

	return insights, nil
}

func scanInsight(scanner interface{ Scan(dest ...any) error }) (domain.Insight, error) {
	var (
		insightID      string
		categoryRaw    string
		severityRaw    string
		detectedAtRaw  string
		periodStartRaw string
		periodEndRaw   string
		payloadJSON    string
	)

	if err := scanner.Scan(
		&insightID,
		&categoryRaw,
		&severityRaw,
		&detectedAtRaw,
		&periodStartRaw,
		&periodEndRaw,
		&payloadJSON,
	); err != nil {
		return domain.Insight{}, fmt.Errorf("scan insight: %w", err)
	}

	detectedAt, err := time.Parse(time.RFC3339Nano, detectedAtRaw)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("parse insight detected_at: %w", err)
	}
	periodStart, err := time.Parse(time.RFC3339Nano, periodStartRaw)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("parse insight period_start_at: %w", err)
	}
	periodEnd, err := time.Parse(time.RFC3339Nano, periodEndRaw)
	if err != nil {
		return domain.Insight{}, fmt.Errorf("parse insight period_end_exclusive: %w", err)
	}

	var payload domain.InsightPayload
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		return domain.Insight{}, fmt.Errorf("unmarshal insight payload: %w", err)
	}

	return domain.NewInsight(domain.Insight{
		InsightID:  insightID,
		Category:   domain.DetectorCategory(categoryRaw),
		Severity:   domain.InsightSeverity(severityRaw),
		DetectedAt: detectedAt,
		Period: domain.MonthlyPeriod{
			StartAt:      periodStart,
			EndExclusive: periodEnd,
		},
		Payload: payload,
	})
}
