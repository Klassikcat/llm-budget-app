import { expect, type Page, test } from '@playwright/test';

const dashboardModule = `
  export async function LoadDashboard(month) {
    const manualSaved = localStorage.getItem('task22ManualSaved') === 'true';
    const budgetLimit = Number(localStorage.getItem('task22BudgetLimit') || '120');
    const currentSpend = manualSaved ? 108 : 96;
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totals: { variableSpendUsd: currentSpend - 25, subscriptionSpendUsd: 25, totalSpendUsd: currentSpend, currency: 'USD' },
      providerSummaries: [
        { provider: 'openai', variableSpendUsd: manualSaved ? 83 : 71, subscriptionSpendUsd: 25, totalSpendUsd: currentSpend, usageEntryCount: manualSaved ? 3 : 2, sessionCount: manualSaved ? 3 : 2, currency: 'USD' }
      ],
      budgets: [
        { budgetId: 'task22-budget', name: 'Task 22 Monthly Budget', provider: '', projectHash: '', limitUsd: budgetLimit, currentSpendUsd: currentSpend, remainingUsd: budgetLimit - currentSpend, triggeredThresholdPercents: [80], warningThresholdPercent: 80, criticalThresholdPercent: 95, budgetOverrunActive: currentSpend >= budgetLimit, currency: 'USD' }
      ],
      recentSessions: [
        { sessionId: 'task22-base', provider: 'openai', billingMode: 'direct_api', projectName: 'task22-baseline', agentName: 'sisyphus', modelId: 'gpt-task22', startedAt: '2026-04-28T10:00:00Z', endedAt: '2026-04-28T10:05:00Z', durationSeconds: 300, totalCostUsd: 71, totalTokens: 7100, currency: 'USD' },
        ...(manualSaved ? [{ sessionId: 'task22-manual', provider: 'openai', billingMode: 'direct_api', projectName: 'task22-manual-project', agentName: 'manual', modelId: 'gpt-task22-manual', startedAt: '2026-04-30T10:00:00Z', endedAt: '2026-04-30T10:05:00Z', durationSeconds: 300, totalCostUsd: 12, totalTokens: 1200, currency: 'USD' }] : [])
      ],
      empty: false
    };
  }
`;

const emptyDashboardModule = `
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
      modelTokenUsages: [
        { modelName: 'gpt-task22', totalTokens: 7100, inputTokens: 5000, outputTokens: 2100, cacheReadTokens: 0, cacheWriteTokens: 0 },
        { modelName: 'claude-task22', totalTokens: 4200, inputTokens: 3000, outputTokens: 1200, cacheReadTokens: 0, cacheWriteTokens: 0 }
      ],
      modelCosts: [
        { modelName: 'gpt-task22', totalCostUsd: 71 },
        { modelName: 'claude-task22', totalCostUsd: 42 }
      ],
      dailyTokenTrends: Array.from({ length: 30 }, (_, index) => ({
        date: '2026-04-' + String(index + 1).padStart(2, '0') + 'T00:00:00Z',
        modelBreakdown: [
          { modelName: 'gpt-task22', totalTokens: 100 + index * 20 },
          { modelName: 'claude-task22', totalTokens: 80 + index * 10 }
        ]
      })),
      modelTokenBreakdowns: [
        { modelName: 'gpt-task22', inputTokens: 5000, outputTokens: 2100, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 7100 },
        { modelName: 'claude-task22', inputTokens: 3000, outputTokens: 1200, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 4200 }
      ]
    };
  }
`;

const emptyGraphsModule = `
  export async function LoadGraphs(month) {
    return { modelTokenUsages: [], modelCosts: [], dailyTokenTrends: [], modelTokenBreakdowns: [] };
  }
`;

