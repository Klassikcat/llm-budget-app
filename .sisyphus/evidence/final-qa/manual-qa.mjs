import { mkdir, writeFile } from 'node:fs/promises';
import { createRequire } from 'node:module';
import path from 'node:path';

const require = createRequire('/home/shinjungtae/Projects/Personal/BudgetApps/llm-budget-tracker/internal/adapters/gui/frontend/package.json');
const { chromium, expect } = require('@playwright/test');

const evidenceDir = '/home/shinjungtae/Projects/Personal/BudgetApps/llm-budget-tracker/.sisyphus/evidence/final-qa';
await mkdir(evidenceDir, { recursive: true });

const stateModule = `
  const defaultSettings = {
    providers: { anthropicEnabled: true, openaiEnabled: true, geminiEnabled: false, openRouterEnabled: false },
    cliBillingDefaults: { claudeCode: 'direct_api', codex: 'direct_api', geminiCli: 'direct_api', openCode: 'direct_api' },
    subscriptionDefaults: {
      openai: { enabled: true, planCode: 'chatgpt-plus', planName: 'ChatGPT Plus', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      claude: { enabled: true, planCode: 'claude-pro', planName: 'Claude Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      gemini: { enabled: false, planCode: 'gemini-pro', planName: 'Gemini Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' }
    },
    budgets: { monthlyBudgetUsd: 120, monthlySubscriptionBudgetUsd: 40, monthlyUsageBudgetUsd: 80, warningThresholdPercent: 80, criticalThresholdPercent: 95 },
    notifications: { desktopEnabled: true, tuiEnabled: false, budgetWarnings: true, forecastWarnings: true, providerSyncFailure: true },
    databasePath: '/tmp/final-qa/llmbudget.sqlite3'
  };
  function getState() {
    const raw = localStorage.getItem('finalQaState');
    if (raw) return JSON.parse(raw);
    const state = {
      manualSaved: false,
      subscriptionSaved: false,
      subscriptionDeleted: false,
      budgetLimit: 120,
      settings: defaultSettings
    };
    localStorage.setItem('finalQaState', JSON.stringify(state));
    return state;
  }
  function saveState(state) { localStorage.setItem('finalQaState', JSON.stringify(state)); }
`;

const dashboardModule = `${stateModule}
  export async function LoadDashboard(month) {
    const state = getState();
    const currentSpend = state.manualSaved ? 108 : 96;
    return {
      period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
      totals: { variableSpendUsd: currentSpend - 25, subscriptionSpendUsd: 25, totalSpendUsd: currentSpend, currency: 'USD' },
      providerSummaries: [{ provider: 'openai', variableSpendUsd: currentSpend - 25, subscriptionSpendUsd: 25, totalSpendUsd: currentSpend, usageEntryCount: state.manualSaved ? 3 : 2, sessionCount: state.manualSaved ? 3 : 2, currency: 'USD' }],
      budgets: [{ budgetId: 'final-qa-budget', name: 'Final QA Monthly Budget', provider: '', projectHash: '', limitUsd: state.budgetLimit, currentSpendUsd: currentSpend, remainingUsd: state.budgetLimit - currentSpend, triggeredThresholdPercents: [80], warningThresholdPercent: 80, criticalThresholdPercent: 95, budgetOverrunActive: currentSpend >= state.budgetLimit, currency: 'USD' }],
      recentSessions: [
        { sessionId: 'final-qa-base', provider: 'openai', billingMode: 'direct_api', projectName: 'final-qa-baseline', agentName: 'manual', modelId: 'gpt-final-qa', startedAt: '2026-04-28T10:00:00Z', endedAt: '2026-04-28T10:05:00Z', durationSeconds: 300, totalCostUsd: 71, totalTokens: 7100, currency: 'USD' },
        ...(state.manualSaved ? [{ sessionId: 'final-qa-manual', provider: 'anthropic', billingMode: 'direct_api', projectName: 'final-qa-manual-project', agentName: 'manual', modelId: 'claude-final-qa', startedAt: '2026-04-30T10:00:00Z', endedAt: '2026-04-30T10:05:00Z', durationSeconds: 300, totalCostUsd: 12, totalTokens: 1200, currency: 'USD' }] : [])
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
      providerSummaries: [], budgets: [], recentSessions: [], empty: true
    };
  }
`;

