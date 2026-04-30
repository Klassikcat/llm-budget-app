import { test, expect } from '@playwright/test';

test.describe('Usage Tracking Page', () => {
  test.beforeEach(async ({ page }) => {
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
              recentSessions: [
                { sessionId: 's1', provider: 'claude', billingMode: 'direct_api', projectName: 'test-project', agentName: 'sisyphus', modelId: 'claude-3.5-sonnet', startedAt: '2026-04-30T10:00:00Z', endedAt: '2026-04-30T10:05:00Z', durationSeconds: 300, totalCostUsd: 1.50, totalTokens: 1500, currency: 'USD' }
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
        `
      });
    });

    await page.route('**/wailsjs/go/gui/FormsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function SaveManualEntry(input) {
            return {
              result: { success: true },
              entry: {
                entryId: 'e1',
                provider: input.provider,
                modelId: input.modelId,
                occurredAt: input.occurredAt,
                inputTokens: input.inputTokens,
                outputTokens: input.outputTokens,
                cachedTokens: input.cachedTokens,
                cacheWriteTokens: input.cacheWriteTokens,
                projectName: input.projectName,
                metadata: input.metadata,
                totalCostUsd: 0
              }
            };
          }
        `
      });
    });
  });

  test('renders manual entry form and history table', async ({ page }) => {
    await page.goto('/usage');
    
    await expect(page.locator('h1').filter({ hasText: 'Usage Tracking' })).toBeVisible();
    await expect(page.locator('h2').filter({ hasText: 'Manual Entry' })).toBeVisible();
    await expect(page.locator('h2').filter({ hasText: 'History' })).toBeVisible();
    
    await page.getByLabel('Provider').selectOption('claude');
    await page.getByLabel('Model ID').fill('claude-3.5-sonnet');
    await page.getByLabel('Input Tokens').fill('1000');
    await page.getByLabel('Output Tokens').fill('500');
    await page.getByLabel('Cached Tokens').fill('0');
    await page.getByLabel('Cache Write Tokens').fill('0');
    await page.getByLabel('Project Name').fill('test-project');
    
    await page.getByRole('button', { name: 'Save Entry' }).click();
    
    await expect(page.getByText('Usage entry saved')).toBeVisible();
    
    await expect(page.getByText('claude-3.5-sonnet').first()).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-15-manual-entry.png' });
  });

  test('shows validation errors for missing required fields', async ({ page }) => {
    await page.goto('/usage');
    
    await page.getByRole('button', { name: 'Save Entry' }).click();
    
    await expect(page.getByText('Provider is required')).toBeVisible();
    await expect(page.getByText('Model ID is required')).toBeVisible();
    await expect(page.getByText('Project Name is required')).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-15-form-validation.png' });
  });
});
