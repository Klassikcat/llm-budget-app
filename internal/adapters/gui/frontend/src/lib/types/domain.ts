export type ISODateTimeString = string;

export type ProviderName =
  | 'anthropic'
  | 'openai'
  | 'gemini'
  | 'openrouter'
  | 'claude'
  | 'codex'
  | 'opencode'
  | (string & {});

export type AlertSeverity = 'info' | 'warning' | 'critical';
export type BillingMode = 'unknown' | 'subscription' | 'byok' | 'direct_api' | 'openrouter';
export type UsageSourceKind = 'subscription' | 'manual_api' | 'openrouter' | 'cli_session';
export type DetectorCategory =
  | 'context_avalanche'
  | 'repeated_file_reads'
  | 'retry_amplification'
  | 'over_qualified_model_choice'
  | 'tool_schema_bloat'
  | 'planning_tax'
  | 'zombie_loops'
  | 'missed_prompt_caching';

export interface MonthlyPeriod {
  start_at: ISODateTimeString;
  end_exclusive: ISODateTimeString;
}

export interface BudgetThreshold {
  severity: AlertSeverity;
  percent: number;
}

export interface MonthlyBudget {
  budget_id: string;
  name: string;
  period: MonthlyPeriod;
  limit_usd: number;
  thresholds: BudgetThreshold[];
  currency: string;
  provider: ProviderName;
  project_hash: string;
}

export interface BudgetStatus {
  triggered_thresholds: BudgetThreshold[];
  remaining_usd: number;
  is_overrun: boolean;
}

export interface ForecastSnapshot {
  forecast_id: string;
  period: MonthlyPeriod;
  generated_at: ISODateTimeString;
  actual_spend_usd: number;
  forecast_spend_usd: number;
  budget_limit_usd: number;
  projected_overrun_usd: number;
  observed_day_count: number;
  remaining_day_count: number;
}

export interface BudgetState {
  budget_id: string;
  period: MonthlyPeriod;
  current_spend_usd: number;
  forecast_spend_usd: number;
  triggered_threshold_percents: number[];
  budget_overrun_active: boolean;
  forecast_overrun_active: boolean;
  updated_at: ISODateTimeString;
}

export interface ModelPricingRef {
  provider: ProviderName;
  model_id: string;
  pricing_lookup_key: string;
}

export interface TokenUsage {
  input_tokens: number;
  output_tokens: number;
  cache_read_tokens: number;
  cache_write_tokens: number;
  total_tokens: number;
}

export interface CostBreakdown {
  input_usd: number;
  output_usd: number;
  cache_read_usd: number;
  cache_write_usd: number;
  tool_usd: number;
  flat_usd: number;
  total_usd: number;
}

export interface UsageEntry {
  entry_id: string;
  source: UsageSourceKind;
  provider: ProviderName;
  billing_mode: BillingMode;
  occurred_at: ISODateTimeString;
  session_id: string;
  external_id: string;
  project_name: string;
  agent_name: string;
  metadata: Record<string, string> | null;
  pricing_ref: ModelPricingRef | null;
  tokens: TokenUsage;
  cost_breakdown: CostBreakdown;
}

export interface Subscription {
  subscription_id: string;
  provider: ProviderName;
  plan_code: string;
  plan_name: string;
  renewal_day: number;
  starts_at: ISODateTimeString;
  ends_at: ISODateTimeString | null;
  fee_usd: number;
  is_active: boolean;
  created_at: ISODateTimeString;
  updated_at: ISODateTimeString;
}

export interface SubscriptionFee {
  subscription_id: string;
  provider: ProviderName;
  plan_code: string;
  charged_at: ISODateTimeString;
  period: MonthlyPeriod;
  fee_usd: number;
}

export interface WasteByDetector {
  category: DetectorCategory;
  attributed_cost_usd: number;
  insight_count: number;
}

export interface WasteTrendPoint {
  day: ISODateTimeString;
  waste_cost_usd: number;
}

export interface WasteSummary {
  period: MonthlyPeriod;
  total_waste_cost_usd: number;
  total_spend_cost_usd: number;
  waste_percent: number;
  weekly_waste_cost_usd: number;
  monthly_waste_cost_usd: number;
  projected_month_end_waste_usd: number;
  by_detector: WasteByDetector[];
  top_causes: WasteByDetector[];
  daily_trend: WasteTrendPoint[];
  generated_at: ISODateTimeString;
}

export const monthlyBudgetFields = [
  'budget_id',
  'name',
  'period',
  'limit_usd',
  'thresholds',
  'currency',
  'provider',
  'project_hash'
] as const satisfies readonly (keyof MonthlyBudget)[];

export const budgetStateFields = [
  'budget_id',
  'period',
  'current_spend_usd',
  'forecast_spend_usd',
  'triggered_threshold_percents',
  'budget_overrun_active',
  'forecast_overrun_active',
  'updated_at'
] as const satisfies readonly (keyof BudgetState)[];

