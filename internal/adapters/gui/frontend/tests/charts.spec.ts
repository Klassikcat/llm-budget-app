import { test, expect } from '@playwright/test';

const graphsModule = `
  export async function LoadGraphs() {
    return {
      modelTokenUsages: [{ modelName: 'claude-3.5-sonnet', totalTokens: 1500, inputTokens: 1000, outputTokens: 500, cacheReadTokens: 0, cacheWriteTokens: 0 }],
      modelCosts: [{ modelName: 'claude-3.5-sonnet', totalCostUsd: 1.5 }],
      dailyTokenTrends: [{ date: '2026-04-30', modelBreakdown: [{ modelName: 'claude-3.5-sonnet', totalTokens: 1500 }] }],
      modelTokenBreakdowns: [{ modelName: 'claude-3.5-sonnet', inputTokens: 1000, outputTokens: 500, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 1500 }]
    };
  }
`;

test('charts render on the graphs route', async ({ page }) => {
  await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: graphsModule });
  });

  await page.goto('/graphs');

  await expect(page.getByRole('main').getByRole('heading', { name: 'Graphs' })).toBeVisible();
  await expect(page.getByText('Model Token Usage').first()).toBeVisible();
  await expect(page.locator('canvas').first()).toBeVisible();

  await page.locator('canvas').first().screenshot({ path: '../../../../.sisyphus/evidence/task-8-line-chart.png' });
});
