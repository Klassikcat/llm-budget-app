import { test, expect } from '@playwright/test';

const dashboardModule = `
  export async function LoadDashboard(month) {
    const manualSaved = localStorage.getItem('task21ManualSaved') === 'true';
    const budgetLimit = Number(localStorage.getItem('task21BudgetLimit') || '150');
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totals: { variableSpendUsd: manualSaved ? 24.5 : 12.5, subscriptionSpendUsd: 20, totalSpendUsd: manualSaved ? 44.5 : 32.5, currency: 'USD' },
      providerSummaries: [
        { provider: 'claude', variableSpendUsd: manualSaved ? 24.5 : 12.5, subscriptionSpendUsd: 0, totalSpendUsd: manualSaved ? 24.5 : 12.5, usageEntryCount: manualSaved ? 2 : 1, sessionCount: manualSaved ? 2 : 1, currency: 'USD' },
        { provider: 'openai', variableSpendUsd: 0, subscriptionSpendUsd: 20, totalSpendUsd: 20, usageEntryCount: 0, sessionCount: 0, currency: 'USD' }
      ],
      budgets: [
        { budgetId: 'task21-budget', name: 'Task 21 Budget', provider: '', projectHash: '', limitUsd: budgetLimit, currentSpendUsd: manualSaved ? 44.5 : 32.5, remainingUsd: budgetLimit - (manualSaved ? 44.5 : 32.5), triggeredThresholdPercents: [], warningThresholdPercent: 80, criticalThresholdPercent: 95, budgetOverrunActive: false, currency: 'USD' }
      ],
      recentSessions: [
        { sessionId: 'base-session', provider: 'claude', billingMode: 'direct_api', projectName: 'baseline-project', agentName: 'sisyphus', modelId: 'claude-3.5-sonnet', startedAt: '2026-04-29T10:00:00Z', endedAt: '2026-04-29T10:05:00Z', durationSeconds: 300, totalCostUsd: 12.5, totalTokens: 1500, currency: 'USD' },
        ...(manualSaved ? [{ sessionId: 'manual-session', provider: 'claude', billingMode: 'direct_api', projectName: 'task21-project', agentName: 'manual', modelId: 'claude-task21', startedAt: '2026-04-30T10:00:00Z', endedAt: '2026-04-30T10:05:00Z', durationSeconds: 300, totalCostUsd: 12, totalTokens: 1400, currency: 'USD' }] : [])
      ],
      empty: false
    };
  }
`;

const graphsModule = `
  export async function LoadGraphs(month) {
    return {
      modelTokenUsages: [{ modelName: 'claude-task21', totalTokens: 1400, inputTokens: 1000, outputTokens: 400, cacheReadTokens: 0, cacheWriteTokens: 0 }],
      modelCosts: [{ modelName: 'claude-task21', totalCostUsd: 12 }],
      dailyTokenTrends: [{ date: '2026-04-30', modelBreakdown: [{ modelName: 'claude-task21', totalTokens: 1400 }] }],
      modelTokenBreakdowns: [{ modelName: 'claude-task21', inputTokens: 1000, outputTokens: 400, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 1400 }]
    };
  }
`;

const insightsModule = `
  export async function LoadWasteSummary(month) {
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totalWasteCostUsd: 4,
      totalSpendCostUsd: 44.5,
      wastePercent: 8.9,
      weeklyWasteCostUsd: 2,
      monthlyWasteCostUsd: 4,
      projectedMonthEndWasteUsd: 6,
      byDetector: [{ category: 'context_avalanche', attributedCostUsd: 4, insightCount: 1 }],
      topCauses: [{ category: 'context_avalanche', attributedCostUsd: 4, insightCount: 1 }],
      dailyTrend: [{ day: '2026-04-30', wasteCostUsd: 4 }],
      generatedAt: '2026-04-30T12:00:00Z'
    };
  }
  export async function LoadInsights(month) {
    return {
      items: [{ insightId: 'task21-insight', category: 'context_avalanche', severity: 'medium', detectedAt: '2026-04-30T12:00:00Z', period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' }, payload: { sessionIds: ['manual-session'], usageEntryIds: [], hashes: [], counts: [], metrics: [{ key: 'waste', unit: 'usd', value: 4 }] } }],
      empty: false
    };
  }
`;

const alertsModule = `
  export async function LoadAlerts(month) {
    return { items: [], empty: true };
  }
`;

const subscriptionsModule = `
  export async function LoadSubscriptions() {
    return {
      items: localStorage.getItem('task21SubscriptionSaved') === 'true' ? [{ subscriptionId: 'openai-task-21-pro-2026-04-30', presetKey: '', provider: 'openai', planName: 'Task 21 Pro', renewalDay: 15, startsAt: '2026-04-30T00:00:00.000Z', endsAt: '', feeUsd: 42, isActive: true }] : [],
      empty: localStorage.getItem('task21SubscriptionSaved') !== 'true'
    };
  }
  export async function DeleteSubscription(subscriptionId) {
    localStorage.removeItem('task21SubscriptionSaved');
    return { success: true };
  }
`;