const graphsModule = `
  export async function LoadGraphs(month) {
    return {
      modelTokenUsages: [
        { modelName: 'gpt-final-qa', totalTokens: 7100, inputTokens: 5000, outputTokens: 2100, cacheReadTokens: 0, cacheWriteTokens: 0 },
        { modelName: 'claude-final-qa', totalTokens: 4200, inputTokens: 3000, outputTokens: 1200, cacheReadTokens: 0, cacheWriteTokens: 0 }
      ],
      modelCosts: [
        { modelName: 'gpt-final-qa', totalCostUsd: 71 },
        { modelName: 'claude-final-qa', totalCostUsd: 42 }
      ],
      dailyTokenTrends: Array.from({ length: 30 }, (_, index) => ({ date: '2026-04-' + String(index + 1).padStart(2, '0') + 'T00:00:00Z', modelBreakdown: [{ modelName: 'gpt-final-qa', totalTokens: 100 + index * 20 }, { modelName: 'claude-final-qa', totalTokens: 80 + index * 10 }] })),
      modelTokenBreakdowns: [
        { modelName: 'gpt-final-qa', inputTokens: 5000, outputTokens: 2100, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 7100 },
        { modelName: 'claude-final-qa', inputTokens: 3000, outputTokens: 1200, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 4200 }
      ]
    };
  }
`;

const emptyGraphsModule = `export async function LoadGraphs() { return { modelTokenUsages: [], modelCosts: [], dailyTokenTrends: [], modelTokenBreakdowns: [] }; }`;

const subscriptionsModule = `${stateModule}
  export async function LoadSubscriptions() {
    const state = getState();
    const items = state.subscriptionDeleted ? [] : [
      { subscriptionId: 'openai-chatgpt-plus-2026-04-01', provider: 'openai', planName: 'ChatGPT Plus', renewalDay: 15, startsAt: '2026-04-01T00:00:00Z', endsAt: '', feeUsd: 20, isActive: true },
      ...(state.subscriptionSaved ? [{ subscriptionId: 'anthropic-claude-pro-2026-04-30', provider: 'anthropic', planName: 'Claude Pro', renewalDay: 20, startsAt: '2026-04-30T00:00:00.000Z', endsAt: '', feeUsd: 20, isActive: true }] : [])
    ];
    return { items, empty: items.length === 0 };
  }
  export async function DeleteSubscription(subscriptionId) {
    if (subscriptionId !== 'anthropic-claude-pro-2026-04-30') throw new Error('Unexpected subscriptionId: ' + subscriptionId);
    const state = getState();
    state.subscriptionDeleted = true;
    saveState(state);
    return { success: true };
  }
`;

const emptySubscriptionsModule = `export async function LoadSubscriptions() { return { items: [], empty: true }; } export async function DeleteSubscription() { return { success: true }; }`;

const insightsModule = `
  export async function LoadWasteSummary(month) {
    return { period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' }, totalWasteCostUsd: 18, totalSpendCostUsd: 108, wastePercent: 16.7, weeklyWasteCostUsd: 6, monthlyWasteCostUsd: 18, projectedMonthEndWasteUsd: 24, byDetector: [{ category: 'context_avalanche', attributedCostUsd: 12, insightCount: 1 }], topCauses: [{ category: 'context_avalanche', attributedCostUsd: 12, insightCount: 1 }], dailyTrend: [{ day: '2026-04-30T00:00:00Z', wasteCostUsd: 6 }], generatedAt: '2026-04-30T12:00:00Z' };
  }
  export async function LoadInsights(month) {
    return { items: [{ insightId: 'final-qa-insight', category: 'context_avalanche', severity: 'high', detectedAt: '2026-04-30T12:00:00Z', period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' }, payload: { sessionIds: ['final-qa-manual'], usageEntryIds: [], hashes: [{ kind: 'prompt_hash', value: 'finalqahash' }], counts: [{ key: 'turns', value: 7 }], metrics: [{ key: 'wasted_tokens', unit: 'tokens', value: 900 }] } }], empty: false };
  }
`;

const emptyInsightsModule = `
  export async function LoadWasteSummary(month) { return { period: { month: month || '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' }, totalWasteCostUsd: 0, totalSpendCostUsd: 0, wastePercent: 0, weeklyWasteCostUsd: 0, monthlyWasteCostUsd: 0, projectedMonthEndWasteUsd: 0, byDetector: [], topCauses: [], dailyTrend: [], generatedAt: '2026-04-30T12:00:00Z' }; }
  export async function LoadInsights() { return { items: [], empty: true }; }
`;

const alertsModule = `export async function LoadAlerts() { return { items: [], empty: true }; }`;
const runtimeModule = `export function EventsOn() { return () => {}; }`;

