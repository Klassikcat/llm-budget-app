import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { notificationStore } from '$lib/stores/notification';
import { 
  checkBudgetThresholds, 
  generateAlertKey, 
  clearAllNotifications,
  setSystemNotificationDispatcher,
  resetSystemNotificationDispatcher
} from './notification';
import type { DashboardBudget } from '$lib/bindings';

describe('Notification Service', () => {
  beforeEach(() => {
    notificationStore.reset();
    resetSystemNotificationDispatcher();
  });

  it('should create threshold alerts for triggered budgets and call dispatcher', () => {
    const dispatcher = vi.fn();
    setSystemNotificationDispatcher(dispatcher);

    const budgets: DashboardBudget[] = [
      {
        budgetId: 'b1',
        name: 'Test Budget',
        provider: 'openai',
        projectHash: 'hash1',
        limitUsd: 100,
        currentSpendUsd: 85,
        remainingUsd: 15,
        triggeredThresholdPercents: [80],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ];

    checkBudgetThresholds(budgets, '2023-10');

    const state = get(notificationStore);
    expect(state.items.length).toBe(1);
    expect(state.items[0].title).toBe('Budget Threshold Exceeded');
    expect(state.items[0].subtitle).toBe('Test Budget');
    expect(state.items[0].severity).toBe('warning');
    expect(state.unreadCount).toBe(1);
    
    const alertKey = generateAlertKey('b1', 80);
    expect(state.sentAlertKeys.has(alertKey)).toBe(true);

    expect(dispatcher).toHaveBeenCalledTimes(1);
    expect(dispatcher).toHaveBeenCalledWith(expect.objectContaining({
      id: state.items[0].id,
      title: 'Budget Threshold Exceeded'
    }));
  });

  it('should prevent duplicate alerts for the same budget and threshold and not call dispatcher again', () => {
    const dispatcher = vi.fn();
    setSystemNotificationDispatcher(dispatcher);

    const budgets: DashboardBudget[] = [
      {
        budgetId: 'b1',
        name: 'Test Budget',
        provider: 'openai',
        projectHash: 'hash1',
        limitUsd: 100,
        currentSpendUsd: 85,
        remainingUsd: 15,
        triggeredThresholdPercents: [80],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ];

    checkBudgetThresholds(budgets, '2023-10');
    expect(get(notificationStore).items.length).toBe(1);
    expect(dispatcher).toHaveBeenCalledTimes(1);

    checkBudgetThresholds(budgets, '2023-10');
    expect(get(notificationStore).items.length).toBe(1);
    expect(dispatcher).toHaveBeenCalledTimes(1);
  });

  it('should allow alerts for different thresholds on the same budget', () => {
    const dispatcher = vi.fn();
    setSystemNotificationDispatcher(dispatcher);

    const budgets1: DashboardBudget[] = [
      {
        budgetId: 'b1',
        name: 'Test Budget',
        provider: 'openai',
        projectHash: 'hash1',
        limitUsd: 100,
        currentSpendUsd: 85,
        remainingUsd: 15,
        triggeredThresholdPercents: [80],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ];

    checkBudgetThresholds(budgets1, '2023-10');
    expect(get(notificationStore).items.length).toBe(1);
    expect(dispatcher).toHaveBeenCalledTimes(1);

    const budgets2: DashboardBudget[] = [
      {
        budgetId: 'b1',
        name: 'Test Budget',
        provider: 'openai',
        projectHash: 'hash1',
        limitUsd: 100,
        currentSpendUsd: 105,
        remainingUsd: -5,
        triggeredThresholdPercents: [80, 100],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: true,
        currency: 'USD'
      }
    ];

    checkBudgetThresholds(budgets2, '2023-10');
    expect(get(notificationStore).items.length).toBe(2);
    expect(dispatcher).toHaveBeenCalledTimes(2);
    
    const state = get(notificationStore);
    expect(state.items[0].severity).toBe('critical');
    expect(state.items[1].severity).toBe('warning');
  });

  it('should allow alerts for the same threshold on different budgets', () => {
    const dispatcher = vi.fn();
    setSystemNotificationDispatcher(dispatcher);

    const budgets: DashboardBudget[] = [
      {
        budgetId: 'b1',
        name: 'Budget 1',
        provider: 'openai',
        projectHash: 'hash1',
        limitUsd: 100,
        currentSpendUsd: 85,
        remainingUsd: 15,
        triggeredThresholdPercents: [80],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      },
      {
        budgetId: 'b2',
        name: 'Budget 2',
        provider: 'anthropic',
        projectHash: 'hash2',
        limitUsd: 200,
        currentSpendUsd: 170,
        remainingUsd: 30,
        triggeredThresholdPercents: [80],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ];

    checkBudgetThresholds(budgets, '2023-10');
    expect(get(notificationStore).items.length).toBe(2);
    expect(dispatcher).toHaveBeenCalledTimes(2);
  });

  it('should clear all notifications', () => {
    const budgets: DashboardBudget[] = [
      {
        budgetId: 'b1',
        name: 'Test Budget',
        provider: 'openai',
        projectHash: 'hash1',
        limitUsd: 100,
        currentSpendUsd: 85,
        remainingUsd: 15,
        triggeredThresholdPercents: [80],
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        budgetOverrunActive: false,
        currency: 'USD'
      }
    ];

    checkBudgetThresholds(budgets, '2023-10');
    expect(get(notificationStore).items.length).toBe(1);

    clearAllNotifications();
    expect(get(notificationStore).items.length).toBe(0);
    expect(get(notificationStore).unreadCount).toBe(0);
    
    expect(get(notificationStore).sentAlertKeys.has(generateAlertKey('b1', 80))).toBe(true);
  });
});
