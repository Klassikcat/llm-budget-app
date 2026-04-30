import { test, expect } from '@playwright/test';

const dashboardModule = `
  export async function LoadDashboard() {
    return {
      period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totals: { variableSpendUsd: 42.5, subscriptionSpendUsd: 0, totalSpendUsd: 42.5, currency: 'USD' },
      providerSummaries: [],
      budgets: [],
      recentSessions: [],
      empty: false
    };
  }
`;

const graphsModule = `
  export async function LoadGraphs() {
    return { modelTokenUsages: [], modelCosts: [], dailyTokenTrends: [], modelTokenBreakdowns: [] };
  }
`;

const insightsModule = `
  export async function LoadWasteSummary() {
    return {
      period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totalWasteCostUsd: 0,
      totalSpendCostUsd: 42.5,
      wastePercent: 0,
      weeklyWasteCostUsd: 0,
      monthlyWasteCostUsd: 0,
      projectedMonthEndWasteUsd: 0,
      byDetector: [],
      topCauses: [],
      dailyTrend: [],
      generatedAt: '2026-04-30T12:00:00Z'
    };
  }
`;

test('dashboard stat card renders correctly', async ({ page }) => {
  await page.route('**/wailsjs/go/gui/DashboardBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: dashboardModule });
  });
  await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: graphsModule });
  });
  await page.route('**/wailsjs/go/gui/InsightsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: insightsModule });
  });

  await page.goto('/');

  await expect(page.getByText('Total Spend')).toBeVisible();
  await expect(page.getByText('$42.50')).toBeVisible();
});