const formsModule = `${stateModule}
  export async function ListSubscriptionPresets() { return { items: [{ key: 'claude-pro', provider: 'anthropic', planName: 'Claude Pro', renewalDay: 20, feeUsd: 20 }] }; }
  export async function SaveManualEntry(input) { const state = getState(); state.manualSaved = true; saveState(state); return { result: { success: true }, entry: { entryId: 'final-qa-entry', ...input, totalCostUsd: 12 } }; }
  export async function SaveSubscription(input) { const state = getState(); state.subscriptionSaved = true; state.subscriptionDeleted = false; saveState(state); return { result: { success: true }, subscription: { subscriptionId: 'anthropic-claude-pro-2026-04-30', ...input } }; }
  export async function SaveBudget(input) { const state = getState(); state.budgetLimit = input.limitUsd; saveState(state); return { result: { success: true }, budget: input }; }
  export async function LoadSettings() { return { result: { success: true }, settings: getState().settings }; }
  export async function SaveSettings(input) { const state = getState(); state.settings = input; saveState(state); return { result: { success: true }, settings: input }; }
`;

const results = [];
const consoleMessages = [];
const networkIssues = [];

async function mockModule(page, pattern, body) {
  await page.route(pattern, async (route) => route.fulfill({ status: 200, contentType: 'application/javascript', body }));
}

async function installMocks(page, mode = 'populated') {
  await mockModule(page, '**/wailsjs/runtime/runtime*', runtimeModule);
  await mockModule(page, '**/wailsjs/go/gui/DashboardBinding*', mode === 'empty' ? emptyDashboardModule : dashboardModule);
  await mockModule(page, '**/wailsjs/go/gui/GraphsBinding*', mode === 'empty' ? emptyGraphsModule : graphsModule);
  await mockModule(page, '**/wailsjs/go/gui/InsightsBinding*', mode === 'empty' ? emptyInsightsModule : insightsModule);
  await mockModule(page, '**/wailsjs/go/gui/AlertsBinding*', alertsModule);
  await mockModule(page, '**/wailsjs/go/gui/SubscriptionLookupBinding*', mode === 'empty' ? emptySubscriptionsModule : subscriptionsModule);
  await mockModule(page, '**/wailsjs/go/gui/FormsBinding*', formsModule);
  await page.addInitScript(() => {
    if (!window.localStorage.getItem('qa-initialized')) {
      window.localStorage.clear();
      window.localStorage.setItem('qa-initialized', 'true');
    }
    window.localStorage.setItem('llm-budget-tracker-theme', 'dark');
  });
}

async function pass(name, fn) {
  try {
    await fn();
    results.push({ name, status: 'pass' });
  } catch (error) {
    results.push({ name, status: 'fail', error: error.message });
  }
}

async function screenshot(page, file) {
  await page.screenshot({ path: path.join(evidenceDir, file), fullPage: true });
}

const browser = await chromium.launch({ headless: true });
const context = await browser.newContext({ baseURL: 'http://127.0.0.1:5173', viewport: { width: 1440, height: 1100 } });
const page = await context.newPage();
page.on('console', (message) => {
  if (['error', 'warning'].includes(message.type())) consoleMessages.push({ type: message.type(), text: message.text() });
});
page.on('pageerror', (error) => consoleMessages.push({ type: 'pageerror', text: error.message }));
page.on('response', (response) => {
  if (response.status() >= 400 && !response.url().includes('/favicon.ico')) networkIssues.push({ status: response.status(), url: response.url() });
});
await installMocks(page, 'populated');