const formsModule = `
  const settings = {
    providers: { anthropicEnabled: true, openaiEnabled: true, geminiEnabled: false, openRouterEnabled: false },
    cliBillingDefaults: { claudeCode: 'direct_api', codex: 'direct_api', geminiCli: 'direct_api', openCode: 'direct_api' },
    subscriptionDefaults: {
      openai: { enabled: true, planCode: 'chatgpt-plus', planName: 'ChatGPT Plus', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      claude: { enabled: false, planCode: 'claude-pro', planName: 'Claude Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      gemini: { enabled: false, planCode: 'gemini-pro', planName: 'Gemini Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' }
    },
    budgets: { monthlyBudgetUsd: 150, monthlySubscriptionBudgetUsd: 50, monthlyUsageBudgetUsd: 100, warningThresholdPercent: 80, criticalThresholdPercent: 95 },
    notifications: { desktopEnabled: true, tuiEnabled: false, budgetWarnings: true, forecastWarnings: true, providerSyncFailure: true },
    databasePath: '/tmp/llmbudget.sqlite3'
  };
  export async function ListSubscriptionPresets() {
    return { items: [{ key: 'task21-pro', provider: 'openai', planName: 'Task 21 Pro', renewalDay: 15, feeUsd: 42 }] };
  }
  export async function SaveManualEntry(input) {
    localStorage.setItem('task21ManualSaved', 'true');
    return { result: { success: true }, entry: { entryId: 'manual-entry', ...input, totalCostUsd: 12 } };
  }
  export async function SaveSubscription(input) {
    localStorage.setItem('task21SubscriptionSaved', 'true');
    return { result: { success: true }, subscription: { subscriptionId: 'openai-task-21-pro-2026-04-30', ...input } };
  }
  export async function SaveBudget(input) {
    localStorage.setItem('task21BudgetLimit', String(input.limitUsd));
    return { result: { success: true }, budget: input };
  }
  export async function LoadSettings() {
    return { result: { success: true }, settings };
  }
  export async function SaveSettings(input) {
    Object.assign(settings, input);
    return { result: { success: true }, settings };
  }
`;

async function installSuccessfulMocks(page: import('@playwright/test').Page) {
  await page.route('**/wailsjs/go/gui/DashboardBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: dashboardModule });
  });
  await page.route('**/wailsjs/go/gui/GraphsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: graphsModule });
  });
  await page.route('**/wailsjs/go/gui/InsightsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: insightsModule });
  });
  await page.route('**/wailsjs/go/gui/AlertsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: alertsModule });
  });
  await page.route('**/wailsjs/go/gui/SubscriptionLookupBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: subscriptionsModule });
  });
  await page.route('**/wailsjs/go/gui/FormsBinding*', async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body: formsModule });
  });
}

test('task 21 full integration flow refreshes mutated data across screens', async ({ page }) => {
  await installSuccessfulMocks(page);
  await page.goto('/usage');
  await page.evaluate(() => {
    localStorage.removeItem('task21ManualSaved');
    localStorage.removeItem('task21SubscriptionSaved');
    localStorage.removeItem('task21BudgetLimit');
  });
  await page.getByLabel('Provider').selectOption('claude');
  await page.getByLabel('Model ID').fill('claude-task21');
  await page.getByLabel('Input Tokens').fill('1000');
  await page.getByLabel('Output Tokens').fill('400');
  await page.getByLabel('Cached Tokens').fill('0');
  await page.getByLabel('Cache Write Tokens').fill('0');
  await page.getByLabel('Project Name').fill('task21-project');
  await page.getByRole('button', { name: 'Save Entry' }).click();
  await expect(page.getByText('Usage entry saved')).toBeVisible();
  await expect(page.getByText('claude-task21').first()).toBeVisible();

  await page.goto('/subscriptions/new');
  await page.getByLabel('Preset').selectOption('task21-pro');
  await page.getByRole('button', { name: 'Save Subscription' }).click();
  await expect(page).toHaveURL(/\/subscriptions$/);
  await expect(page.getByText('Task 21 Pro')).toBeVisible();

  await page.goto('/budgets');
  await page.getByLabel('Monthly Limit (USD)').fill('175');
  await page.getByRole('button', { name: 'Save Budget' }).click();
  await expect(page.getByText('Budget saved')).toBeVisible();
  await expect(page.getByText('$175.00')).toBeVisible();

  await page.goto('/graphs');
  await expect(page.getByRole('main').getByRole('heading', { name: 'Graphs' })).toBeVisible();
  await expect(page.getByText('Model Token Usage').first()).toBeVisible();

  await page.goto('/insights');
  await expect(page.getByText('Waste Headline')).toBeVisible();
  await expect(page.getByText('Context Avalanche').first()).toBeVisible();

  await page.goto('/settings');
  const notificationToggle = page.locator('input#notification-toggle');
  await expect(notificationToggle).toBeEnabled();
  await notificationToggle.evaluate((element: HTMLInputElement) => element.click());
  await expect(page.getByText('Settings saved')).toBeVisible();

  await page.goto('/');
  await expect(page.getByText('task21-project')).toBeVisible();
  await expect(page.getByText('Task 21 Budget')).toBeVisible();
  await page.screenshot({ path: '../../../../.sisyphus/evidence/task-21-full-integration.png', fullPage: true });
});

test('task 21 binding errors surface visibly', async ({ page }) => {
  await page.route('**/wailsjs/go/gui/FormsBinding*', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/javascript',
      body: `
        export async function LoadSettings() {
          throw new Error('task 21 forced binding failure');
        }
      `
    });
  });

  await page.goto('/settings');
  await expect(page.getByText(/FormsBinding\.LoadSettings failed: task 21 forced binding failure/).first()).toBeVisible();
  await page.screenshot({ path: '../../../../.sisyphus/evidence/task-21-binding-error.png', fullPage: true });
});
