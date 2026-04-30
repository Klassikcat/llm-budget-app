import type { AlertSeverity, DetectorCategory } from './domain';

export type AlertKind =
  | 'budget_threshold'
  | 'budget_overrun'
  | 'forecast_overrun'
  | 'insight_detected';

export interface ThresholdAlert {
  alertId: string;
  kind: AlertKind;
  severity: AlertSeverity;
  triggeredAt: string;
  periodMonth: string;
  budgetId: string;
  forecastId: string;
  insightId: string;
  detectorCategory: DetectorCategory | '';
  currentSpendUsd: number;
  limitUsd: number;
  thresholdPercent: number;
}

export interface GuiNotification {
  id: string;
  title: string;
  subtitle?: string;
  body: string;
  kind: string;
  severity: AlertSeverity | string;
  data?: Record<string, unknown>;
}

export interface NotificationState {
  items: GuiNotification[];
  unreadCount: number;
  lastDispatchedAt: string | null;
  permission: 'default' | 'granted' | 'denied' | 'unsupported';
}

export interface NotificationDispatchResponse {
  result: {
    success: boolean;
    error?: {
      code: string;
      field?: string;
      message: string;
    } | null;
  };
  dispatched: boolean;
}

export const thresholdAlertFields = [
  'alertId',
  'kind',
  'severity',
  'triggeredAt',
  'periodMonth',
  'budgetId',
  'forecastId',
  'insightId',
  'detectorCategory',
  'currentSpendUsd',
  'limitUsd',
  'thresholdPercent'
] as const satisfies readonly (keyof ThresholdAlert)[];

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

export function isThresholdAlert(value: unknown): value is ThresholdAlert {
  return (
    isRecord(value) &&
    thresholdAlertFields.every((field) => field in value) &&
    typeof value.alertId === 'string' &&
    typeof value.kind === 'string' &&
    typeof value.severity === 'string' &&
    typeof value.triggeredAt === 'string' &&
    typeof value.periodMonth === 'string' &&
    typeof value.budgetId === 'string' &&
    typeof value.forecastId === 'string' &&
    typeof value.insightId === 'string' &&
    typeof value.detectorCategory === 'string' &&
    typeof value.currentSpendUsd === 'number' &&
    typeof value.limitUsd === 'number' &&
    typeof value.thresholdPercent === 'number'
  );
}
