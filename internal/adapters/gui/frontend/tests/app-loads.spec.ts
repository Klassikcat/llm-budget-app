import { expect, test } from '@playwright/test';

test('loads the dashboard shell through the Vite dev server', async ({ page }) => {
  await page.goto('/');

  await expect(page).toHaveTitle('Dashboard - LLM Budget Tracker');
  await expect(page.getByRole('navigation').locator('a[aria-current="page"]')).toContainText('Dashboard');
  await expect(page.getByRole('main').getByRole('heading', { name: 'Dashboard', level: 1 })).toBeVisible();
  await expect(page.getByRole('complementary', { name: 'Sidebar Navigation' })).toBeVisible();
  await expect(page.getByText('Total Spend')).toBeVisible();
});
