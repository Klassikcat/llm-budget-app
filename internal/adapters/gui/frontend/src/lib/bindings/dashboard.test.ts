import { beforeEach, describe, expect, it } from 'vitest';

import { setBindingClient, type BindingName, type WailsBindingClient } from './index';
import { loadDashboard, type DashboardResponse } from './dashboard';

function clientReturning(value: DashboardResponse, calls: { binding: BindingName; method: string; args: readonly unknown[] }[]): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      calls.push({ binding, method, args });
      return value as T;
    }
  };
}

describe('dashboard bindings', () => {
  const dashboard: DashboardResponse = {
    period: {
      month: '2026-04',
      startAt: '2026-04-01T00:00:00Z',
      endExclusive: '2026-05-01T00:00:00Z',
      currency: 'USD'
    },
    totals: {
      variableSpendUsd: 12,
      subscriptionSpendUsd: 20,
      totalSpendUsd: 32,
      currency: 'USD'
    },
    providerSummaries: [
      {
        provider: 'openai',
        variableSpendUsd: 12,
        subscriptionSpendUsd: 20,
        totalSpendUsd: 32,
        usageEntryCount: 2,
        sessionCount: 1,
        currency: 'USD'
      }
    ],
    budgets: [
      {
        budgetId: 'budget-1',
        name: 'Monthly',
        provider: 'openai',
        projectHash: '',
        limitUsd: 100,
        currentSpendUsd: 32,
        remainingUsd: 68,
        triggeredThresholdPercents: [],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ],
    recentSessions: [
      {
        sessionId: 'session-1',
        provider: 'openai',
        billingMode: 'direct_api',
        projectName: 'tracker',
        agentName: 'codex',
        modelId: 'gpt-4.1',
        startedAt: '2026-04-15T12:00:00Z',
        endedAt: '2026-04-15T12:10:00Z',
        durationSeconds: 600,
        totalCostUsd: 12,
        totalTokens: 1200,
        currency: 'USD'
      }
    ],
    empty: false
  };

  beforeEach(() => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient(clientReturning(dashboard, calls));
  });

  it('loads dashboard data through DashboardBinding.LoadDashboard', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient(clientReturning(dashboard, calls));

    await expect(loadDashboard('2026-04')).resolves.toEqual(dashboard);
    expect(calls).toEqual([
      { binding: 'DashboardBinding', method: 'LoadDashboard', args: ['2026-04'] }
    ]);
  });

  it('defaults to the backend-selected month when no month is supplied', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient(clientReturning(dashboard, calls));

    await loadDashboard();
    expect(calls[0]).toEqual({ binding: 'DashboardBinding', method: 'LoadDashboard', args: [''] });
  });

  it('rejects when the injected client rejects', async () => {
    setBindingClient({
      async invoke() {
        throw new Error('database unavailable');
      }
    });

    await expect(loadDashboard('2026-04')).rejects.toThrow('database unavailable');
  });
});
