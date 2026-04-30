import { test, expect } from '@playwright/test';

test.describe('Budgets', () => {
  test('shows warning alert when budget threshold is reached', async ({ page }) => {
    await page.route('**/wailsjs/go/gui/DashboardBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadDashboard(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              totals: { variableSpendUsd: 9, subscriptionSpendUsd: 0, totalSpendUsd: 9, currency: 'USD' },
              providerSummaries: [
                { provider: 'openai', variableSpendUsd: 9, subscriptionSpendUsd: 0, totalSpendUsd: 9, usageEntryCount: 1, sessionCount: 1, currency: 'USD' }
              ],
              budgets: [
                { 
                  budgetId: 'b1', 
                  name: 'Monthly Budget', 
                  provider: '', 
                  projectHash: '', 
                  limitUsd: 10.00, 
                  currentSpendUsd: 9.00, 
                  remainingUsd: 1.00, 
                  triggeredThresholdPercents: [0.8], 
                  warningThresholdPercent: 80,
                  criticalThresholdPercent: 95,
                  budgetOverrunActive: false, 
                  currency: 'USD' 
                }
              ],
              recentSessions: [
                { sessionId: 's1', provider: 'openai', billingMode: 'direct_api', projectName: 'test', agentName: 'test', modelId: 'gpt-4', startedAt: '2026-04-15T10:00:00Z', endedAt: '2026-04-15T10:05:00Z', durationSeconds: 300, totalCostUsd: 9.00, totalTokens: 1000, currency: 'USD' }
              ],
              empty: false
            };
          }
        `
      });
    });

    await page.goto('/budgets');
    
    await expect(page.locator('text=Budget Management')).toBeVisible();
    
    await expect(page.locator('text=Warning Threshold Reached')).toBeVisible();
    await expect(page.locator('text=Warning: 90% of budget used.')).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-17-budget-warning.png', fullPage: true });
  });

  test('sets budget and shows success toast', async ({ page }) => {
    await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadGraphs(month) {
            return { modelTokenUsages: [], modelCosts: [], dailyTokenTrends: [], modelTokenBreakdowns: [] };
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
          export async function SaveBudget(input) {
            window.__mockBudgetSaved = true;
            return {
              result: { success: true },
              budget: {
                budgetId: input.budgetId,
                name: input.name,
                provider: input.provider,
                projectHash: input.projectHash,
                periodMonth: input.periodMonth,
                limitUsd: input.limitUsd,
                warningThresholdPercent: input.warningThresholdPercent,
                criticalThresholdPercent: input.criticalThresholdPercent,
                currency: input.currency
              }
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/DashboardBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadDashboard(month) {
            if (window.__mockBudgetSaved) {
              return {
                period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
                totals: { variableSpendUsd: 0, subscriptionSpendUsd: 0, totalSpendUsd: 0, currency: 'USD' },
                providerSummaries: [],
                budgets: [
                  { 
                    budgetId: 'b1', 
                    name: 'Monthly Budget', 
                    provider: '', 
                    projectHash: '', 
                    limitUsd: 100.00, 
                    currentSpendUsd: 0.00, 
                    remainingUsd: 100.00, 
                    triggeredThresholdPercents: [], 
                    warningThresholdPercent: 80,
                    criticalThresholdPercent: 95,
                    budgetOverrunActive: false, 
                    currency: 'USD' 
                  }
                ],
                recentSessions: [],
                empty: false
              };
            } else {
              return {
                period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
                totals: { variableSpendUsd: 0, subscriptionSpendUsd: 0, totalSpendUsd: 0, currency: 'USD' },
                providerSummaries: [],
                budgets: [],
                recentSessions: [],
                empty: true
              };
            }
          }
        `
      });
    });

    await page.goto('/budgets');
    
    await expect(page.locator('text=Budget Management')).toBeVisible();
    
    await page.fill('input[id="limitUsd"]', '100');
    await page.fill('input[id="warningThresholdPercent"]', '80');
    await page.fill('input[id="criticalThresholdPercent"]', '95');
    
    await page.click('button:has-text("Save Budget")');
    
    await expect(page.locator('text=Budget saved')).toBeVisible();
    
    await expect(page.locator('text=of $100.00')).toBeVisible();
    
    await expect(page.getByText('$0.00').first()).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-17-budget-set.png', fullPage: true });
  });
});
