import { notificationStore } from '$lib/stores/notification';
import type { DashboardBudget } from '$lib/bindings';
import type { GuiNotification, ThresholdAlert } from '$lib/types/notifications';

export type SystemNotificationDispatcher = (notification: GuiNotification) => void;

let systemDispatcher: SystemNotificationDispatcher = () => undefined;

export function setSystemNotificationDispatcher(dispatcher: SystemNotificationDispatcher): void {
  systemDispatcher = dispatcher;
}

export function resetSystemNotificationDispatcher(): void {
  systemDispatcher = () => undefined;
}

export function generateAlertKey(budgetId: string, thresholdPercent: number): string {
  return `${budgetId}-${thresholdPercent}`;
}

function normalizeThresholdPercent(thresholdPercent: number): number {
  return thresholdPercent <= 1 ? thresholdPercent * 100 : thresholdPercent;
}

function getAlertSeverity(
  thresholdPercent: number,
  warningThresholdPercent: number,
  criticalThresholdPercent: number
): ThresholdAlert['severity'] {
  if (thresholdPercent >= normalizeThresholdPercent(criticalThresholdPercent)) {
    return 'critical';
  }

  if (thresholdPercent >= normalizeThresholdPercent(warningThresholdPercent)) {
    return 'warning';
  }

  return 'info';
}

export function checkBudgetThresholds(budgets: DashboardBudget[], periodMonth: string): void {
  for (const budget of budgets) {
    if (!budget.triggeredThresholdPercents || budget.triggeredThresholdPercents.length === 0) {
      continue;
    }

    for (const threshold of budget.triggeredThresholdPercents) {
      const alertKey = generateAlertKey(budget.budgetId, threshold);

      if (!notificationStore.hasSentAlertKey(alertKey)) {
        const alertId = crypto.randomUUID();
        const thresholdDisplay = normalizeThresholdPercent(threshold);
        const thresholdFraction = threshold <= 1 ? threshold : threshold / 100;

        const alertData: ThresholdAlert = {
          alertId,
          kind: 'budget_threshold',
          severity: getAlertSeverity(
            thresholdDisplay,
            budget.warningThresholdPercent,
            budget.criticalThresholdPercent
          ),
          triggeredAt: new Date().toISOString(),
          periodMonth,
          budgetId: budget.budgetId,
          forecastId: '',
          insightId: '',
          detectorCategory: '',
          currentSpendUsd: budget.currentSpendUsd,
          limitUsd: budget.limitUsd,
          thresholdPercent: thresholdFraction
        };

        const notification: GuiNotification = {
          id: alertId,
          title: 'Budget Threshold Exceeded',
          subtitle: budget.name,
          body: `Budget "${budget.name}" has exceeded ${thresholdDisplay}% of its limit ($${budget.limitUsd.toFixed(2)}). Current spend: $${budget.currentSpendUsd.toFixed(2)}.`,
          kind: 'budget_threshold',
          severity: alertData.severity,
          data: alertData as unknown as Record<string, unknown>
        };

        notificationStore.addNotification(notification);
        notificationStore.addSentAlertKey(alertKey);
        systemDispatcher(notification);
      }
    }
  }
}

export function dismissNotification(id: string): void {
  notificationStore.dismissNotification(id);
}

export function clearAllNotifications(): void {
  notificationStore.clearAll();
}

export function markNotificationsAsRead(): void {
  notificationStore.markAsRead();
}
