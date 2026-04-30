import { describe, expect, it } from 'vitest';

import {
  budgetInputFields,
  manualEntryInputFields,
  subscriptionInputFields
} from './forms';
import {
  budgetStateFields,
  isBudgetState,
  isUsageEntry,
  isWasteSummary,
  monthlyBudgetFields,
  subscriptionFields,
  usageEntryFields,
  wasteSummaryFields
} from './domain';
import { isThresholdAlert, thresholdAlertFields } from './notifications';

const pascalCaseFields = [
  'BudgetID',
  'LimitUSD',
  'OccurredAt',
  'SubscriptionID',
  'WastePercent',
  'GeneratedAt'
];

function expectNoPascalCase(fields: readonly string[]) {
  for (const field of pascalCaseFields) {
    expect(fields).not.toContain(field);
  }
}

describe('domain type field names', () => {
  it('uses domain JSON-style snake_case names instead of Go PascalCase names', () => {
    expect(monthlyBudgetFields).toContain('budget_id');
    expect(monthlyBudgetFields).toContain('limit_usd');
    expect(budgetStateFields).toContain('triggered_threshold_percents');
    expect(usageEntryFields).toContain('occurred_at');
    expect(usageEntryFields).toContain('cost_breakdown');
    expect(subscriptionFields).toContain('subscription_id');
    expect(subscriptionFields).toContain('fee_usd');
    expect(wasteSummaryFields).toContain('waste_percent');
    expect(wasteSummaryFields).toContain('projected_month_end_waste_usd');

    expectNoPascalCase(monthlyBudgetFields);
    expectNoPascalCase(budgetStateFields);
    expectNoPascalCase(usageEntryFields);
    expectNoPascalCase(subscriptionFields);
    expectNoPascalCase(wasteSummaryFields);
  });

  it('keeps GUI form inputs aligned to Wails binding camelCase fields', () => {
    expect(manualEntryInputFields).toEqual([
      'provider',
      'modelId',
      'occurredAt',
      'inputTokens',
      'outputTokens',
      'cachedTokens',
      'cacheWriteTokens',
      'projectName',
      'metadata'
    ]);
    expect(subscriptionInputFields).toContain('presetKey');
    expect(subscriptionInputFields).toContain('feeUsd');
    expect(budgetInputFields).toContain('budgetId');
    expect(budgetInputFields).toContain('limitUsd');
  });
});

describe('domain type guards', () => {
  const period = {
    start_at: '2026-04-01T00:00:00Z',
    end_exclusive: '2026-05-01T00:00:00Z'
  };

  it('accepts a representative BudgetState payload and rejects PascalCase payloads', () => {
    expect(
      isBudgetState({
        budget_id: 'budget-1',
        period,
        current_spend_usd: 12.5,
        forecast_spend_usd: 25,
        triggered_threshold_percents: [0.8],
        budget_overrun_active: false,
        forecast_overrun_active: false,
        updated_at: '2026-04-15T12:00:00Z'
      })
    ).toBe(true);

    expect(
      isBudgetState({
        BudgetID: 'budget-1',
        Period: period,
        CurrentSpendUSD: 12.5,
        ForecastSpendUSD: 25,
        TriggeredThresholdPercents: [0.8],
        BudgetOverrunActive: false,
        ForecastOverrunActive: false,
        UpdatedAt: '2026-04-15T12:00:00Z'
      })
    ).toBe(false);
  });

  it('accepts representative UsageEntry and WasteSummary payloads', () => {
    expect(
      isUsageEntry({
        entry_id: 'entry-1',
        source: 'manual_api',
        provider: 'openai',
        billing_mode: 'direct_api',
        occurred_at: '2026-04-15T12:00:00Z',
        session_id: '',
        external_id: '',
        project_name: 'tracker',
        agent_name: '',
        metadata: { source: 'test' },
        pricing_ref: {
          provider: 'openai',
          model_id: 'gpt-4.1',
          pricing_lookup_key: 'gpt-4.1'
        },
        tokens: {
          input_tokens: 10,
          output_tokens: 20,
          cache_read_tokens: 0,
          cache_write_tokens: 0,
          total_tokens: 30
        },
        cost_breakdown: {
          input_usd: 0.01,
          output_usd: 0.02,
          cache_read_usd: 0,
          cache_write_usd: 0,
          tool_usd: 0,
          flat_usd: 0,
          total_usd: 0.03
        }
      })
    ).toBe(true);

    expect(
      isWasteSummary({
        period,
        total_waste_cost_usd: 2,
        total_spend_cost_usd: 10,
        waste_percent: 20,
        weekly_waste_cost_usd: 1,
        monthly_waste_cost_usd: 2,
        projected_month_end_waste_usd: 4,
        by_detector: [],
        top_causes: [],
        daily_trend: [],
        generated_at: '2026-04-15T12:00:00Z'
      })
    ).toBe(true);
  });

  it('validates threshold notification shape without applying business rules', () => {
    expect(thresholdAlertFields).toContain('thresholdPercent');
    expect(
      isThresholdAlert({
        alertId: 'alert-1',
        kind: 'budget_threshold',
        severity: 'warning',
        triggeredAt: '2026-04-15T12:00:00Z',
        periodMonth: '2026-04',
        budgetId: 'budget-1',
        forecastId: '',
        insightId: '',
        detectorCategory: '',
        currentSpendUsd: 80,
        limitUsd: 100,
        thresholdPercent: 0.8
      })
    ).toBe(true);
  });
});
