import { test, expect } from '@playwright/test';

test.describe('Graphs Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadGraphs(month) {
            return {
              modelTokenUsages: [
                { modelName: 'gpt-4', totalTokens: 10000, inputTokens: 5000, outputTokens: 5000, cacheReadTokens: 0, cacheWriteTokens: 0 },
                { modelName: 'claude-3', totalTokens: 5000, inputTokens: 2500, outputTokens: 2500, cacheReadTokens: 0, cacheWriteTokens: 0 }
              ],
              modelCosts: [
                { modelName: 'gpt-4', totalCostUsd: 0.30 },
                { modelName: 'claude-3', totalCostUsd: 0.15 }
              ],
              dailyTokenTrends: Array.from({ length: 30 }, (_, i) => ({
                date: \`2026-04-\${(i + 1).toString().padStart(2, '0')}T00:00:00Z\`,
                modelBreakdown: [
                  { modelName: 'gpt-4', totalTokens: 300 + i * 10 },
                  { modelName: 'claude-3', totalTokens: 150 + i * 5 }
                ]
              })),
              modelTokenBreakdowns: [
                { modelName: 'gpt-4', inputTokens: 5000, outputTokens: 5000, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 10000 },
                { modelName: 'claude-3', inputTokens: 2500, outputTokens: 2500, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 5000 }
              ]
            };
          }
        `
      });
    });
  });

  test('renders tabs and switches between them', async ({ page }) => {
    await page.goto('/graphs');

    await expect(page.getByTestId('tab-Model-Token-Usage')).toBeVisible();
    await expect(page.getByText('No data available')).not.toBeVisible();
    await expect(page.getByText('GraphsBinding.LoadGraphs failed')).not.toBeVisible();

    await page.getByTestId('tab-Model-Cost').click();
    await expect(page.getByTestId('tab-Model-Cost')).toHaveClass(/border-primary/);
    await expect(page.getByRole('heading', { name: 'Model Cost' })).toBeVisible();
    await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();
    
    await page.getByTestId('tab-Daily-Token-Trend').click();
    await expect(page.getByTestId('tab-Daily-Token-Trend')).toHaveClass(/border-primary/);
    await expect(page.getByRole('heading', { name: 'Daily Token Trend' })).toBeVisible();
    await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-19-graph-tabs.png' });
  });

  test('filters daily token trend by time range', async ({ page }) => {
    await page.goto('/graphs');

    await expect(page.getByTestId('tab-Daily-Token-Trend')).toBeVisible();
    await expect(page.getByText('No data available')).not.toBeVisible();
    await expect(page.getByText('GraphsBinding.LoadGraphs failed')).not.toBeVisible();

    await page.getByTestId('tab-Daily-Token-Trend').click();
    await expect(page.getByRole('heading', { name: 'Daily Token Trend' })).toBeVisible();
    await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();

    await page.getByTestId('time-range-selector').selectOption('7 days');
    
    await expect(page.getByTestId('time-range-selector')).toHaveValue('7 days');

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-19-time-range.png' });
  });
});
