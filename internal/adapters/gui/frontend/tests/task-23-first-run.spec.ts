import { expect, type Page, test } from '@playwright/test';

const dashboardModule = `
  export async function LoadDashboard(month) {
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totals: { variableSpendUsd: 0, subscriptionSpendUsd: 0, totalSpendUsd: 0, currency: 'USD' },
      providerSummaries: [],
      budgets: [],
      recentSessions: [],
      empty: true
    };
  }
`;

const graphsModule = `
  export async function LoadGraphs(month) {
    return {
      modelTokenUsages: [],
      modelCosts: [],
      dailyTokenTrends: [],
      modelTokenBreakdowns: []
    };
  }
`;

const insightsModule = `
  export async function LoadWasteSummary(month) {
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totalWasteCostUsd: 0,
      totalSpendCostUsd: 0,
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

const alertsModule = `
  export async function LoadAlerts(month) {
    return { items: [], empty: true };
  }
`;

async function mockModule(page: Page, pattern: string, body: string) {
  await page.route(pattern, async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body });
  });
}

test('empty database first run loads dashboard without crashing', async ({ page }) => {
  const pageErrors: Error[] = [];
  page.on('pageerror', (error) => pageErrors.push(error));

  await mockModule(page, '**/wailsjs/go/gui/DashboardBinding*', dashboardModule);
  await mockModule(page, '**/wailsjs/go/gui/GraphsBinding*', graphsModule);
  await mockModule(page, '**/wailsjs/go/gui/InsightsBinding*', insightsModule);
  await mockModule(page, '**/wailsjs/go/gui/AlertsBinding*', alertsModule);
  await page.addInitScript(() => {
    window.localStorage.clear();
    window.localStorage.setItem('llm-budget-tracker-theme', 'dark');
  });

  await page.goto('/');

  await expect(page).toHaveTitle('Dashboard - LLM Budget Tracker');
  await expect(page.getByRole('main').getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(page.getByText('No Data Available')).toBeVisible();
  await expect(page.getByText('There is no usage data or budgets configured for the current period.')).toBeVisible();
  expect(pageErrors).toHaveLength(0);

  await page.screenshot({ path: '../../../../.sisyphus/evidence/task-23-first-run.png', fullPage: true });
});
