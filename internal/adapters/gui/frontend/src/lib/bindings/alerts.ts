import type { AlertKind } from '$lib/types/notifications';
import type { AlertSeverity, DetectorCategory } from '$lib/types/domain';

import type { DashboardPeriod } from './dashboard';
import { getBindingClient } from './client';

export interface AlertState {
  alertId: string;
  kind: AlertKind | string;
  severity: AlertSeverity | string;
  triggeredAt: string;
  period: DashboardPeriod;
  budgetId: string;
  forecastId: string;
  insightId: string;
  detectorCategory: DetectorCategory | '' | string;
  currentSpendUsd: number;
  limitUsd: number;
  thresholdPercent: number;
}

export interface AlertListResponse {
  items: AlertState[];
  empty: boolean;
}

export async function loadAlerts(month = ''): Promise<AlertListResponse> {
  return getBindingClient().invoke<AlertListResponse>('AlertsBinding', 'LoadAlerts', month);
}
