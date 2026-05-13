import { dispatchAlertNotification } from '$lib/bindings';
import { notificationStore } from '$lib/stores/notification';
import type { GuiNotification, ThresholdAlert } from '$lib/types/notifications';
import { setSystemNotificationDispatcher } from './notification';

const desktopNotificationEvent = 'llmbudget:desktop-notification';

type RuntimeModule = {
  EventsOn?: (eventName: string, callback: (...payload: unknown[]) => void) => (() => void) | void;
};

declare global {
  interface Window {
    runtime?: RuntimeModule;
  }
}

function isRuntimeModule(value: unknown): value is RuntimeModule {
  return typeof value === 'object' && value !== null;
}

function toThresholdAlert(notification: GuiNotification): ThresholdAlert | null {
  const data = notification.data;
  if (!data) return null;

  const alertId = typeof data.alertId === 'string' ? data.alertId : notification.id;
  const kind = data.kind === 'budget_threshold' ? data.kind : notification.kind;
  const severity = typeof data.severity === 'string' ? data.severity : notification.severity;
  const triggeredAt = typeof data.triggeredAt === 'string' ? data.triggeredAt : new Date().toISOString();
  const periodMonth = typeof data.periodMonth === 'string' ? data.periodMonth : new Date().toISOString().slice(0, 7);
  const budgetId = typeof data.budgetId === 'string' ? data.budgetId : '';
  const currentSpendUsd = typeof data.currentSpendUsd === 'number' ? data.currentSpendUsd : 0;
  const limitUsd = typeof data.limitUsd === 'number' ? data.limitUsd : 0;
  const thresholdPercent = typeof data.thresholdPercent === 'number' ? data.thresholdPercent : 0;

  if (kind !== 'budget_threshold' || (severity !== 'info' && severity !== 'warning' && severity !== 'critical')) {
    return null;
  }

  return {
    alertId,
    kind,
    severity,
    triggeredAt,
    periodMonth,
    budgetId,
    forecastId: typeof data.forecastId === 'string' ? data.forecastId : '',
    insightId: typeof data.insightId === 'string' ? data.insightId : '',
    detectorCategory: '',
    currentSpendUsd,
    limitUsd,
    thresholdPercent
  };
}

export function getWailsRuntime(): RuntimeModule | null {
  if (typeof window === 'undefined') return null;
  return isRuntimeModule(window.runtime) ? window.runtime : null;
}

async function loadRuntime(): Promise<RuntimeModule | null> {
  return getWailsRuntime();
}

export async function wireWailsNotifications(): Promise<() => void> {
  const runtime = await loadRuntime();

  setSystemNotificationDispatcher((notification) => {
    const alert = toThresholdAlert(notification);
    if (alert) {
      void dispatchAlertNotification(alert).catch(() => undefined);
    }
  });

  if (!runtime?.EventsOn) {
    return () => setSystemNotificationDispatcher(() => undefined);
  }

  const unsubscribe = runtime.EventsOn(desktopNotificationEvent, (payload) => {
    notificationStore.addNotification(payload as GuiNotification);
  });

  return () => {
    setSystemNotificationDispatcher(() => undefined);
    if (typeof unsubscribe === 'function') unsubscribe();
  };
}