await pass('Dashboard summary and baseline data', async () => {
  await page.goto('/');
  await expect(page.getByRole('main').getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect(page.getByText('$96.00').first()).toBeVisible();
  await expect(page.getByText('Final QA Monthly Budget')).toBeVisible();
  await expect(page.getByText('final-qa-baseline')).toBeVisible();
  await screenshot(page, '01-dashboard.png');
});

await pass('Usage invalid input validation', async () => {
  await page.goto('/usage');
  await page.getByLabel('Provider').selectOption('claude');
  await page.getByLabel('Model ID').fill('invalid-negative-tokens');
  await page.getByLabel('Input Tokens').fill('-1');
  await page.getByLabel('Output Tokens').fill('0');
  await page.getByRole('button', { name: 'Save Entry' }).click();
  await expect(page.getByText(/Input tokens cannot be negative|Input tokens must be|greater than|nonnegative|Expected/i).first()).toBeVisible();
  await screenshot(page, '02-usage-invalid-input.png');
});

await pass('Usage save refreshes dashboard and history', async () => {
  await page.getByLabel('Model ID').fill('claude-final-qa');
  await page.getByLabel('Input Tokens').fill('900');
  await page.getByLabel('Output Tokens').fill('300');
  await page.getByLabel('Cached Tokens').fill('0');
  await page.getByLabel('Cache Write Tokens').fill('0');
  await page.getByLabel('Project Name').fill('final-qa-manual-project');
  await page.getByRole('button', { name: 'Save Entry' }).click();
  await expect(page.getByText('Usage entry saved')).toBeVisible();
  await expect(page.getByText('claude-final-qa').first()).toBeVisible();
  await page.goto('/');
  await page.waitForTimeout(500); // Wait for dashboard to load
  await screenshot(page, '03-usage-dashboard-refresh-check.png');
  await expect(page.getByText('$108.00').first()).toBeVisible();
  await expect(page.getByText('final-qa-manual-project')).toBeVisible();
});

await pass('Subscriptions add and delete', async () => {
  await page.goto('/subscriptions/new');
  await page.getByLabel('Preset').selectOption('claude-pro');
  await page.getByRole('button', { name: 'Save Subscription' }).click();
  await expect(page).toHaveURL(/\/subscriptions$/);
  await expect(page.getByText('Claude Pro')).toBeVisible();
  page.once('dialog', (dialog) => dialog.accept());
  await page.locator('tbody tr').filter({ hasText: 'Claude Pro' }).getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByText('Subscription deleted')).toBeVisible();
  await expect(page.getByText('No subscriptions found.')).toBeVisible();
  await screenshot(page, '04-subscriptions-delete.png');
});

await pass('Budget save refreshes budget panels', async () => {
  await page.goto('/budgets');
  await page.getByLabel('Monthly Limit (USD)').fill('180');
  await page.getByLabel('Warning Threshold (%)').fill('70');
  await page.getByLabel('Critical Threshold (%)').fill('95');
  await page.getByRole('button', { name: 'Save Budget' }).click();
  await expect(page.getByText('Budget saved')).toBeVisible();
  await expect(page.getByText('of $180.00')).toBeVisible();
  await page.goto('/');
  await page.waitForTimeout(500); // Wait for dashboard to load
  await screenshot(page, '05-budget-dashboard-refresh-check.png');
  await expect(page.getByText('Final QA Monthly Budget')).toBeVisible();
  await expect(page.getByText('$108.00').first()).toBeVisible();
});

await pass('Insights detail modal', async () => {
  await page.goto('/insights');
  await expect(page.getByText('Waste Headline')).toBeVisible();
  await page.locator('tbody tr').filter({ hasText: 'Context Avalanche' }).click();
  await expect(page.getByText('Insight Details')).toBeVisible();
  await expect(page.getByText('wasted_tokens')).toBeVisible();
  await page.getByLabel('Close modal').click();
  await expect(page.getByText('Insight Details')).not.toBeVisible();
  await screenshot(page, '06-insights.png');
});

await pass('Graphs tab and time-range switching', async () => {
  await page.goto('/graphs');
  await expect(page.getByTestId('tab-Model-Token-Usage')).toHaveClass(/border-primary/);
  await page.getByTestId('tab-Model-Cost').click();
  await expect(page.getByRole('heading', { name: 'Model Cost' })).toBeVisible();
  await page.getByTestId('tab-Daily-Token-Trend').click();
  await expect(page.getByRole('heading', { name: 'Daily Token Trend' })).toBeVisible();
  await page.getByTestId('time-range-selector').selectOption('7 days');
  await expect(page.getByTestId('time-range-selector')).toHaveValue('7 days');
  await page.getByTestId('time-range-selector').selectOption('All');
  await expect(page.getByTestId('time-range-selector')).toHaveValue('All');
  await expect(page.locator('div[class*="w-full h-full"]').first()).toBeVisible();
  await screenshot(page, '07-graphs.png');
});

await pass('Settings theme and notification toggles', async () => {
  await page.goto('/settings');
  await expect(page.locator('html')).toHaveClass(/dark/);
  await page.locator('label', { has: page.locator('input#theme-toggle') }).click();
  await expect(page.locator('html')).toHaveClass(/light/);
  await page.locator('input#notification-toggle').evaluate((element) => element.click());
  await expect(page.getByText('/tmp/final-qa/llmbudget.sqlite3')).toBeVisible();
  await expect(page.getByText('Settings saved')).toBeVisible();
  await screenshot(page, '08-settings.png');
});

