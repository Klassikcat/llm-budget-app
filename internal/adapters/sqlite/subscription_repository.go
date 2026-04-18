package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"llm-budget-tracker/internal/domain"
	"llm-budget-tracker/internal/ports"
)

func (s *Store) UpsertSubscriptions(ctx context.Context, subscriptions []domain.Subscription) error {
	if len(subscriptions) == 0 {
		return nil
	}

	validated := make([]domain.Subscription, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		normalized, err := domain.NewSubscription(subscription)
		if err != nil {
			return err
		}
		validated = append(validated, normalized)
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO subscriptions (
				id, provider, plan_code, plan_name, renewal_day, amount_usd,
				starts_at, ends_at, is_active, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(id) DO UPDATE SET
				provider = excluded.provider,
				plan_code = excluded.plan_code,
				plan_name = excluded.plan_name,
				renewal_day = excluded.renewal_day,
				amount_usd = excluded.amount_usd,
				starts_at = excluded.starts_at,
				ends_at = excluded.ends_at,
				is_active = excluded.is_active,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, subscription := range validated {
			_, err := stmt.ExecContext(
				ctx,
				subscription.SubscriptionID,
				subscription.Provider.String(),
				subscription.PlanCode,
				subscription.PlanName,
				subscription.RenewalDay,
				subscription.FeeUSD,
				formatTime(subscription.StartsAt),
				formatNullableTime(subscription.EndsAt),
				boolToInt(subscription.IsActive),
				formatTime(subscription.CreatedAt),
				formatTime(subscription.UpdatedAt),
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) ListSubscriptions(ctx context.Context, filter ports.SubscriptionFilter) ([]domain.Subscription, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	query := strings.Builder{}
	query.WriteString(`
		SELECT id, provider, plan_code, plan_name, renewal_day, amount_usd,
		       starts_at, ends_at, is_active, created_at, updated_at
		FROM subscriptions
		WHERE 1 = 1
	`)
	args := make([]any, 0, 6)

	if filter.SubscriptionID != "" {
		query.WriteString(` AND id = ?`)
		args = append(args, strings.TrimSpace(filter.SubscriptionID))
	}
	if filter.Provider != "" {
		query.WriteString(` AND provider = ?`)
		args = append(args, filter.Provider.String())
	}
	if filter.PlanCode != "" {
		query.WriteString(` AND plan_code = ?`)
		args = append(args, strings.TrimSpace(filter.PlanCode))
	}
	if filter.Active != nil {
		query.WriteString(` AND is_active = ?`)
		args = append(args, boolToInt(*filter.Active))
	}
	if filter.Period != nil {
		query.WriteString(` AND starts_at < ? AND (ends_at IS NULL OR ends_at > ?)`)
		args = append(args, formatTime(filter.Period.EndExclusive), formatTime(filter.Period.StartAt))
	}

	query.WriteString(` ORDER BY provider, plan_code, starts_at`)

	rows, err := s.db.QueryContext(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subscriptions := make([]domain.Subscription, 0)
	for rows.Next() {
		subscription, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subscriptions, nil
}

func (s *Store) DisableSubscription(ctx context.Context, subscriptionID string, disabledAt time.Time) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}

	normalizedDisabledAt, err := domain.NormalizeUTCTimestamp("disabled_at", disabledAt)
	if err != nil {
		return err
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
			UPDATE subscriptions
			SET is_active = 0,
			    ends_at = ?,
			    updated_at = ?
			WHERE id = ?
		`, formatTime(normalizedDisabledAt), formatTime(normalizedDisabledAt), strings.TrimSpace(subscriptionID))
		if err != nil {
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return sql.ErrNoRows
		}

		return nil
	})
}

func (s *Store) UpsertSubscriptionFees(ctx context.Context, fees []domain.SubscriptionFee) error {
	if len(fees) == 0 {
		return nil
	}

	validated := make([]domain.SubscriptionFee, 0, len(fees))
	for _, fee := range fees {
		normalized, err := domain.NewSubscriptionFee(fee)
		if err != nil {
			return err
		}
		validated = append(validated, normalized)
	}

	now := time.Now().UTC()
	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO subscription_fees (
				subscription_id, provider, plan_code, charged_at,
				period_start_at, period_end_exclusive, fee_usd, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(subscription_id, period_start_at) DO UPDATE SET
				provider = excluded.provider,
				plan_code = excluded.plan_code,
				charged_at = excluded.charged_at,
				period_end_exclusive = excluded.period_end_exclusive,
				fee_usd = excluded.fee_usd,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, fee := range validated {
			_, err := stmt.ExecContext(
				ctx,
				fee.SubscriptionID,
				fee.Provider.String(),
				fee.PlanCode,
				formatTime(fee.ChargedAt),
				formatTime(fee.Period.StartAt),
				formatTime(fee.Period.EndExclusive),
				fee.FeeUSD,
				formatTime(now),
				formatTime(now),
			)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *Store) ListSubscriptionFees(ctx context.Context, period domain.MonthlyPeriod) ([]domain.SubscriptionFee, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT subscription_id, provider, plan_code, charged_at,
		       period_start_at, period_end_exclusive, fee_usd
		FROM subscription_fees
		WHERE period_start_at = ?
		ORDER BY provider, plan_code, charged_at
	`, formatTime(period.StartAt))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fees := make([]domain.SubscriptionFee, 0)
	for rows.Next() {
		var (
			subscriptionID string
			providerRaw    string
			planCode       string
			chargedAtRaw   string
			periodStartRaw string
			periodEndRaw   string
			feeUSD         float64
		)
		if err := rows.Scan(&subscriptionID, &providerRaw, &planCode, &chargedAtRaw, &periodStartRaw, &periodEndRaw, &feeUSD); err != nil {
			return nil, err
		}

		provider, err := domain.NewProviderName(providerRaw)
		if err != nil {
			return nil, err
		}
		chargedAt, err := parseTime(chargedAtRaw)
		if err != nil {
			return nil, err
		}
		periodStart, err := parseTime(periodStartRaw)
		if err != nil {
			return nil, err
		}
		periodEnd, err := parseTime(periodEndRaw)
		if err != nil {
			return nil, err
		}

		fee, err := domain.NewSubscriptionFee(domain.SubscriptionFee{
			SubscriptionID: subscriptionID,
			Provider:       provider,
			PlanCode:       planCode,
			ChargedAt:      chargedAt,
			Period:         domain.MonthlyPeriod{StartAt: periodStart, EndExclusive: periodEnd},
			FeeUSD:         feeUSD,
		})
		if err != nil {
			return nil, err
		}
		fees = append(fees, fee)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return fees, nil
}

func scanSubscription(scanner interface{ Scan(dest ...any) error }) (domain.Subscription, error) {
	var (
		subscriptionID string
		providerRaw    string
		planCode       string
		planName       string
		renewalDay     int
		feeUSD         float64
		startsAtRaw    string
		endsAtRaw      sql.NullString
		isActive       int
		createdAtRaw   string
		updatedAtRaw   string
	)

	if err := scanner.Scan(&subscriptionID, &providerRaw, &planCode, &planName, &renewalDay, &feeUSD, &startsAtRaw, &endsAtRaw, &isActive, &createdAtRaw, &updatedAtRaw); err != nil {
		return domain.Subscription{}, err
	}

	provider, err := domain.NewProviderName(providerRaw)
	if err != nil {
		return domain.Subscription{}, err
	}
	startsAt, err := parseTime(startsAtRaw)
	if err != nil {
		return domain.Subscription{}, err
	}
	createdAt, err := parseTime(createdAtRaw)
	if err != nil {
		return domain.Subscription{}, err
	}
	updatedAt, err := parseTime(updatedAtRaw)
	if err != nil {
		return domain.Subscription{}, err
	}
	endsAt, err := parseNullableTime(endsAtRaw)
	if err != nil {
		return domain.Subscription{}, err
	}

	return domain.NewSubscription(domain.Subscription{
		SubscriptionID: subscriptionID,
		Provider:       provider,
		PlanCode:       planCode,
		PlanName:       planName,
		RenewalDay:     renewalDay,
		StartsAt:       startsAt,
		EndsAt:         endsAt,
		FeeUSD:         feeUSD,
		IsActive:       isActive == 1,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	})
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func formatNullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return formatTime(*value)
}

func parseTime(raw string) (time.Time, error) {
	return time.Parse(time.RFC3339Nano, raw)
}

func parseNullableTime(raw sql.NullString) (*time.Time, error) {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil, nil
	}

	parsed, err := parseTime(raw.String)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
