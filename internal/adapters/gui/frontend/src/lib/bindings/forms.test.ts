import { describe, expect, it } from 'vitest';

import type {
  BudgetInput,
  ManualEntryInput,
  MutationResponse,
  SubscriptionInput
} from '$lib/types/forms';
import type { ThresholdAlert } from '$lib/types/notifications';

import { setBindingClient, type BindingName, type WailsBindingClient } from './index';
import {
  BindingMutationError,
  deleteProviderSecret,
  dispatchAlertNotification,
  listSubscriptionPresets,
  loadSettings,
  saveBudget,
  saveManualEntry,
  saveProviderSecret,
  saveSettings,
  saveSubscription,
  type ProviderSecretDeleteInput,
  type ProviderSecretInput,
  type SettingsFormInput,
  type SettingsFormResponse
} from './forms';

type BindingCall = { binding: BindingName; method: string; args: readonly unknown[] };

type WrapperCase = {
  name: string;
  method: string;
  args: readonly unknown[];
  call: () => Promise<unknown>;
  expected: unknown;
};

function clientFromResponses(responses: Record<string, unknown>, calls: BindingCall[]): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      calls.push({ binding, method, args });
      if (!(method in responses)) {
        throw new Error(`missing test response for ${method}`);
      }
      return responses[method] as T;
    }
  };
}

const success: MutationResponse = { success: true };
const failed: MutationResponse = {
  success: false,
  error: {
    code: 'invalid_input',
    field: 'provider',
    message: 'provider is invalid'
  }
};

const settings: SettingsFormInput = {
  providers: {
    anthropicEnabled: true,
    openaiEnabled: true,
    geminiEnabled: false,
    openRouterEnabled: true
  },
  cliBillingDefaults: {
    claudeCode: 'subscription',
    codex: 'direct_api',
    geminiCli: 'subscription',
    openCode: 'openrouter'
  },
  subscriptionDefaults: {
    openai: {
      enabled: true,
      planCode: 'chatgpt-plus',
      planName: 'ChatGPT Plus',
      feeUsd: 20,
      renewalDay: 1,
      sourceUrl: 'https://openai.com'
    },
    claude: {
      enabled: true,
      planCode: 'claude-pro',
      planName: 'Claude Pro',
      feeUsd: 20,
      renewalDay: 2,
      sourceUrl: 'https://anthropic.com'
    },
    gemini: {
      enabled: false,
      planCode: 'gemini-pro',
      planName: 'Gemini Pro',
      feeUsd: 20,
      renewalDay: 3,
      sourceUrl: 'https://gemini.google.com'
    }
  },
  budgets: {
    monthlyBudgetUsd: 100,
    monthlySubscriptionBudgetUsd: 60,
    monthlyUsageBudgetUsd: 40,
    warningThresholdPercent: 80,
    criticalThresholdPercent: 95
  },
  notifications: {
    desktopEnabled: true,
    tuiEnabled: true,
    budgetWarnings: true,
    forecastWarnings: true,
    providerSyncFailure: true
  },
  databasePath: '/tmp/llmbudget.sqlite3'
};

const settingsResponse: SettingsFormResponse = {
  result: success,
  settings
};

const secretInput: ProviderSecretInput = {
  provider: 'openai',
  secretType: 'api_key',
  value: 'secret-value'
};

const secretDeleteInput: ProviderSecretDeleteInput = {
  provider: 'openai',
  secretType: 'api_key'
};

const subscriptionInput: SubscriptionInput = {
  presetKey: 'claude-pro',
  provider: '',
  planName: '',
  renewalDay: 0,
  startsAt: '2026-04-01',
  endsAt: '',
  feeUsd: 0,
  isActive: true
};

const manualEntryInput: ManualEntryInput = {
  provider: 'openai',
  modelId: 'gpt-4.1',
  occurredAt: '2026-04-15T12:00:00Z',
  inputTokens: 100,
  outputTokens: 50,
  cachedTokens: 10,
  cacheWriteTokens: 0,
  projectName: 'tracker',
  metadata: { source: 'test' }
};

const budgetInput: BudgetInput = {
  budgetId: 'budget-1',
  name: 'Monthly',
  provider: 'openai',
  projectHash: '',
  periodMonth: '2026-04',
  limitUsd: 100,
  warningThresholdPercent: 80,
  criticalThresholdPercent: 95,
  currency: 'USD'
};

const alertInput: ThresholdAlert = {
  alertId: 'alert-1',
  kind: 'budget_threshold',
  severity: 'warning',
  triggeredAt: '2026-04-15T12:00:00Z',
  periodMonth: '2026-04',
  budgetId: 'budget-1',
  forecastId: '',
  insightId: '',
  detectorCategory: '',
  currentSpendUsd: 85,
  limitUsd: 100,
  thresholdPercent: 0.8
};

