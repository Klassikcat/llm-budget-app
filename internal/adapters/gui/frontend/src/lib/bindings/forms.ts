import type {
  BudgetInput,
  BudgetMutationResponse,
  ManualEntryInput,
  ManualEntryMutationResponse,
  MutationResponse,
  SubscriptionInput,
  SubscriptionMutationResponse,
  SubscriptionPresetsResponse
} from '$lib/types/forms';
import type { NotificationDispatchResponse, ThresholdAlert } from '$lib/types/notifications';

import { getBindingClient } from './client';

export interface ProviderSettingsState {
  anthropicEnabled: boolean;
  openaiEnabled: boolean;
  geminiEnabled: boolean;
  openRouterEnabled: boolean;
}

export interface CLIBillingDefaultsState {
  claudeCode: string;
  codex: string;
  geminiCli: string;
  openCode: string;
}

export interface SubscriptionPlanState {
  enabled: boolean;
  planCode: string;
  planName: string;
  feeUsd: number;
  renewalDay: number;
  sourceUrl: string;
}

export interface SubscriptionDefaultsState {
  openai: SubscriptionPlanState;
  claude: SubscriptionPlanState;
  gemini: SubscriptionPlanState;
}

export interface BudgetSettingsState {
  monthlyBudgetUsd: number;
  monthlySubscriptionBudgetUsd: number;
  monthlyUsageBudgetUsd: number;
  warningThresholdPercent: number;
  criticalThresholdPercent: number;
}

export interface NotificationSettingsState {
  desktopEnabled: boolean;
  tuiEnabled: boolean;
  budgetWarnings: boolean;
  forecastWarnings: boolean;
  providerSyncFailure: boolean;
}

export interface SettingsFormState {
  providers: ProviderSettingsState;
  cliBillingDefaults: CLIBillingDefaultsState;
  subscriptionDefaults: SubscriptionDefaultsState;
  budgets: BudgetSettingsState;
  notifications: NotificationSettingsState;
  databasePath: string;
}

export interface SettingsFormInput extends SettingsFormState {}

export interface SettingsFormResponse {
  result: MutationResponse;
  settings: SettingsFormState;
}

export interface ProviderSecretInput {
  provider: string;
  secretType: string;
  value: string;
}

export interface ProviderSecretDeleteInput {
  provider: string;
  secretType: string;
}

type ResultEnvelope = {
  result: MutationResponse;
};

export class BindingMutationError extends Error {
  readonly result: MutationResponse;

  constructor(result: MutationResponse) {
    super(result.error?.message ?? 'binding mutation failed');
    this.name = 'BindingMutationError';
    this.result = result;
  }
}

function ensureMutationResult<T extends MutationResponse>(response: T): T {
  if (!response.success) {
    throw new BindingMutationError(response);
  }
  return response;
}

function ensureEnvelope<T extends ResultEnvelope>(response: T): T {
  if (!response.result.success) {
    throw new BindingMutationError(response.result);
  }
  return response;
}

export async function listSubscriptionPresets(): Promise<SubscriptionPresetsResponse> {
  return getBindingClient().invoke<SubscriptionPresetsResponse>('FormsBinding', 'ListSubscriptionPresets');
}

export async function loadSettings(): Promise<SettingsFormResponse> {
  const response = await getBindingClient().invoke<SettingsFormResponse>('FormsBinding', 'LoadSettings');
  return ensureEnvelope(response);
}

export async function saveSettings(input: SettingsFormInput): Promise<SettingsFormResponse> {
  const response = await getBindingClient().invoke<SettingsFormResponse>('FormsBinding', 'SaveSettings', input);
  return ensureEnvelope(response);
}

export async function saveProviderSecret(input: ProviderSecretInput): Promise<MutationResponse> {
  const response = await getBindingClient().invoke<MutationResponse>('FormsBinding', 'SaveProviderSecret', input);
  return ensureMutationResult(response);
}

export async function deleteProviderSecret(input: ProviderSecretDeleteInput): Promise<MutationResponse> {
  const response = await getBindingClient().invoke<MutationResponse>('FormsBinding', 'DeleteProviderSecret', input);
  return ensureMutationResult(response);
}

export async function saveSubscription(input: SubscriptionInput): Promise<SubscriptionMutationResponse> {
  const response = await getBindingClient().invoke<SubscriptionMutationResponse>('FormsBinding', 'SaveSubscription', input);
  return ensureEnvelope(response);
}

export async function saveManualEntry(input: ManualEntryInput): Promise<ManualEntryMutationResponse> {
  const response = await getBindingClient().invoke<ManualEntryMutationResponse>('FormsBinding', 'SaveManualEntry', input);
  return ensureEnvelope(response);
}

export async function saveBudget(input: BudgetInput): Promise<BudgetMutationResponse> {
  const response = await getBindingClient().invoke<BudgetMutationResponse>('FormsBinding', 'SaveBudget', input);
  return ensureEnvelope(response);
}

export async function dispatchAlertNotification(input: ThresholdAlert): Promise<NotificationDispatchResponse> {
  const response = await getBindingClient().invoke<NotificationDispatchResponse>('FormsBinding', 'DispatchAlertNotification', input);
  return ensureEnvelope(response);
}
