export interface FormError {
  code: string;
  field?: string;
  message: string;
}

export interface MutationResponse {
  success: boolean;
  error?: FormError | null;
}

export interface ManualEntryInput {
  provider: string;
  modelId: string;
  occurredAt: string;
  inputTokens: number;
  outputTokens: number;
  cachedTokens: number;
  cacheWriteTokens: number;
  projectName: string;
  metadata: Record<string, string>;
}

export interface ManualEntryState extends ManualEntryInput {
  entryId: string;
  totalCostUsd: number;
}

export interface ManualEntryMutationResponse {
  result: MutationResponse;
  entry: ManualEntryState;
}

export interface SubscriptionInput {
  presetKey: string;
  provider: string;
  planName: string;
  renewalDay: number;
  startsAt: string;
  endsAt: string;
  feeUsd: number;
  isActive: boolean;
}

export interface SubscriptionState {
  subscriptionId: string;
  presetKey?: string;
  provider: string;
  planName: string;
  renewalDay: number;
  startsAt: string;
  endsAt?: string;
  feeUsd: number;
  isActive: boolean;
}

export interface SubscriptionMutationResponse {
  result: MutationResponse;
  subscription: SubscriptionState;
}

export interface SubscriptionPresetState {
  key: string;
  provider: string;
  planName: string;
  renewalDay: number;
  feeUsd: number;
}

export interface SubscriptionPresetsResponse {
  items: SubscriptionPresetState[];
}

export interface SubscriptionListResponse {
  items: SubscriptionState[];
  empty: boolean;
}

export interface BudgetInput {
  budgetId: string;
  name: string;
  provider: string;
  projectHash: string;
  periodMonth: string;
  limitUsd: number;
  warningThresholdPercent: number;
  criticalThresholdPercent: number;
  currency: string;
}

export interface BudgetState extends BudgetInput {
  provider: string;
  projectHash: string;
  criticalThresholdPercent: number;
}

export interface BudgetMutationResponse {
  result: MutationResponse;
  budget: BudgetState;
}

export const manualEntryInputFields = [
  'provider',
  'modelId',
  'occurredAt',
  'inputTokens',
  'outputTokens',
  'cachedTokens',
  'cacheWriteTokens',
  'projectName',
  'metadata'
] as const satisfies readonly (keyof ManualEntryInput)[];

export const subscriptionInputFields = [
  'presetKey',
  'provider',
  'planName',
  'renewalDay',
  'startsAt',
  'endsAt',
  'feeUsd',
  'isActive'
] as const satisfies readonly (keyof SubscriptionInput)[];

export const budgetInputFields = [
  'budgetId',
  'name',
  'provider',
  'projectHash',
  'periodMonth',
  'limitUsd',
  'warningThresholdPercent',
  'criticalThresholdPercent',
  'currency'
] as const satisfies readonly (keyof BudgetInput)[];
