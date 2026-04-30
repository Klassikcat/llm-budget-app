import { describe, expect, it } from 'vitest';

import { setBindingClient, type BindingName, type WailsBindingClient } from './index';
import { loadAlerts, type AlertListResponse } from './alerts';

function clientReturning(value: AlertListResponse, calls: { binding: BindingName; method: string; args: readonly unknown[] }[]): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      calls.push({ binding, method, args });
      return value as T;
    }
  };
}

describe('alert bindings', () => {
  const response: AlertListResponse = {
    items: [
      {
        alertId: 'alert-1',
        kind: 'budget_threshold',
        severity: 'warning',
        triggeredAt: '2026-04-15T12:00:00Z',
        period: {
          month: '2026-04',
          startAt: '2026-04-01T00:00:00Z',
          endExclusive: '2026-05-01T00:00:00Z',
          currency: 'USD'
        },
        budgetId: 'budget-1',
        forecastId: '',
        insightId: '',
        detectorCategory: '',
        currentSpendUsd: 80,
        limitUsd: 100,
        thresholdPercent: 0.8
      }
    ],
    empty: false
  };

  it('loads alerts through AlertsBinding.LoadAlerts', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient(clientReturning(response, calls));

    await expect(loadAlerts('2026-04')).resolves.toEqual(response);
    expect(calls).toEqual([{ binding: 'AlertsBinding', method: 'LoadAlerts', args: ['2026-04'] }]);
  });

  it('rejects when the injected client rejects', async () => {
    setBindingClient({
      async invoke() {
        throw new Error('alert query failed');
      }
    });

    await expect(loadAlerts()).rejects.toThrow('alert query failed');
  });
});
