import { test, expect } from '@playwright/test';

test.describe('Subscriptions Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/wailsjs/go/gui/SubscriptionLookupBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          let subscriptions = [
            {
              subscriptionId: 'openai-chatgpt-plus-2024-01-01',
              provider: 'openai',
              planName: 'ChatGPT Plus',
              renewalDay: 15,
              startsAt: '2024-01-01T00:00:00Z',
              feeUsd: 20.00,
              isActive: true
            }
          ];
          export async function LoadSubscriptions() {
            return {
              items: subscriptions,
              empty: subscriptions.length === 0
            };
          }
          export async function DeleteSubscription(id) {
            subscriptions = [];
            return { success: true };
          }
        `
      });
    });

    await page.route('**/wailsjs/go/gui/FormsBinding*', async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/javascript',
        body: `
          export async function ListSubscriptionPresets() {
            return {
              items: [
                {
                  key: 'chatgpt-plus',
                  provider: 'openai',
                  planName: 'ChatGPT Plus',
                  renewalDay: 1,
                  feeUsd: 20.00
                }
              ]
            };
          }
          export async function SaveSubscription(input) {
            return {
              result: { success: true },
              subscription: {
                subscriptionId: 'openai-chatgpt-plus-2024-01-01',
                provider: input.provider,
                planName: input.planName,
                renewalDay: input.renewalDay,
                startsAt: input.startsAt,
                endsAt: input.endsAt,
                feeUsd: input.feeUsd,
                isActive: input.isActive
              }
            };
          }
        `
      });
    });
  });

  test('renders subscriptions list and deletes a subscription', async ({ page }) => {
    await page.goto('/subscriptions');
    
    await expect(page.locator('h1.text-3xl').filter({ hasText: 'Subscriptions' })).toBeVisible();
    await expect(page.getByText('ChatGPT Plus')).toBeVisible();
    
    page.on('dialog', dialog => dialog.accept());
    
    await page.getByRole('button', { name: 'Delete' }).click();
    
    await expect(page.getByText('Subscription deleted')).toBeVisible();
    await expect(page.getByText('ChatGPT Plus')).not.toBeVisible();
    await expect(page.getByText('No subscriptions found.')).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-16-subscription-delete.png' });
  });

  test('adds a new subscription', async ({ page }) => {
    await page.goto('/subscriptions/new');
    
    await expect(page.locator('h1').filter({ hasText: 'Add Subscription' })).toBeVisible();
    
    await page.getByLabel('Preset').selectOption('chatgpt-plus');
    
    await expect(page.getByLabel('Provider')).toHaveValue('openai');
    await expect(page.getByLabel('Plan Name')).toHaveValue('ChatGPT Plus');
    await expect(page.getByLabel('Fee (USD)')).toHaveValue('20');
    await expect(page.getByLabel('Renewal Day (1-31)')).toHaveValue('1');
    
    await page.getByRole('button', { name: 'Save Subscription' }).click();
    
    await expect(page.getByText('Subscription saved')).toBeVisible();
    
    await page.screenshot({ path: '../../../../.sisyphus/evidence/task-16-subscription-add.png' });
  });
});