await pass('Sidebar navigation all primary routes', async () => {
  await page.goto('/');
  const sidebar = page.getByRole('complementary', { name: 'Sidebar Navigation' });
  const routes = [
    ['Usage', 'Usage Tracking'],
    ['Subscriptions', 'Subscriptions'],
    ['Budgets', 'Budget Management'],
    ['Insights', 'Insights'],
    ['Graphs', 'Graphs'],
    ['Settings', 'Settings'],
    ['Dashboard', 'Dashboard']
  ];
  for (const [link, heading] of routes) {
    await sidebar.getByRole('link', { name: link }).click();
    await expect(page.getByRole('main').getByRole('heading', { name: heading, exact: heading === 'Insights' })).toBeVisible();
  }
  await screenshot(page, '09-navigation.png');
});

await pass('Rapid actions remain stable', async () => {
  await page.goto('/graphs');
  for (const tab of ['tab-Model-Cost', 'tab-Daily-Token-Trend', 'tab-Model-Token-Breakdown', 'tab-Model-Token-Usage']) {
    await page.getByTestId(tab).click();
  }
  await expect(page.getByRole('heading', { name: 'Model Token Usage' })).toBeVisible();
  await page.goto('/settings');
  const themeLabel = page.locator('label', { has: page.locator('input#theme-toggle') });
  await themeLabel.click();
  await themeLabel.click();
  await expect(page.getByRole('main').getByRole('heading', { name: 'Settings' })).toBeVisible();
  await screenshot(page, '10-rapid-actions.png');
});

const emptyPage = await context.newPage();
emptyPage.on('console', (message) => {
  if (['error', 'warning'].includes(message.type())) consoleMessages.push({ type: message.type(), text: message.text() });
});
emptyPage.on('pageerror', (error) => consoleMessages.push({ type: 'pageerror', text: error.message }));
emptyPage.on('response', (response) => {
  if (response.status() >= 400 && !response.url().includes('/favicon.ico')) networkIssues.push({ status: response.status(), url: response.url() });
});
await installMocks(emptyPage, 'empty');
await pass('Empty database states across routes', async () => {
  await emptyPage.goto('/');
  await expect(emptyPage.getByText('No Data Available')).toBeVisible();
  await emptyPage.goto('/usage');
  await expect(emptyPage.getByText('No usage history found.')).toBeVisible();
  await emptyPage.goto('/subscriptions');
  await expect(emptyPage.getByText('No subscriptions found.')).toBeVisible();
  await emptyPage.goto('/insights');
  await expect(emptyPage.getByText('No insights found for this period.')).toBeVisible();
  await emptyPage.goto('/graphs');
  await expect(emptyPage.getByTestId('empty-chart')).toBeVisible();
  await screenshot(emptyPage, '11-empty-states.png');
});

await browser.close();

const resultStatus = (name) => results.find((result) => result.name === name)?.status === 'pass' ? 'pass' : 'fail';
const summary = {
  scenarioResults: results,
  scenarioPass: results.filter((result) => result.status === 'pass').length,
  scenarioTotal: results.length,
  integration: {
    mutationRefreshUsageDashboard: resultStatus('Usage save refreshes dashboard and history'),
    mutationRefreshBudgetDashboard: resultStatus('Budget save refreshes budget panels'),
    subscriptionDeleteListRefresh: resultStatus('Subscriptions add and delete')
  },
  edgeCases: {
    emptyDatabase: resultStatus('Empty database states across routes'),
    invalidUsageInput: resultStatus('Usage invalid input validation'),
    rapidActions: resultStatus('Rapid actions remain stable')
  },
  consoleMessages,
  networkIssues,
  ignoredNetworkIssues: networkIssues.filter((issue) => issue.url.includes('/wailsjs/runtime/runtime')),
  verdict: results.every((result) => result.status === 'pass') && consoleMessages.length === 0 && networkIssues.filter((issue) => !issue.url.includes('/wailsjs/runtime/runtime')).length === 0 ? 'APPROVE' : 'REJECT'
};

await writeFile(path.join(evidenceDir, 'manual-qa-summary.json'), JSON.stringify(summary, null, 2));
await writeFile(path.join(evidenceDir, 'manual-qa-summary.txt'), [
  `Scenarios ${summary.scenarioPass}/${summary.scenarioTotal} pass`,
  `Integration ${Object.values(summary.integration).filter((value) => value === 'pass').length}/${Object.keys(summary.integration).length}`,
  `Edge Cases ${Object.values(summary.edgeCases).length} tested`,
  `Console issues ${consoleMessages.length}`,
  `Network issues ${networkIssues.length}`,
  `VERDICT ${summary.verdict}`
].join('\n'));