export const usageEntryFields = [
  'entry_id',
  'source',
  'provider',
  'billing_mode',
  'occurred_at',
  'session_id',
  'external_id',
  'project_name',
  'agent_name',
  'metadata',
  'pricing_ref',
  'tokens',
  'cost_breakdown'
] as const satisfies readonly (keyof UsageEntry)[];

export const subscriptionFields = [
  'subscription_id',
  'provider',
  'plan_code',
  'plan_name',
  'renewal_day',
  'starts_at',
  'ends_at',
  'fee_usd',
  'is_active',
  'created_at',
  'updated_at'
] as const satisfies readonly (keyof Subscription)[];

export const wasteSummaryFields = [
  'period',
  'total_waste_cost_usd',
  'total_spend_cost_usd',
  'waste_percent',
  'weekly_waste_cost_usd',
  'monthly_waste_cost_usd',
  'projected_month_end_waste_usd',
  'by_detector',
  'top_causes',
  'daily_trend',
  'generated_at'
] as const satisfies readonly (keyof WasteSummary)[];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function hasKeys(value: unknown, keys: readonly string[]): value is Record<string, unknown> {
  return isRecord(value) && keys.every((key) => key in value);
}

function isNumberArray(value: unknown): value is number[] {
  return Array.isArray(value) && value.every((item) => typeof item === 'number');
}

function isStringRecordOrNull(value: unknown): value is Record<string, string> | null {
  return (
    value === null ||
    (isRecord(value) && Object.values(value).every((item) => typeof item === 'string'))
  );
}

export function isMonthlyPeriod(value: unknown): value is MonthlyPeriod {
  return (
    hasKeys(value, ['start_at', 'end_exclusive']) &&
    typeof value.start_at === 'string' &&
    typeof value.end_exclusive === 'string'
  );
}

export function isTokenUsage(value: unknown): value is TokenUsage {
  return (
    hasKeys(value, ['input_tokens', 'output_tokens', 'cache_read_tokens', 'cache_write_tokens', 'total_tokens']) &&
    typeof value.input_tokens === 'number' &&
    typeof value.output_tokens === 'number' &&
    typeof value.cache_read_tokens === 'number' &&
    typeof value.cache_write_tokens === 'number' &&
    typeof value.total_tokens === 'number'
  );
}

export function isCostBreakdown(value: unknown): value is CostBreakdown {
  return (
    hasKeys(value, ['input_usd', 'output_usd', 'cache_read_usd', 'cache_write_usd', 'tool_usd', 'flat_usd', 'total_usd']) &&
    typeof value.input_usd === 'number' &&
    typeof value.output_usd === 'number' &&
    typeof value.cache_read_usd === 'number' &&
    typeof value.cache_write_usd === 'number' &&
    typeof value.tool_usd === 'number' &&
    typeof value.flat_usd === 'number' &&
    typeof value.total_usd === 'number'
  );
}

export function isBudgetState(value: unknown): value is BudgetState {
  return (
    hasKeys(value, budgetStateFields) &&
    typeof value.budget_id === 'string' &&
    isMonthlyPeriod(value.period) &&
    typeof value.current_spend_usd === 'number' &&
    typeof value.forecast_spend_usd === 'number' &&
    isNumberArray(value.triggered_threshold_percents) &&
    typeof value.budget_overrun_active === 'boolean' &&
    typeof value.forecast_overrun_active === 'boolean' &&
    typeof value.updated_at === 'string'
  );
}

export function isUsageEntry(value: unknown): value is UsageEntry {
  return (
    hasKeys(value, usageEntryFields) &&
    typeof value.entry_id === 'string' &&
    typeof value.source === 'string' &&
    typeof value.provider === 'string' &&
    typeof value.billing_mode === 'string' &&
    typeof value.occurred_at === 'string' &&
    typeof value.session_id === 'string' &&
    typeof value.external_id === 'string' &&
    typeof value.project_name === 'string' &&
    typeof value.agent_name === 'string' &&
    isStringRecordOrNull(value.metadata) &&
    (value.pricing_ref === null || hasKeys(value.pricing_ref, ['provider', 'model_id', 'pricing_lookup_key'])) &&
    isTokenUsage(value.tokens) &&
    isCostBreakdown(value.cost_breakdown)
  );
}

export function isWasteSummary(value: unknown): value is WasteSummary {
  return (
    hasKeys(value, wasteSummaryFields) &&
    isMonthlyPeriod(value.period) &&
    typeof value.total_waste_cost_usd === 'number' &&
    typeof value.total_spend_cost_usd === 'number' &&
    typeof value.waste_percent === 'number' &&
    typeof value.weekly_waste_cost_usd === 'number' &&
    typeof value.monthly_waste_cost_usd === 'number' &&
    typeof value.projected_month_end_waste_usd === 'number' &&
    Array.isArray(value.by_detector) &&
    Array.isArray(value.top_causes) &&
    Array.isArray(value.daily_trend) &&
    typeof value.generated_at === 'string'
  );
}
