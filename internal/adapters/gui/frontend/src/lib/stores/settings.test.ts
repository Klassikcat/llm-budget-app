import { get } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { BindingName, SettingsFormState, WailsBindingClient } from '$lib/bindings';

function createSettings(budgetWarnings: boolean): SettingsFormState {
  return {
    providers: {
      anthropicEnabled: true,
      openaiEnabled: true,
      geminiEnabled: false,
      openRouterEnabled: false
    },
    cliBillingDefaults: {
      claudeCode: 'direct_api',
      codex: 'direct_api',
      geminiCli: 'direct_api',
      openCode: 'direct_api'
    },
    subscriptionDefaults: {
      openai: {
        enabled: true,
        planCode: 'chatgpt-plus',
        planName: 'ChatGPT Plus',
        feeUsd: 20,
        renewalDay: 1,
        sourceUrl: ''
      },
      claude: {
        enabled: false,
        planCode: 'claude-pro',
        planName: 'Claude Pro',
        feeUsd: 20,
        renewalDay: 1,
        sourceUrl: ''
      },
      gemini: {
        enabled: false,
        planCode: 'gemini-pro',
        planName: 'Gemini Pro',
        feeUsd: 20,
        renewalDay: 1,
        sourceUrl: ''
      }
    },
    budgets: {
      monthlyBudgetUsd: 100,
      monthlySubscriptionBudgetUsd: 40,
      monthlyUsageBudgetUsd: 60,
      warningThresholdPercent: 80,
      criticalThresholdPercent: 95
    },
    notifications: {
      desktopEnabled: budgetWarnings,
      tuiEnabled: false,
      budgetWarnings,
      forecastWarnings: true,
      providerSyncFailure: true
    },
    databasePath: '/tmp/llmbudget.sqlite3'
  };
}

async function loadSettingsStore(client: WailsBindingClient) {
  vi.resetModules();
  const bindings = await import('$lib/bindings');
  bindings.setBindingClient(client);
  return import('./settings');
}

describe('settings store', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads settings through the binding wrapper', async () => {
    const settingsState = createSettings(true);
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const { settings } = await loadSettingsStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return { result: { success: true }, settings: settingsState } as T;
      }
    });

    await expect(settings.load()).resolves.toEqual(settingsState);

    expect(get(settings)).toEqual({
      data: settingsState,
      loading: false,
      saving: false,
      error: null
    });
    expect(calls).toEqual([{ binding: 'FormsBinding', method: 'LoadSettings', args: [] }]);
  });

  it('refreshes settings after saving', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const saved = createSettings(false);
    const refreshed = createSettings(false);
    const { settings } = await loadSettingsStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        if (method === 'SaveSettings') {
          return { result: { success: true }, settings: saved } as T;
        }
        return { result: { success: true }, settings: refreshed } as T;
      }
    });

    await expect(settings.save(saved)).resolves.toEqual(refreshed);

    expect(calls.map((call) => call.method)).toEqual(['SaveSettings', 'LoadSettings']);
    expect(get(settings).data?.notifications.budgetWarnings).toBe(false);
  });

  it('keeps existing data and exposes binding errors', async () => {
    const initial = createSettings(true);
    let shouldReject = false;
    const { settings } = await loadSettingsStore({
      async invoke<T>() {
        if (shouldReject) {
          throw new Error('settings unavailable');
        }
        return { result: { success: true }, settings: initial } as T;
      }
    });
    await settings.load();

    shouldReject = true;
    await expect(settings.refresh()).rejects.toThrow('settings unavailable');

    expect(get(settings)).toEqual({
      data: initial,
      loading: false,
      saving: false,
      error: 'settings unavailable'
    });
  });
});
