import { get } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { BindingName, DashboardResponse, GraphResponse, WailsBindingClient } from '$lib/bindings';
import type { ManualEntryInput, ManualEntryMutationResponse } from '$lib/types/forms';

function createDashboard(totalSpendUsd: number): DashboardResponse {
  return {
    period: {
      month: '2026-04',
      startAt: '2026-04-01T00:00:00Z',
      endExclusive: '2026-05-01T00:00:00Z',
      currency: 'USD'
    },
    totals: {
      variableSpendUsd: totalSpendUsd,
      subscriptionSpendUsd: 0,
      totalSpendUsd,
      currency: 'USD'
    },
    providerSummaries: [
      {
        provider: 'openai',
        variableSpendUsd: totalSpendUsd,
        subscriptionSpendUsd: 0,
        totalSpendUsd,
        usageEntryCount: 1,
        sessionCount: 1,
        currency: 'USD'
      }
    ],
    budgets: [],
    recentSessions: [
      {
        sessionId: 'session-1',
        provider: 'openai',
        billingMode: 'direct_api',
        projectName: 'tracker',
        agentName: 'codex',
        modelId: 'gpt-4.1',
        startedAt: '2026-04-10T00:00:00Z',
        endedAt: '2026-04-10T00:05:00Z',
        durationSeconds: 300,
        totalCostUsd: totalSpendUsd,
        totalTokens: 1000,
        currency: 'USD'
      }
    ],
    empty: false
  };
}

function createGraphs(totalTokens: number): GraphResponse {
  return {
    modelTokenUsages: [
      {
        modelName: 'gpt-4.1',
        totalTokens,
        inputTokens: 600,
        outputTokens: 400,
        cacheReadTokens: 0,
        cacheWriteTokens: 0
      }
    ],
    modelCosts: [{ modelName: 'gpt-4.1', totalCostUsd: 3 }],
    dailyTokenTrends: [{ date: '2026-04-10', modelBreakdown: [{ modelName: 'gpt-4.1', totalTokens }] }],
    modelTokenBreakdowns: [
      {
        modelName: 'gpt-4.1',
        inputTokens: 600,
        outputTokens: 400,
        cacheReadTokens: 0,
        cacheWriteTokens: 0,
        totalTokens
      }
    ]
  };
}

async function loadUsageStore(client: WailsBindingClient) {
  vi.resetModules();
  const bindings = await import('$lib/bindings');
  bindings.setBindingClient(client);
  return import('./usage');
}

describe('usage store', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads dashboard usage and graph data into readable state', async () => {
    const dashboard = createDashboard(3);
    const graphs = createGraphs(1000);
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const { usage } = await loadUsageStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return (method === 'LoadGraphs' ? graphs : dashboard) as T;
      }
    });

    await expect(usage.load('2026-04')).resolves.toEqual({
      dashboard,
      graphs,
      providerSummaries: dashboard.providerSummaries,
      recentSessions: dashboard.recentSessions
    });

    expect(get(usage).data.recentSessions).toEqual(dashboard.recentSessions);
    expect(calls.map((call) => call.method).sort()).toEqual(['LoadDashboard', 'LoadGraphs']);
  });

  it('sets loading while usage requests are pending', async () => {
    let resolveDashboard: (value: DashboardResponse) => void = () => undefined;
    const pendingDashboard = new Promise<DashboardResponse>((resolve) => {
      resolveDashboard = resolve;
    });
    const { usage } = await loadUsageStore({
      async invoke<T>(binding: BindingName, method: string) {
        if (method === 'LoadDashboard') {
          return pendingDashboard as Promise<T>;
        }
        return createGraphs(1000) as T;
      }
    });

    const loadPromise = usage.load('2026-04');

    expect(get(usage).loading).toBe(true);
    resolveDashboard(createDashboard(3));
    await loadPromise;
    expect(get(usage).loading).toBe(false);
  });

  it('refreshes by reusing the active month', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const { usage } = await loadUsageStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return (method === 'LoadGraphs' ? createGraphs(2000) : createDashboard(4)) as T;
      }
    });

    await usage.load('2026-04');
    await usage.refresh();

    expect(calls.every((call) => call.args[0] === '2026-04')).toBe(true);
    expect(calls.filter((call) => call.method === 'LoadDashboard')).toHaveLength(2);
  });

  it('tracks errors without clearing existing usage data', async () => {
    const dashboard = createDashboard(3);
    const graphs = createGraphs(1000);
    let shouldReject = false;
    const { usage } = await loadUsageStore({
      async invoke<T>(binding: BindingName, method: string) {
        if (shouldReject && method === 'LoadGraphs') {
          throw new Error('graphs unavailable');
        }
        return (method === 'LoadGraphs' ? graphs : dashboard) as T;
      }
    });
    await usage.load('2026-04');

    shouldReject = true;
    await expect(usage.refresh()).rejects.toThrow('graphs unavailable');

    expect(get(usage).data.graphs).toBe(graphs);
    expect(get(usage).error).toBe('graphs unavailable');
  });

  it('refreshes after a successful manual usage mutation', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const input: ManualEntryInput = {
      provider: 'openai',
      modelId: 'gpt-4.1',
      occurredAt: '2026-04-10T00:00:00Z',
      inputTokens: 600,
      outputTokens: 400,
      cachedTokens: 0,
      cacheWriteTokens: 0,
      projectName: 'tracker',
      metadata: {}
    };
    const mutation: ManualEntryMutationResponse = {
      result: { success: true },
      entry: { ...input, entryId: 'entry-1', totalCostUsd: 3 }
    };
    const { usage } = await loadUsageStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        if (method === 'SaveManualEntry') {
          return mutation as T;
        }
        return (method === 'LoadGraphs' ? createGraphs(1000) : createDashboard(3)) as T;
      }
    });

    await expect(usage.save(input)).resolves.toBe(mutation);

    expect(calls.map((call) => call.method).sort()).toEqual([
      'LoadDashboard',
      'LoadDashboard',
      'LoadGraphs',
      'LoadGraphs',
      'LoadWasteSummary',
      'SaveManualEntry'
    ]);
  });
});
