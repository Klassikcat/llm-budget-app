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

var _ ports.SessionRepository = (*Store)(nil)

func (s *Store) UpsertSessions(ctx context.Context, sessions []domain.SessionSummary) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("sqlite store is not initialized")
	}
	if len(sessions) == 0 {
		return nil
	}

	return s.WithTx(ctx, nil, func(tx *sql.Tx) error {
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO sessions (
				session_id, source_type, provider, tool_name, billing_mode, project_name,
				model_name, pricing_lookup_key, started_at, ended_at,
				input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
				input_cost_usd, output_cost_usd, cache_creation_cost_usd, cache_read_cost_usd,
				tool_cost_usd, flat_cost_usd, total_cost_usd, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(session_id) DO UPDATE SET
				source_type = excluded.source_type,
				provider = excluded.provider,
				tool_name = excluded.tool_name,
				billing_mode = excluded.billing_mode,
				project_name = excluded.project_name,
				model_name = excluded.model_name,
				pricing_lookup_key = excluded.pricing_lookup_key,
				started_at = excluded.started_at,
				ended_at = excluded.ended_at,
				input_tokens = excluded.input_tokens,
				output_tokens = excluded.output_tokens,
				cache_creation_tokens = excluded.cache_creation_tokens,
				cache_read_tokens = excluded.cache_read_tokens,
				input_cost_usd = excluded.input_cost_usd,
				output_cost_usd = excluded.output_cost_usd,
				cache_creation_cost_usd = excluded.cache_creation_cost_usd,
				cache_read_cost_usd = excluded.cache_read_cost_usd,
				tool_cost_usd = excluded.tool_cost_usd,
				flat_cost_usd = excluded.flat_cost_usd,
				total_cost_usd = excluded.total_cost_usd,
				updated_at = excluded.updated_at
		`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, session := range sessions {
			validated, err := domain.NewSessionSummary(session)
			if err != nil {
				return err
			}

			now := time.Now().UTC().Format(time.RFC3339Nano)
			if _, err := stmt.ExecContext(ctx,
				validated.SessionID,
				string(validated.Source),
				validated.Provider.String(),
				nullIfBlank(validated.AgentName),
				string(validated.BillingMode),
				nullIfBlank(validated.ProjectName),
				nullIfBlank(modelID(validated.PricingRef)),
				nullIfBlank(pricingLookupKey(validated.PricingRef)),
				validated.StartedAt.Format(time.RFC3339Nano),
				validated.EndedAt.Format(time.RFC3339Nano),
				validated.Tokens.InputTokens,
				validated.Tokens.OutputTokens,
				validated.Tokens.CacheWriteTokens,
				validated.Tokens.CacheReadTokens,
				validated.CostBreakdown.InputUSD,
				validated.CostBreakdown.OutputUSD,
				validated.CostBreakdown.CacheWriteUSD,
				validated.CostBreakdown.CacheReadUSD,
				validated.CostBreakdown.ToolUSD,
				validated.CostBreakdown.FlatUSD,
				validated.CostBreakdown.TotalUSD,
				now,
				now,
			); err != nil {
				return fmt.Errorf("upsert session %s: %w", validated.SessionID, err)
			}
		}

		return nil
	})
}

func (s *Store) ListSessions(ctx context.Context, filter ports.SessionFilter) ([]domain.SessionSummary, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("sqlite store is not initialized")
	}

	query := `
		SELECT
			session_id, source_type, provider, tool_name, billing_mode, project_name,
			model_name, pricing_lookup_key, started_at, ended_at,
			input_tokens, output_tokens, cache_creation_tokens, cache_read_tokens,
			input_cost_usd, output_cost_usd, cache_creation_cost_usd, cache_read_cost_usd,
			tool_cost_usd, flat_cost_usd, total_cost_usd
		FROM sessions
		WHERE 1=1`
	args := make([]any, 0, 8)

	if filter.Period != nil {
		query += ` AND started_at < ? AND ended_at >= ?`
		args = append(args, filter.Period.EndExclusive.Format(time.RFC3339Nano), filter.Period.StartAt.Format(time.RFC3339Nano))
	}
	if filter.Provider != "" {
		query += ` AND provider = ?`
		args = append(args, filter.Provider.String())
	}
	if strings.TrimSpace(filter.Project) != "" {
		query += ` AND project_name = ?`
		args = append(args, strings.TrimSpace(filter.Project))
	}
	if strings.TrimSpace(filter.Agent) != "" {
		query += ` AND tool_name = ?`
		args = append(args, strings.TrimSpace(filter.Agent))
	}
	if strings.TrimSpace(filter.SessionID) != "" {
		query += ` AND session_id = ?`
		args = append(args, strings.TrimSpace(filter.SessionID))
	}

	query += ` ORDER BY started_at ASC, session_id ASC`

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	results := make([]domain.SessionSummary, 0)
	for rows.Next() {
		summary, err := scanSessionSummary(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return results, nil
}

func scanSessionSummary(scanner interface{ Scan(dest ...any) error }) (domain.SessionSummary, error) {
	var (
		sessionID         string
		sourceRaw         string
		providerRaw       string
		agentName         sql.NullString
		billingModeRaw    string
		projectName       sql.NullString
		modelName         sql.NullString
		pricingLookup     sql.NullString
		startedAtRaw      string
		endedAtRaw        string
		inputTokens       int64
		outputTokens      int64
		cacheWriteTokens  int64
		cacheReadTokens   int64
		inputCostUSD      float64
		outputCostUSD     float64
		cacheWriteCostUSD float64
		cacheReadCostUSD  float64
		toolCostUSD       float64
		flatCostUSD       float64
		totalCostUSD      float64
	)

	if err := scanner.Scan(
		&sessionID,
		&sourceRaw,
		&providerRaw,
		&agentName,
		&billingModeRaw,
		&projectName,
		&modelName,
		&pricingLookup,
		&startedAtRaw,
		&endedAtRaw,
		&inputTokens,
		&outputTokens,
		&cacheWriteTokens,
		&cacheReadTokens,
		&inputCostUSD,
		&outputCostUSD,
		&cacheWriteCostUSD,
		&cacheReadCostUSD,
		&toolCostUSD,
		&flatCostUSD,
		&totalCostUSD,
	); err != nil {
		return domain.SessionSummary{}, fmt.Errorf("scan session summary: %w", err)
	}

	provider, err := domain.NewProviderName(providerRaw)
	if err != nil {
		return domain.SessionSummary{}, err
	}
	source, err := domain.ParseUsageSourceKind(sourceRaw)
	if err != nil {
		return domain.SessionSummary{}, err
	}
	billingMode, err := domain.ParseBillingMode(billingModeRaw)
	if err != nil {
		return domain.SessionSummary{}, err
	}
	startedAt, err := time.Parse(time.RFC3339Nano, startedAtRaw)
	if err != nil {
		return domain.SessionSummary{}, fmt.Errorf("parse session start time: %w", err)
	}
	endedAt, err := time.Parse(time.RFC3339Nano, endedAtRaw)
	if err != nil {
		return domain.SessionSummary{}, fmt.Errorf("parse session end time: %w", err)
	}
	tokens, err := domain.NewTokenUsage(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens)
	if err != nil {
		return domain.SessionSummary{}, err
	}
	costBreakdown, err := domain.NewCostBreakdown(inputCostUSD, outputCostUSD, cacheReadCostUSD, cacheWriteCostUSD, toolCostUSD, flatCostUSD)
	if err != nil {
		return domain.SessionSummary{}, err
	}
	if totalCostUSD != costBreakdown.TotalUSD {
		costBreakdown.TotalUSD = totalCostUSD
	}

	var pricingRef *domain.ModelPricingRef
	if modelName.Valid {
		ref, err := domain.NewModelPricingRef(provider, modelName.String, pricingLookup.String)
		if err != nil {
			return domain.SessionSummary{}, err
		}
		pricingRef = &ref
	}

	return domain.NewSessionSummary(domain.SessionSummary{
		SessionID:     sessionID,
		Source:        source,
		Provider:      provider,
		BillingMode:   billingMode,
		StartedAt:     startedAt,
		EndedAt:       endedAt,
		ProjectName:   projectName.String,
		AgentName:     agentName.String,
		PricingRef:    pricingRef,
		Tokens:        tokens,
		CostBreakdown: costBreakdown,
	})
}
