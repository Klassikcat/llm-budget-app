import { get } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { BindingName, DashboardResponse, WailsBindingClient } from '$lib/bindings';
import type { BudgetInput, BudgetMutationResponse } from '$lib/types/forms';

function createDashboard(limitUsd: number): DashboardResponse {
  return {
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
    providerSummaries: [],
    budgets: [
      {
        budgetId: 'budget-1',
        name: 'Monthly',
        provider: 'openai',
        projectHash: '',
        limitUsd,
        currentSpendUsd: 32,
        remainingUsd: limitUsd - 32,
        triggeredThresholdPercents: [],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ],
    recentSessions: [],
    empty: false
  };
}

async function loadBudgetStore(client: WailsBindingClient) {
  vi.resetModules();
  const bindings = await import('$lib/bindings');
  bindings.setBindingClient(client);
  return import('./budget');
}

describe('budget store', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads budget dashboard data into typed readable state', async () => {
    const dashboard = createDashboard(100);
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const { budget } = await loadBudgetStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return dashboard as T;
      }
    });

    await expect(budget.load('2026-04')).resolves.toEqual({ dashboard, budgets: dashboard.budgets });

    expect(get(budget)).toEqual({
      data: { dashboard, budgets: dashboard.budgets },
      loading: false,
      error: null
    });
    expect(calls).toEqual([{ binding: 'DashboardBinding', method: 'LoadDashboard', args: ['2026-04'] }]);
  });

  it('sets loading while a load is pending', async () => {
    let resolveLoad: (value: DashboardResponse) => void = () => undefined;
    const pendingDashboard = new Promise<DashboardResponse>((resolve) => {
      resolveLoad = resolve;
    });
    const { budget } = await loadBudgetStore({
      async invoke<T>() {
        return pendingDashboard as Promise<T>;
      }
    });

    const loadPromise = budget.load('2026-04');

    expect(get(budget).loading).toBe(true);
    expect(get(budget).error).toBeNull();
    resolveLoad(createDashboard(100));
    await loadPromise;
    expect(get(budget).loading).toBe(false);
  });

  it('refreshes by reusing the active month', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const dashboards = [createDashboard(100), createDashboard(120)];
    const { budget } = await loadBudgetStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return dashboards.shift() as T;
      }
    });

    await budget.load('2026-04');
    await budget.refresh();

    expect(get(budget).data.budgets[0]?.limitUsd).toBe(120);
    expect(calls.map((call) => call.args)).toEqual([['2026-04'], ['2026-04']]);
  });

  it('tracks errors without clearing existing budget data', async () => {
    const dashboard = createDashboard(100);
    let shouldReject = false;
    const { budget } = await loadBudgetStore({
      async invoke<T>() {
        if (shouldReject) {
          throw new Error('budget unavailable');
        }
        return dashboard as T;
      }
    });
    await budget.load('2026-04');

    shouldReject = true;
    await expect(budget.refresh()).rejects.toThrow('budget unavailable');

    expect(get(budget)).toEqual({
      data: { dashboard, budgets: dashboard.budgets },
      loading: false,
      error: 'budget unavailable'
    });
  });

  it('refreshes after a successful budget mutation', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const input: BudgetInput = {
      budgetId: 'budget-1',
      name: 'Monthly',
      provider: 'openai',
      projectHash: '',
      periodMonth: '2026-04',
      limitUsd: 140,
      warningThresholdPercent: 75,
      criticalThresholdPercent: 90,
      currency: 'USD'
    };
    const mutation: BudgetMutationResponse = { result: { success: true }, budget: input };
    const { budget } = await loadBudgetStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        if (method === 'SaveBudget') {
          return mutation as T;
        }
        return createDashboard(140) as T;
      }
    });

    await expect(budget.save(input)).resolves.toBe(mutation);

    expect(calls.map((call) => call.method)).toEqual([
      'SaveBudget',
      'LoadDashboard',
      'LoadDashboard',
      'LoadGraphs',
      'LoadWasteSummary'
    ]);
    expect(get(budget).data.budgets[0]?.limitUsd).toBe(140);
  });
});
