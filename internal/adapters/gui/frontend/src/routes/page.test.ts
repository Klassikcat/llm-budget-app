import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/svelte';
import Page from './+page.svelte';
import { dashboardStore, type DashboardStoreState } from '$lib/stores/dashboard';

vi.mock('$lib/stores/dashboard', () => ({
  dashboardStore: {
    subscribe: vi.fn(),
    load: vi.fn(),
    refresh: vi.fn()
  },
  loadDashboardData: vi.fn(),
  refreshDashboardData: vi.fn()
}));

vi.mock('$lib/components/charts/echartsAction', () => ({
  chartAction: () => ({
    destroy: vi.fn(),
    update: vi.fn()
  })
}));

describe('Dashboard Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when dashboard is empty', () => {
    const mockStore: DashboardStoreState = {
      data: {
        dashboard: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totals: { variableSpendUsd: 0, subscriptionSpendUsd: 0, totalSpendUsd: 0, currency: 'USD' },
          providerSummaries: [],
          budgets: [],
          recentSessions: [],
          empty: true
        },
        graphs: null,
        waste: null
      },
      loading: false,
      error: null
    };
    
    vi.mocked(dashboardStore.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('No Data Available')).toBeInTheDocument();
    expect(screen.getByText(/There is no usage data or budgets configured/)).toBeInTheDocument();
  });

  it('renders dashboard with data', () => {
    const mockStore: DashboardStoreState = {
      data: {
        dashboard: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totals: {
            variableSpendUsd: 100.50,
            subscriptionSpendUsd: 50.00,
            totalSpendUsd: 150.50,
            currency: 'USD'
          },
          providerSummaries: [
            { provider: 'openai', variableSpendUsd: 100.50, subscriptionSpendUsd: 0, totalSpendUsd: 100.50, usageEntryCount: 10, sessionCount: 5, currency: 'USD' }
          ],
          budgets: [],
          recentSessions: [],
          empty: false
        },
        graphs: {
          modelTokenUsages: [
            { modelName: 'gpt-4-turbo', totalTokens: 10000, inputTokens: 5000, outputTokens: 5000, cacheReadTokens: 0, cacheWriteTokens: 0 }
          ],
          modelCosts: [],
          dailyTokenTrends: [],
          modelTokenBreakdowns: []
        },
        waste: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totalWasteCostUsd: 15.5,
          totalSpendCostUsd: 100,
          wastePercent: 15.5,
          weeklyWasteCostUsd: 5,
          monthlyWasteCostUsd: 15.5,
          projectedMonthEndWasteUsd: 15.5,
          byDetector: [],
          topCauses: [],
          dailyTrend: [],
          generatedAt: '2026-04-30T12:00:00Z'
        }
      },
      loading: false,
      error: null
    };
    
    vi.mocked(dashboardStore.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('Total Spend')).toBeInTheDocument();
    expect(screen.getByText('$150.50')).toBeInTheDocument();
    
    expect(screen.getByText('Total Tokens')).toBeInTheDocument();
    expect(screen.getByText('10,000')).toBeInTheDocument();
    
    expect(screen.getByText('Subscription Cost')).toBeInTheDocument();
    expect(screen.getByText('$50.00')).toBeInTheDocument();
    
    expect(screen.getByText('Waste %')).toBeInTheDocument();
    expect(screen.getByText('15.5%')).toBeInTheDocument();
    
    expect(screen.getByText('Daily Cost Trend')).toBeInTheDocument();
    expect(screen.getByText('Provider Costs')).toBeInTheDocument();
    expect(screen.getByText('Budgets')).toBeInTheDocument();
    expect(screen.getByText('Recent Sessions')).toBeInTheDocument();
  });

  it('shows error message when error exists', () => {
    const mockStore: DashboardStoreState = {
      data: {
        dashboard: null,
        graphs: null,
        waste: null
      },
      loading: false,
      error: 'Failed to load data'
    };
    
    vi.mocked(dashboardStore.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('Failed to load data')).toBeInTheDocument();
  });
});
