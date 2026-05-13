import { describe, expect, it } from 'vitest';

import { setBindingClient, type BindingName, type WailsBindingClient } from './index';
import { loadGraphs, type GraphResponse } from './graphs';

function clientReturning(value: GraphResponse, calls: { binding: BindingName; method: string; args: readonly unknown[] }[]): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      calls.push({ binding, method, args });
      return value as T;
    }
  };
}

describe('graph bindings', () => {
  const response: GraphResponse = {
    modelTokenUsages: [
      {
        modelName: 'gpt-4.1',
        totalTokens: 150,
        inputTokens: 100,
        outputTokens: 40,
        cacheReadTokens: 10,
        cacheWriteTokens: 0
      }
    ],
    modelCosts: [{ modelName: 'gpt-4.1', totalCostUsd: 1.25 }],
    dailyTokenTrends: [
      {
        date: '2026-04-01T00:00:00Z',
        modelBreakdown: [{ modelName: 'gpt-4.1', totalTokens: 150 }]
      }
    ],
    modelTokenBreakdowns: [
      {
        modelName: 'gpt-4.1',
        inputTokens: 100,
        outputTokens: 40,
        cacheReadTokens: 10,
        cacheWriteTokens: 0,
        totalTokens: 150
      }
    ]
  };

  it('loads graph data through GraphsBinding.LoadGraphs', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient(clientReturning(response, calls));

    await expect(loadGraphs('2026-04', '7 days')).resolves.toEqual(response);
    expect(calls).toEqual([{ binding: 'GraphsBinding', method: 'LoadGraphs', args: ['2026-04', '7 days'] }]);
  });

  it('rejects when the injected client rejects', async () => {
    setBindingClient({
      async invoke() {
        throw new Error('graph load failed');
      }
    });

    await expect(loadGraphs()).rejects.toThrow('graph load failed');
  });
});