const subscriptionsModule = `
  export async function LoadSubscriptions() {
    const deleted = localStorage.getItem('task22SubscriptionDeleted') === 'true';
    const saved = localStorage.getItem('task22SubscriptionSaved') === 'true';
    const items = deleted ? [] : [
      { subscriptionId: 'openai-chatgpt-plus-2026-04-01', provider: 'openai', planName: 'ChatGPT Plus', renewalDay: 15, startsAt: '2026-04-01T00:00:00Z', endsAt: '', feeUsd: 20, isActive: true },
      ...(saved ? [{ subscriptionId: 'anthropic-claude-pro-2026-04-30', provider: 'anthropic', planName: 'Claude Pro', renewalDay: 20, startsAt: '2026-04-30T00:00:00.000Z', endsAt: '', feeUsd: 20, isActive: true }] : [])
    ];
    return { items, empty: items.length === 0 };
  }
  export async function DeleteSubscription(subscriptionId) {
    localStorage.setItem('task22SubscriptionDeleted', 'true');
    return { success: true };
  }
`;

const emptySubscriptionsModule = `
  export async function LoadSubscriptions() {
    return { items: [], empty: true };
  }
  export async function DeleteSubscription(subscriptionId) {
    return { success: true };
  }
`;

const insightsModule = `
  export async function LoadWasteSummary(month) {
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totalWasteCostUsd: 18,
      totalSpendCostUsd: 108,
      wastePercent: 16.7,
      weeklyWasteCostUsd: 6,
      monthlyWasteCostUsd: 18,
      projectedMonthEndWasteUsd: 24,
      byDetector: [{ category: 'context_avalanche', attributedCostUsd: 12, insightCount: 1 }],
      topCauses: [{ category: 'context_avalanche', attributedCostUsd: 12, insightCount: 1 }],
      dailyTrend: [{ day: '2026-04-30T00:00:00Z', wasteCostUsd: 6 }],
      generatedAt: '2026-04-30T12:00:00Z'
    };
  }
  export async function LoadInsights(month) {
    return {
      items: [{ insightId: 'task22-insight', category: 'context_avalanche', severity: 'high', detectedAt: '2026-04-30T12:00:00Z', period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' }, payload: { sessionIds: ['task22-manual'], usageEntryIds: [], hashes: [{ kind: 'prompt_hash', value: 'task22hash' }], counts: [{ key: 'turns', value: 7 }], metrics: [{ key: 'wasted_tokens', unit: 'tokens', value: 900 }] } }],
      empty: false
    };
  }
`;

const emptyInsightsModule = `
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
  export async function LoadInsights(month) {
    return { items: [], empty: true };
  }
`;

const alertsModule = `
  export async function LoadAlerts(month) {
    return { items: [], empty: true };
  }
`;

