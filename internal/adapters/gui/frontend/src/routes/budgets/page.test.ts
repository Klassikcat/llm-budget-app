import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import Page from './+page.svelte';
import { budget } from '$lib/stores/budget';
import { notificationStore } from '$lib/stores/notification';

vi.mock('$lib/stores/budget', () => ({
  budget: {
    subscribe: vi.fn(),
    load: vi.fn(),
    refresh: vi.fn(),
    save: vi.fn()
  },
  loadBudget: vi.fn(),
  saveBudgetAndRefresh: vi.fn()
}));

vi.mock('$lib/stores/notification', () => ({
  notificationStore: {
    subscribe: vi.fn(),
    addNotification: vi.fn(),
    removeNotification: vi.fn()
  }
}));

vi.mock('echarts', () => ({
  init: vi.fn(() => ({
    setOption: vi.fn(),
    resize: vi.fn(),
    dispose: vi.fn()
  }))
}));

global.ResizeObserver = class ResizeObserver {
  observe() {}
  unobserve() {}
  disconnect() {}
};

describe('Budgets Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    
    const mockBudgetState = {
      data: {
        dashboard: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totals: { variableSpendUsd: 0, subscriptionSpendUsd: 0, totalSpendUsd: 0, currency: 'USD' },
          providerSummaries: [],
          budgets: [],
          recentSessions: [],
          empty: true
        },
        budgets: []
      },
      loading: false,
      error: null
    };
    
    vi.mocked(budget.subscribe).mockImplementation((fn) => {
      fn(mockBudgetState);
      return () => {};
    });
    
    vi.mocked(notificationStore.subscribe).mockImplementation((fn) => {
      fn({ items: [], unreadCount: 0, lastDispatchedAt: null, permission: 'default', sentAlertKeys: new Set() });
      return () => {};
    });
  });

  it('renders the budget settings form', () => {
    render(Page);
    
    expect(screen.getByText('Budget Management')).toBeInTheDocument();
    expect(screen.getByLabelText(/Monthly Limit/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Warning Threshold/)).toBeInTheDocument();
    expect(screen.getByLabelText(/Critical Threshold/)).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Save Budget' })).toBeInTheDocument();
  });

  it('shows empty state when no budget exists', () => {
    render(Page);
    expect(screen.getByText('No Budget Set')).toBeInTheDocument();
  });

  it('shows monitoring panels when budget exists', () => {
    const mockBudgetState = {
      data: {
        dashboard: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totals: { variableSpendUsd: 9, subscriptionSpendUsd: 0, totalSpendUsd: 9, currency: 'USD' },
          providerSummaries: [
            { provider: 'openai', totalSpendUsd: 9, variableSpendUsd: 9, subscriptionSpendUsd: 0, usageEntryCount: 1, sessionCount: 1, currency: 'USD' }
          ],
          budgets: [
            {
              budgetId: 'test-budget',
              name: 'Monthly Budget',
              provider: '',
              projectHash: '',
              limitUsd: 10,
              currentSpendUsd: 9,
              remainingUsd: 1,
              triggeredThresholdPercents: [0.8],
              warningThresholdPercent: 80,
              criticalThresholdPercent: 95,
              budgetOverrunActive: false,
              currency: 'USD'
            }
          ],
          recentSessions: [
            {
              sessionId: '1',
              provider: 'openai',
              billingMode: 'variable',
              projectName: 'test',
              agentName: 'test',
              modelId: 'gpt-4',
              startedAt: '2026-04-15T10:00:00Z',
              endedAt: '2026-04-15T10:05:00Z',
              durationSeconds: 300,
              totalCostUsd: 9,
              totalTokens: 1000,
              currency: 'USD'
            }
          ],
          empty: false
        },
        budgets: [
          {
            budgetId: 'test-budget',
            name: 'Monthly Budget',
            provider: '',
            projectHash: '',
            limitUsd: 10,
            currentSpendUsd: 9,
            remainingUsd: 1,
            triggeredThresholdPercents: [0.8],
            warningThresholdPercent: 80,
            criticalThresholdPercent: 95,
            budgetOverrunActive: false,
            currency: 'USD'
          }
        ]
      },
      loading: false,
      error: null
    };
    
    vi.mocked(budget.subscribe).mockImplementation((fn) => {
      fn(mockBudgetState);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('Current Month Progress')).toBeInTheDocument();
    expect(screen.getByText('Provider Costs')).toBeInTheDocument();
    expect(screen.getByText('Cumulative Spend')).toBeInTheDocument();
    
    expect(screen.getByText('Warning Threshold Reached')).toBeInTheDocument();
    expect(screen.getByText('Warning: 90% of budget used.')).toBeInTheDocument();
  });

  it('submits the form and shows success notification', async () => {
    const { saveBudgetAndRefresh } = await import('$lib/stores/budget');
    vi.mocked(saveBudgetAndRefresh).mockResolvedValue({
      result: { success: true },
      budget: {
        budgetId: 'test',
        name: 'Monthly Budget',
        provider: '',
        projectHash: '',
        periodMonth: '2026-04',
        limitUsd: 100,
        warningThresholdPercent: 80,
        criticalThresholdPercent: 95,
        currency: 'USD'
      }
    });

    render(Page);
    
    const limitInput = screen.getByLabelText(/Monthly Limit/);
    await fireEvent.input(limitInput, { target: { value: '100' } });
    
    const saveButton = screen.getByRole('button', { name: 'Save Budget' });
    await fireEvent.click(saveButton);
    
    await waitFor(() => {
      expect(saveBudgetAndRefresh).toHaveBeenCalled();
      expect(notificationStore.addNotification).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Success',
          body: 'Budget saved'
        })
      );
    });
  });
});
