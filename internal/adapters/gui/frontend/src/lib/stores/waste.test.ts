import { get } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type {
  AlertListResponse,
  BindingName,
  InsightListResponse,
  WailsBindingClient,
  WasteSummaryResponse
} from '$lib/bindings';

function createWasteSummary(totalWasteCostUsd: number): WasteSummaryResponse {
  return {
    period: {
      month: '2026-04',
      startAt: '2026-04-01T00:00:00Z',
      endExclusive: '2026-05-01T00:00:00Z',
      currency: 'USD'
    },
    totalWasteCostUsd,
    totalSpendCostUsd: 40,
    wastePercent: 25,
    weeklyWasteCostUsd: 3,
    monthlyWasteCostUsd: totalWasteCostUsd,
    projectedMonthEndWasteUsd: 12,
    byDetector: [{ category: 'planning_tax', attributedCostUsd: totalWasteCostUsd, insightCount: 1 }],
    topCauses: [{ category: 'planning_tax', attributedCostUsd: totalWasteCostUsd, insightCount: 1 }],
    dailyTrend: [{ day: '2026-04-10T00:00:00Z', wasteCostUsd: totalWasteCostUsd }],
    generatedAt: '2026-04-10T00:00:00Z'
  };
}

function createInsights(): InsightListResponse {
  return {
    items: [
      {
        insightId: 'insight-1',
        category: 'planning_tax',
        severity: 'medium',
        detectedAt: '2026-04-10T00:00:00Z',
        period: {
          month: '2026-04',
          startAt: '2026-04-01T00:00:00Z',
          endExclusive: '2026-05-01T00:00:00Z',
          currency: 'USD'
        },
        payload: {
          sessionIds: ['session-1'],
          usageEntryIds: ['entry-1'],
          hashes: [],
          counts: [{ key: 'steps', value: 3 }],
          metrics: [{ key: 'cost', unit: 'usd', value: 10 }]
        }
      }
    ],
    empty: false
  };
}

function createAlerts(): AlertListResponse {
  return {
    items: [
      {
        alertId: 'alert-1',
        kind: 'waste',
        severity: 'warning',
        triggeredAt: '2026-04-10T00:00:00Z',
        period: {
          month: '2026-04',
          startAt: '2026-04-01T00:00:00Z',
          endExclusive: '2026-05-01T00:00:00Z',
          currency: 'USD'
        },
        budgetId: '',
        forecastId: '',
        insightId: 'insight-1',
        detectorCategory: 'planning_tax',
        currentSpendUsd: 10,
        limitUsd: 0,
        thresholdPercent: 0
      }
    ],
    empty: false
  };
}

async function loadWasteStore(client: WailsBindingClient) {
  vi.resetModules();
  const bindings = await import('$lib/bindings');
  bindings.setBindingClient(client);
  return import('./waste');
}

describe('waste store', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads waste summary, insights, and alerts into readable state', async () => {
    const summary = createWasteSummary(10);
    const insights = createInsights();
    const alerts = createAlerts();
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const { waste } = await loadWasteStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        if (method === 'LoadWasteSummary') {
          return summary as T;
        }
        if (method === 'LoadInsights') {
          return insights as T;
        }
        return alerts as T;
      }
    });

    await expect(waste.load('2026-04')).resolves.toEqual({ summary, insights, alerts });

    expect(get(waste)).toEqual({ data: { summary, insights, alerts }, loading: false, error: null });
    expect(calls.map((call) => call.method).sort()).toEqual(['LoadAlerts', 'LoadInsights', 'LoadWasteSummary']);
  });

  it('sets loading while waste requests are pending', async () => {
    let resolveSummary: (value: WasteSummaryResponse) => void = () => undefined;
    const pendingSummary = new Promise<WasteSummaryResponse>((resolve) => {
      resolveSummary = resolve;
    });
    const { waste } = await loadWasteStore({
      async invoke<T>(binding: BindingName, method: string) {
        if (method === 'LoadWasteSummary') {
          return pendingSummary as Promise<T>;
        }
        return (method === 'LoadInsights' ? createInsights() : createAlerts()) as T;
      }
    });

    const loadPromise = waste.load('2026-04');

    expect(get(waste).loading).toBe(true);
    resolveSummary(createWasteSummary(10));
    await loadPromise;
    expect(get(waste).loading).toBe(false);
  });

  it('refreshes by reusing the active month', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const summaries = [createWasteSummary(10), createWasteSummary(12)];
    const { waste } = await loadWasteStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        if (method === 'LoadWasteSummary') {
          return summaries.shift() as T;
        }
        return (method === 'LoadInsights' ? createInsights() : createAlerts()) as T;
      }
    });

    await waste.load('2026-04');
    await waste.refresh();

    expect(get(waste).data.summary?.totalWasteCostUsd).toBe(12);
    expect(calls.every((call) => call.args[0] === '2026-04')).toBe(true);
  });

  it('tracks errors without clearing existing waste data', async () => {
    const summary = createWasteSummary(10);
    const insights = createInsights();
    const alerts = createAlerts();
    let shouldReject = false;
    const { waste } = await loadWasteStore({
      async invoke<T>(binding: BindingName, method: string) {
        if (shouldReject && method === 'LoadInsights') {
          throw new Error('insights unavailable');
        }
        if (method === 'LoadWasteSummary') {
          return summary as T;
        }
        return (method === 'LoadInsights' ? insights : alerts) as T;
      }
    });
    await waste.load('2026-04');

    shouldReject = true;
    await expect(waste.refresh()).rejects.toThrow('insights unavailable');

    expect(get(waste).data.summary).toBe(summary);
    expect(get(waste).error).toBe('insights unavailable');
  });
});
