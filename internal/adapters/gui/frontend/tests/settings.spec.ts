import { test, expect } from '@playwright/test';

const settingsModule = `
  const settings = {
    providers: { anthropicEnabled: true, openaiEnabled: true, geminiEnabled: false, openRouterEnabled: false },
    cliBillingDefaults: { claudeCode: 'direct_api', codex: 'direct_api', geminiCli: 'direct_api', openCode: 'direct_api' },
    subscriptionDefaults: {
      openai: { enabled: true, planCode: 'chatgpt-plus', planName: 'ChatGPT Plus', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      claude: { enabled: false, planCode: 'claude-pro', planName: 'Claude Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' },
      gemini: { enabled: false, planCode: 'gemini-pro', planName: 'Gemini Pro', feeUsd: 20, renewalDay: 1, sourceUrl: '' }
    },
    budgets: { monthlyBudgetUsd: 100, monthlySubscriptionBudgetUsd: 40, monthlyUsageBudgetUsd: 60, warningThresholdPercent: 80, criticalThresholdPercent: 95 },
    notifications: { desktopEnabled: true, tuiEnabled: false, budgetWarnings: true, forecastWarnings: true, providerSyncFailure: true },
    databasePath: '/tmp/llmbudget.sqlite3'
  };
  export async function LoadSettings() {
    return { result: { success: true }, settings };
  }
  export async function SaveSettings(input) {
    Object.assign(settings, input);
    return { result: { success: true }, settings };
  }
`;

test.describe('Settings Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/wailsjs/go/gui/FormsBinding*', async (route) => {
      await route.fulfill({ status: 200, contentType: 'application/javascript', body: settingsModule });
    });
    
    await page.addInitScript(() => {
      window.localStorage.clear();
      window.localStorage.setItem('llm-budget-tracker-theme', 'dark');
    });
    
    await page.goto('/settings');
  });

  test('renders settings sections and takes evidence screenshot', async ({ page }) => {
    await expect(page.getByRole('main').getByRole('heading', { name: 'Settings' })).toBeVisible();
    await expect(page.locator('h3', { hasText: 'Appearance' })).toBeVisible();
    await expect(page.getByRole('main').locator('h3', { hasText: 'Notifications' }).nth(1)).toBeVisible();
    await expect(page.locator('h3', { hasText: 'Data Management' })).toBeVisible();
    await expect(page.locator('h3', { hasText: 'About' })).toBeVisible();

    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-20-settings.png', fullPage: true });
  });

  test('toggles theme and takes evidence screenshot', async ({ page }) => {
    await expect(page.locator('html')).toHaveClass(/dark/);
    
    const themeToggle = page.locator('label', { has: page.locator('input#theme-toggle') });
    await themeToggle.click();
    
    await expect(page.locator('html')).toHaveClass(/light/);
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-20-theme-toggle-settings.png', fullPage: true });
  });
});
