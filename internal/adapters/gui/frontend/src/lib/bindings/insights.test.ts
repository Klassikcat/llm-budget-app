import { describe, expect, it } from 'vitest';

import { setBindingClient, type BindingName, type WailsBindingClient } from './index';
import {
  loadInsights,
  loadWasteSummary,
  type InsightListResponse,
  type WasteSummaryResponse
} from './insights';

type BindingCall = { binding: BindingName; method: string; args: readonly unknown[] };

function clientFromResponses(responses: Record<string, unknown>, calls: BindingCall[]): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      calls.push({ binding, method, args });
      return responses[method] as T;
    }
  };
}

describe('insight bindings', () => {
  const wasteSummary: WasteSummaryResponse = {
    period: {
      month: '2026-04',
      startAt: '2026-04-01T00:00:00Z',
      endExclusive: '2026-05-01T00:00:00Z',
      currency: 'USD'
    },
    totalWasteCostUsd: 3,
    totalSpendCostUsd: 12,
    wastePercent: 25,
    weeklyWasteCostUsd: 1,
    monthlyWasteCostUsd: 3,
    projectedMonthEndWasteUsd: 6,
    byDetector: [{ category: 'retry_amplification', attributedCostUsd: 3, insightCount: 1 }],
    topCauses: [{ category: 'retry_amplification', attributedCostUsd: 3, insightCount: 1 }],
    dailyTrend: [{ day: '2026-04-01T00:00:00Z', wasteCostUsd: 3 }],
    generatedAt: '2026-04-15T00:00:00Z'
  };

  const insights: InsightListResponse = {
    items: [
      {
        insightId: 'insight-1',
        category: 'retry_amplification',
        severity: 'high',
        detectedAt: '2026-04-01T12:00:00Z',
        period: wasteSummary.period,
        payload: {
          sessionIds: [],
          usageEntryIds: ['entry-1'],
          hashes: [],
          counts: [{ key: 'retry_count', value: 2 }],
          metrics: [{ key: 'waste_cost', unit: 'usd', value: 3 }]
        }
      }
    ],
    empty: false
  };

  it('loads waste summary through InsightsBinding.LoadWasteSummary', async () => {
    const calls: BindingCall[] = [];
    setBindingClient(clientFromResponses({ LoadWasteSummary: wasteSummary }, calls));

    await expect(loadWasteSummary('2026-04')).resolves.toEqual(wasteSummary);
    expect(calls).toEqual([{ binding: 'InsightsBinding', method: 'LoadWasteSummary', args: ['2026-04'] }]);
  });

  it('loads insights through InsightsBinding.LoadInsights', async () => {
    const calls: BindingCall[] = [];
    setBindingClient(clientFromResponses({ LoadInsights: insights }, calls));

    await expect(loadInsights('2026-04')).resolves.toEqual(insights);
    expect(calls).toEqual([{ binding: 'InsightsBinding', method: 'LoadInsights', args: ['2026-04'] }]);
  });

  it('rejects when the injected client rejects', async () => {
    setBindingClient({
      async invoke() {
        throw new Error('insights unavailable');
      }
    });

    await expect(loadInsights()).rejects.toThrow('insights unavailable');
  });
});