const formsModule = `
  const settings = {
    providers: { anthropicEnabled: true, openaiEnabled: true, geminiEnabled: false, openRouterEnabled: false },
    cliBillingDefaults: { claudeCode: 'direct_api', codex: 'direct_api', geminiCli: 'direct_api', openCode: 'direct_api' },
    subscriptionDefaults: {
      openai: { enabled: true, planCode: 'chatgpt-plus', planName: 'ChatGPT Plus', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      claude: { enabled: true, planCode: 'claude-pro', planName: 'Claude Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      gemini: { enabled: false, planCode: 'gemini-pro', planName: 'Gemini Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' }
    },
    budgets: { monthlyBudgetUsd: 120, monthlySubscriptionBudgetUsd: 40, monthlyUsageBudgetUsd: 80, warningThresholdPercent: 80, criticalThresholdPercent: 95 },
    notifications: { desktopEnabled: true, tuiEnabled: false, budgetWarnings: true, forecastWarnings: true, providerSyncFailure: true },
    databasePath: '/tmp/llmbudget.sqlite3'
  };
  export async function ListSubscriptionPresets() {
    return { items: [{ key: 'claude-pro', provider: 'anthropic', planName: 'Claude Pro', renewalDay: 20, feeUsd: 20 }] };
  }
  export async function SaveManualEntry(input) {
    localStorage.setItem('task22ManualSaved', 'true');
    return { result: { success: true }, entry: { entryId: 'task22-manual-entry', ...input, totalCostUsd: 12 } };
  }
  export async function SaveSubscription(input) {
    localStorage.setItem('task22SubscriptionSaved', 'true');
    return { result: { success: true }, subscription: { subscriptionId: 'anthropic-claude-pro-2026-04-30', ...input } };
  }
  export async function SaveBudget(input) {
    localStorage.setItem('task22BudgetLimit', String(input.limitUsd));
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

async function mockModule(page: Page, pattern: string, body: string) {
  await page.route(pattern, async (route) => {
    await route.fulfill({ status: 200, contentType: 'application/javascript', body });
  });
}

async function installTask22Mocks(page: Page) {
  await mockModule(page, '**/wailsjs/go/gui/DashboardBinding*', dashboardModule);
  await mockModule(page, '**/wailsjs/go/gui/GraphsBinding*', graphsModule);
  await mockModule(page, '**/wailsjs/go/gui/InsightsBinding*', insightsModule);
  await mockModule(page, '**/wailsjs/go/gui/AlertsBinding*', alertsModule);
  await mockModule(page, '**/wailsjs/go/gui/SubscriptionLookupBinding*', subscriptionsModule);
  await mockModule(page, '**/wailsjs/go/gui/FormsBinding*', formsModule);
  await page.addInitScript(() => {
    window.localStorage.clear();
    window.localStorage.setItem('llm-budget-tracker-theme', 'dark');
  });
}

test.describe('Task 22 E2E coverage', () => {
  test.beforeEach(async ({ page }) => {
    await installTask22Mocks(page);
  });

  test('dashboard shows summary cards, budget warning, and recent activity', async ({ page }) => {
    await page.goto('/');

    await expect(page.getByRole('main').getByRole('heading', { name: 'Dashboard' })).toBeVisible();
    await expect(page.getByText('$96.00').first()).toBeVisible();
    await expect(page.getByText('Task 22 Monthly Budget')).toBeVisible();
    await expect(page.getByText('task22-baseline')).toBeVisible();
    await expect(page.getByText('Daily Cost Trend')).toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-22-dashboard.png', fullPage: true });
  });

  test('manual usage input saves and refreshes usage history', async ({ page }) => {
    await page.goto('/usage');

    await page.getByLabel('Provider').selectOption('claude');
    await page.getByLabel('Model ID').fill('gpt-task22-manual');
    await page.getByLabel('Input Tokens').fill('900');
    await page.getByLabel('Output Tokens').fill('300');
    await page.getByLabel('Cached Tokens').fill('0');
    await page.getByLabel('Cache Write Tokens').fill('0');
    await page.getByLabel('Project Name').fill('task22-manual-project');
    await page.getByRole('button', { name: 'Save Entry' }).click();

    await expect(page.getByText('Usage entry saved')).toBeVisible();
    await expect(page.getByText('gpt-task22-manual').first()).toBeVisible();
    await expect(page.getByText('$12.00').first()).toBeVisible();
  });

  test('subscription list supports add and delete flows', async ({ page }) => {
    await page.goto('/subscriptions/new');

    await page.getByLabel('Preset').selectOption('claude-pro');
    await page.getByRole('button', { name: 'Save Subscription' }).click();
    await expect(page).toHaveURL(/\/subscriptions$/);
    await expect(page.getByText('Claude Pro')).toBeVisible();

    page.on('dialog', (dialog) => dialog.accept());
    await page.getByRole('button', { name: 'Delete' }).first().click();

    await expect(page.getByText('Subscription deleted')).toBeVisible();
    await expect(page.getByText('No subscriptions found.')).toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-22-subscriptions.png', fullPage: true });
  });

  test('budget setting updates budget panels and warnings', async ({ page }) => {
    await page.goto('/budgets');

    await expect(page.getByText('Warning Threshold Reached')).toBeVisible();
    await page.getByLabel('Monthly Limit (USD)').fill('180');
    await page.getByLabel('Warning Threshold (%)').fill('70');
    await page.getByLabel('Critical Threshold (%)').fill('95');
    await page.getByRole('button', { name: 'Save Budget' }).click();

    await expect(page.getByText('Budget saved')).toBeVisible();
    await expect(page.getByText('of $180.00')).toBeVisible();
  });

  test('insights summary opens and closes the detail modal', async ({ page }) => {
    await page.goto('/insights');

    await expect(page.getByText('Waste Headline')).toBeVisible();
    await expect(page.getByText('Context Avalanche').first()).toBeVisible();
    await page.locator('tbody tr').filter({ hasText: 'Context Avalanche' }).click();

    await expect(page.getByText('Insight Details')).toBeVisible();
    await expect(page.getByText('wasted_tokens')).toBeVisible();
    await page.getByLabel('Close modal').click();
    await expect(page.getByText('Insight Details')).not.toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-22-insights.png', fullPage: true });
  });

  test('graphs tabs and time range controls switch deterministically', async ({ page }) => {
    await page.goto('/graphs');

    await expect(page.getByTestId('tab-Model-Token-Usage')).toHaveClass(/border-primary/);
    await page.getByTestId('tab-Model-Cost').click();
    await expect(page.getByRole('heading', { name: 'Model Cost' })).toBeVisible();
    await page.getByTestId('tab-Daily-Token-Trend').click();
    await expect(page.getByRole('heading', { name: 'Daily Token Trend' })).toBeVisible();
    await page.getByTestId('time-range-selector').selectOption('7 days');
    await expect(page.getByTestId('time-range-selector')).toHaveValue('7 days');
    await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();
  });

  test('settings theme and notification controls persist through settings binding', async ({ page }) => {
    await page.goto('/settings');

    await expect(page.locator('html')).toHaveClass(/dark/);
    await page.locator('label', { has: page.locator('input#theme-toggle') }).click();
    await expect(page.locator('html')).toHaveClass(/light/);

    const notificationToggle = page.locator('input#notification-toggle');
    await notificationToggle.evaluate((element: HTMLInputElement) => element.click());
    await expect(page.getByText('Settings saved')).toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-22-settings.png', fullPage: true });
  });

  test('sidebar navigation visits every primary route', async ({ page }) => {
    await page.goto('/');

    const sidebar = page.getByRole('complementary', { name: 'Sidebar Navigation' });
    await sidebar.getByRole('link', { name: 'Usage' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Usage Tracking' })).toBeVisible();
    await sidebar.getByRole('link', { name: 'Subscriptions' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Subscriptions' })).toBeVisible();
    await sidebar.getByRole('link', { name: 'Budgets' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Budget Management' })).toBeVisible();
    await sidebar.getByRole('link', { name: 'Insights' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Insights', exact: true })).toBeVisible();
    await sidebar.getByRole('link', { name: 'Graphs' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Graphs' })).toBeVisible();
    await sidebar.getByRole('link', { name: 'Settings' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Settings' })).toBeVisible();
    await sidebar.getByRole('link', { name: 'Dashboard' }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  });
});

test.describe('Task 22 empty-state coverage', () => {
  test('dashboard, subscriptions, insights, usage, and graphs render empty states', async ({ page }) => {
    await mockModule(page, '**/wailsjs/go/gui/DashboardBinding*', emptyDashboardModule);
    await mockModule(page, '**/wailsjs/go/gui/GraphsBinding*', emptyGraphsModule);
    await mockModule(page, '**/wailsjs/go/gui/InsightsBinding*', emptyInsightsModule);
    await mockModule(page, '**/wailsjs/go/gui/AlertsBinding*', alertsModule);
    await mockModule(page, '**/wailsjs/go/gui/SubscriptionLookupBinding*', emptySubscriptionsModule);
    await mockModule(page, '**/wailsjs/go/gui/FormsBinding*', formsModule);

    await page.goto('/');
    await expect(page.getByText('No Data Available')).toBeVisible();
    await page.goto('/usage');
    await expect(page.getByText('No usage history found.')).toBeVisible();
    await page.goto('/subscriptions');
    await expect(page.getByText('No subscriptions found.')).toBeVisible();
    await page.goto('/insights');
    await expect(page.getByText('No waste causes found.')).toBeVisible();
    await expect(page.getByText('No insights found for this period.')).toBeVisible();
    await page.goto('/graphs');
    await expect(page.getByTestId('empty-chart')).toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-22-empty-states.png', fullPage: true });
  });
});
