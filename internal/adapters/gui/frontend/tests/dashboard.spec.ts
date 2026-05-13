import { test, expect } from '@playwright/test';

test.describe('Dashboard', () => {
  test('renders empty state', async ({ page }) => {
    // Mock the dashboard binding to return empty data
    await page.route('**/wailsjs/go/gui/DashboardBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadDashboard(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              totals: { variableSpendUsd: 0, subscriptionSpendUsd: 0, totalSpendUsd: 0, currency: 'USD' },
              providerSummaries: [],
              budgets: [],
              recentSessions: [],
              empty: true
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadGraphs(month) {
            return {
              modelTokenUsages: [],
              modelCosts: [],
              dailyTokenTrends: [],
              modelTokenBreakdowns: []
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/InsightsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadWasteSummary(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
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
        `
      });
    });

    await page.goto('/');
    
    // Wait for the empty state to appear
    await expect(page.locator('text=No Data Available')).toBeVisible();
    
    // Take screenshot
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-14-dashboard-empty.png', fullPage: true });
  });

  test('renders full state', async ({ page }) => {
    // Mock the dashboard binding to return populated data
    await page.route('**/wailsjs/go/gui/DashboardBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadDashboard(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              totals: { variableSpendUsd: 100.50, subscriptionSpendUsd: 50.00, totalSpendUsd: 150.50, currency: 'USD' },
              providerSummaries: [
                { provider: 'openai', variableSpendUsd: 60.00, subscriptionSpendUsd: 0, totalSpendUsd: 60.00, usageEntryCount: 10, sessionCount: 5, currency: 'USD' },
                { provider: 'anthropic', variableSpendUsd: 40.50, subscriptionSpendUsd: 0, totalSpendUsd: 40.50, usageEntryCount: 8, sessionCount: 4, currency: 'USD' },
                { provider: 'github', variableSpendUsd: 0, subscriptionSpendUsd: 50.00, totalSpendUsd: 50.00, usageEntryCount: 0, sessionCount: 0, currency: 'USD' }
              ],
              budgets: [
                { budgetId: 'b1', name: 'Development', provider: '', projectHash: '', limitUsd: 200.00, currentSpendUsd: 150.50, remainingUsd: 49.50, triggeredThresholdPercents: [50], warningThresholdPercent: 50, criticalThresholdPercent: 90, budgetOverrunActive: false, currency: 'USD' },
                { budgetId: 'b2', name: 'OpenAI Only', provider: 'openai', projectHash: '', limitUsd: 50.00, currentSpendUsd: 60.00, remainingUsd: -10.00, triggeredThresholdPercents: [50, 80, 100], warningThresholdPercent: 50, criticalThresholdPercent: 80, budgetOverrunActive: true, currency: 'USD' }
              ],
              recentSessions: [
                { sessionId: 's1', provider: 'openai', billingMode: 'direct_api', projectName: 'llm-budget-tracker', agentName: 'sisyphus', modelId: 'gpt-4-turbo', startedAt: '2026-04-29T10:00:00Z', endedAt: '2026-04-29T10:05:00Z', durationSeconds: 300, totalCostUsd: 1.50, totalTokens: 15000, currency: 'USD' },
                { sessionId: 's2', provider: 'anthropic', billingMode: 'direct_api', projectName: 'llm-budget-tracker', agentName: 'sisyphus', modelId: 'claude-3-opus', startedAt: '2026-04-30T09:00:00Z', endedAt: '2026-04-30T09:10:00Z', durationSeconds: 600, totalCostUsd: 2.50, totalTokens: 20000, currency: 'USD' }
              ],
              empty: false
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadGraphs(month) {
            return {
              modelTokenUsages: [
                { modelName: 'gpt-4-turbo', totalTokens: 15000, inputTokens: 10000, outputTokens: 5000, cacheReadTokens: 0, cacheWriteTokens: 0 },
                { modelName: 'claude-3-opus', totalTokens: 20000, inputTokens: 15000, outputTokens: 5000, cacheReadTokens: 0, cacheWriteTokens: 0 }
              ],
              modelCosts: [],
              dailyTokenTrends: [
                { date: '2026-04-28T00:00:00Z', modelBreakdown: [{ modelName: 'gpt-4-turbo', totalTokens: 5000 }] },
                { date: '2026-04-29T00:00:00Z', modelBreakdown: [{ modelName: 'gpt-4-turbo', totalTokens: 10000 }, { modelName: 'claude-3-opus', totalTokens: 5000 }] },
                { date: '2026-04-30T00:00:00Z', modelBreakdown: [{ modelName: 'gpt-4-turbo', totalTokens: 15000 }, { modelName: 'claude-3-opus', totalTokens: 20000 }] }
              ],
              modelTokenBreakdowns: []
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/InsightsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadWasteSummary(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              totalWasteCostUsd: 15.05,
              totalSpendCostUsd: 150.50,
              wastePercent: 10.0,
              weeklyWasteCostUsd: 5.00,
              monthlyWasteCostUsd: 15.05,
              projectedMonthEndWasteUsd: 15.05,
              byDetector: [],
              topCauses: [],
              dailyTrend: [],
              generatedAt: '2026-04-30T12:00:00Z'
            };
          }
        `
      });
    });

    await page.goto('/');
    
    // Wait for the dashboard to load
    await expect(page.locator('text=Total Spend')).toBeVisible();
    await expect(page.locator('text=$150.50').first()).toBeVisible();
    
    // Assert Daily Cost Trend is visible
    await expect(page.locator('text=Daily Cost Trend')).toBeVisible();
    
    await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();
    
    // Take screenshot
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-14-dashboard-full.png', fullPage: true });
  });
});
