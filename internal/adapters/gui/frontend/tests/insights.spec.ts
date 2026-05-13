import { test, expect } from '@playwright/test';

test.describe('Insights', () => {
  test('renders full state', async ({ page }) => {
    await page.route('**/wailsjs/go/gui/InsightsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadWasteSummary(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              totalWasteCostUsd: 15.50,
              totalSpendCostUsd: 100.00,
              wastePercent: 15.5,
              weeklyWasteCostUsd: 5.00,
              monthlyWasteCostUsd: 15.50,
              projectedMonthEndWasteUsd: 20.50,
              byDetector: [],
              topCauses: [
                { category: 'context_avalanche', attributedCostUsd: 10.50, insightCount: 5 },
                { category: 'repeated_file_reads', attributedCostUsd: 5.00, insightCount: 2 }
              ],
              dailyTrend: [
                { day: '2026-04-28T00:00:00Z', wasteCostUsd: 2.50 },
                { day: '2026-04-29T00:00:00Z', wasteCostUsd: 5.00 },
                { day: '2026-04-30T00:00:00Z', wasteCostUsd: 8.00 }
              ],
              generatedAt: '2026-04-30T12:00:00Z'
            };
          }

          export async function LoadInsights(month) {
            return {
              items: [
                {
                  insightId: 'ins-1',
                  category: 'context_avalanche',
                  severity: 'high',
                  detectedAt: '2026-04-30T10:00:00Z',
                  period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
                  payload: {
                    sessionIds: ['sess-1'],
                    usageEntryIds: [],
                    hashes: [{ kind: 'prompt_hash', value: 'abc123def456' }],
                    counts: [{ key: 'turns', value: 5 }],
                    metrics: [{ key: 'wasted_tokens', unit: 'tokens', value: 1000 }]
                  }
                },
                {
                  insightId: 'ins-2',
                  category: 'repeated_file_reads',
                  severity: 'medium',
                  detectedAt: '2026-04-29T15:30:00Z',
                  period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
                  payload: {
                    sessionIds: ['sess-2'],
                    usageEntryIds: [],
                    hashes: [],
                    counts: [{ key: 'read_count', value: 10 }],
                    metrics: []
                  }
                }
              ],
              empty: false
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/AlertsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadAlerts(month) {
            return {
              items: [],
              empty: true
            };
          }
        `
      });
    });

    await page.goto('/insights');
    
    await expect(page.locator('text=Waste Headline')).toBeVisible();
    await expect(page.locator('text=Context Avalanche').first()).toBeVisible();
    
    await expect(page.locator('text=Top Waste Causes')).toBeVisible();
    await expect(page.locator('text=Daily Waste Trend (30-day)')).toBeVisible();
    await expect(page.locator('text=Insights Log')).toBeVisible();
    
    await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-18-insights-full.png', fullPage: true });
  });

  test('opens insight detail modal', async ({ page }) => {
    await page.route('**/wailsjs/go/gui/InsightsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadWasteSummary(month) {
            return {
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              totalWasteCostUsd: 15.50,
              totalSpendCostUsd: 100.00,
              wastePercent: 15.5,
              weeklyWasteCostUsd: 5.00,
              monthlyWasteCostUsd: 15.50,
              projectedMonthEndWasteUsd: 20.50,
              byDetector: [],
              topCauses: [],
              dailyTrend: [],
              generatedAt: '2026-04-30T12:00:00Z'
            };
          }

          export async function LoadInsights(month) {
            return {
              items: [
                {
                  insightId: 'ins-1',
                  category: 'context_avalanche',
                  severity: 'high',
                  detectedAt: '2026-04-30T10:00:00Z',
                  period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
                  payload: {
                    sessionIds: ['sess-1'],
                    usageEntryIds: [],
                    hashes: [{ kind: 'prompt_hash', value: 'abc123def456' }],
                    counts: [{ key: 'turns', value: 5 }],
                    metrics: [{ key: 'wasted_tokens', unit: 'tokens', value: 1000 }]
                  }
                }
              ],
              empty: false
            };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/AlertsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function LoadAlerts(month) {
            return {
              items: [],
              empty: true
            };
          }
        `
      });
    });

    await page.goto('/insights');
    
    await expect(page.locator('text=Context Avalanche').first()).toBeVisible();
    
    await page.locator('text=Context Avalanche').first().click();
    
    await expect(page.locator('text=Insight Details')).toBeVisible();
    await expect(page.locator('text=Payload Details')).toBeVisible();
    await expect(page.locator('text=wasted_tokens')).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-18-insight-detail.png' });
    
    await page.locator('button:has-text("Close")').click();
    
    await expect(page.locator('text=Insight Details')).not.toBeVisible();
  });
});