const responses: Record<string, unknown> = {
  ListSubscriptionPresets: {
    items: [
      {
        key: 'claude-pro',
        provider: 'anthropic',
        planName: 'Claude Pro',
        renewalDay: 1,
        feeUsd: 20
      }
    ]
  },
  LoadSettings: settingsResponse,
  SaveSettings: settingsResponse,
  SaveProviderSecret: success,
  DeleteProviderSecret: success,
  SaveSubscription: {
    result: success,
    subscription: {
      subscriptionId: 'anthropic-claude-pro-2026-04-01',
      provider: 'anthropic',
      planName: 'Claude Pro',
      renewalDay: 1,
      startsAt: '2026-04-01',
      feeUsd: 20,
      isActive: true
    }
  },
  SaveManualEntry: {
    result: success,
    entry: {
      ...manualEntryInput,
      entryId: 'entry-1',
      totalCostUsd: 0.42
    }
  },
  SaveBudget: {
    result: success,
    budget: budgetInput
  },
  DispatchAlertNotification: {
    result: success,
    dispatched: true
  }
};

describe('form bindings', () => {
  const successCases: WrapperCase[] = [
    {
      name: 'lists subscription presets',
      method: 'ListSubscriptionPresets',
      args: [],
      call: listSubscriptionPresets,
      expected: responses.ListSubscriptionPresets
    },
    {
      name: 'loads settings',
      method: 'LoadSettings',
      args: [],
      call: loadSettings,
      expected: settingsResponse
    },
    {
      name: 'saves settings',
      method: 'SaveSettings',
      args: [settings],
      call: () => saveSettings(settings),
      expected: settingsResponse
    },
    {
      name: 'saves provider secrets',
      method: 'SaveProviderSecret',
      args: [secretInput],
      call: () => saveProviderSecret(secretInput),
      expected: success
    },
    {
      name: 'deletes provider secrets',
      method: 'DeleteProviderSecret',
      args: [secretDeleteInput],
      call: () => deleteProviderSecret(secretDeleteInput),
      expected: success
    },
    {
      name: 'saves subscriptions',
      method: 'SaveSubscription',
      args: [subscriptionInput],
      call: () => saveSubscription(subscriptionInput),
      expected: responses.SaveSubscription
    },
    {
      name: 'saves manual API entries',
      method: 'SaveManualEntry',
      args: [manualEntryInput],
      call: () => saveManualEntry(manualEntryInput),
      expected: responses.SaveManualEntry
    },
    {
      name: 'saves budgets',
      method: 'SaveBudget',
      args: [budgetInput],
      call: () => saveBudget(budgetInput),
      expected: responses.SaveBudget
    },
    {
      name: 'dispatches alert notifications',
      method: 'DispatchAlertNotification',
      args: [alertInput],
      call: () => dispatchAlertNotification(alertInput),
      expected: responses.DispatchAlertNotification
    }
  ];

  for (const testCase of successCases) {
    it(`${testCase.name} through FormsBinding.${testCase.method}`, async () => {
      const calls: BindingCall[] = [];
      setBindingClient(clientFromResponses(responses, calls));

      await expect(testCase.call()).resolves.toEqual(testCase.expected);
      expect(calls).toEqual([
        { binding: 'FormsBinding', method: testCase.method, args: testCase.args }
      ]);
    });
  }

  it('rejects raw client failures for read-only form calls', async () => {
    setBindingClient({
      async invoke() {
        throw new Error('Wails runtime unavailable');
      }
    });

    await expect(listSubscriptionPresets()).rejects.toThrow('Wails runtime unavailable');
  });

  const failureCases = [
    { name: 'loadSettings', call: loadSettings, method: 'LoadSettings', response: { result: failed, settings } },
    { name: 'saveSettings', call: () => saveSettings(settings), method: 'SaveSettings', response: { result: failed, settings } },
    { name: 'saveProviderSecret', call: () => saveProviderSecret(secretInput), method: 'SaveProviderSecret', response: failed },
    { name: 'deleteProviderSecret', call: () => deleteProviderSecret(secretDeleteInput), method: 'DeleteProviderSecret', response: failed },
    { name: 'saveSubscription', call: () => saveSubscription(subscriptionInput), method: 'SaveSubscription', response: { result: failed, subscription: responses.SaveSubscription } },
    { name: 'saveManualEntry', call: () => saveManualEntry(manualEntryInput), method: 'SaveManualEntry', response: { result: failed, entry: responses.SaveManualEntry } },
    { name: 'saveBudget', call: () => saveBudget(budgetInput), method: 'SaveBudget', response: { result: failed, budget: budgetInput } },
    { name: 'dispatchAlertNotification', call: () => dispatchAlertNotification(alertInput), method: 'DispatchAlertNotification', response: { result: failed, dispatched: false } }
  ];

  for (const testCase of failureCases) {
    it(`throws BindingMutationError when ${testCase.name} returns a failed mutation`, async () => {
      const calls: BindingCall[] = [];
      setBindingClient(clientFromResponses({ [testCase.method]: testCase.response }, calls));

      await expect(testCase.call()).rejects.toBeInstanceOf(BindingMutationError);
      await expect(testCase.call()).rejects.toThrow('provider is invalid');
    });
  }
});
