import type { BillingMode, ProviderName } from '$lib/types/domain';

import { getBindingClient } from './client';

export interface DashboardPeriod {
  month: string;
  startAt: string;
  endExclusive: string;
  currency: string;
}

export interface DashboardTotals {
  variableSpendUsd: number;
  subscriptionSpendUsd: number;
  totalSpendUsd: number;
  currency: string;
}

export interface DashboardProviderSummary {
  provider: ProviderName;
  variableSpendUsd: number;
  subscriptionSpendUsd: number;
  totalSpendUsd: number;
  usageEntryCount: number;
  sessionCount: number;
  currency: string;
}

export interface DashboardBudget {
  budgetId: string;
  name: string;
  provider: ProviderName | '';
  projectHash: string;
  limitUsd: number;
  currentSpendUsd: number;
  remainingUsd: number;
  triggeredThresholdPercents: number[];
  warningThresholdPercent: number;
  criticalThresholdPercent: number;
  budgetOverrunActive: boolean;
  currency: string;
}

export interface DashboardRecentSession {
  sessionId: string;
  provider: ProviderName;
  billingMode: BillingMode | string;
  projectName: string;
  agentName: string;
  modelId: string;
  startedAt: string;
  endedAt: string;
  durationSeconds: number;
  totalCostUsd: number;
  totalTokens: number;
  currency: string;
}

export interface DashboardResponse {
  period: DashboardPeriod;
  totals: DashboardTotals;
  providerSummaries: DashboardProviderSummary[];
  budgets: DashboardBudget[];
  recentSessions: DashboardRecentSession[];
  empty: boolean;
}

export async function loadDashboard(month = ''): Promise<DashboardResponse> {
  return getBindingClient().invoke<DashboardResponse>('DashboardBinding', 'LoadDashboard', month);
}
