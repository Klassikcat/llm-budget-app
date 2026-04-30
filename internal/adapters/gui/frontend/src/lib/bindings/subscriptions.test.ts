import { describe, expect, it } from 'vitest';

import type { SubscriptionListResponse } from '$lib/types/forms';

import { setBindingClient, type BindingName, type WailsBindingClient } from './index';
import { BindingMutationError } from './forms';
import { deleteSubscription, loadSubscriptions } from './subscriptions';

function clientReturning(value: SubscriptionListResponse, calls: { binding: BindingName; method: string; args: readonly unknown[] }[]): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      calls.push({ binding, method, args });
      return value as T;
    }
  };
}

describe('subscription lookup bindings', () => {
  it('loads subscriptions through SubscriptionLookupBinding.LoadSubscriptions', async () => {
    const response: SubscriptionListResponse = {
      items: [
        {
          subscriptionId: 'anthropic-claude-pro-2026-04-01',
          provider: 'anthropic',
          planName: 'Claude Pro',
          renewalDay: 3,
          startsAt: '2026-04-01',
          feeUsd: 20,
          isActive: true
        }
      ],
      empty: false
    };
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient(clientReturning(response, calls));

    await expect(loadSubscriptions()).resolves.toEqual(response);
    expect(calls).toEqual([
      { binding: 'SubscriptionLookupBinding', method: 'LoadSubscriptions', args: [] }
    ]);
  });

  it('deletes subscriptions through SubscriptionLookupBinding.DeleteSubscription', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    setBindingClient({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return { success: true } as T;
      }
    });

    await expect(deleteSubscription('sub-1')).resolves.toEqual({ success: true });
    expect(calls).toEqual([
      { binding: 'SubscriptionLookupBinding', method: 'DeleteSubscription', args: ['sub-1'] }
    ]);
  });

  it('throws BindingMutationError when delete returns a failed mutation', async () => {
    setBindingClient({
      async invoke<T>() {
        return {
          success: false,
          error: { code: 'required', field: 'subscription_id', message: 'value is required' }
        } as T;
      }
    });

    await expect(deleteSubscription('')).rejects.toBeInstanceOf(BindingMutationError);
    await expect(deleteSubscription('')).rejects.toThrow('value is required');
  });

  it('rejects when the injected client rejects', async () => {
    setBindingClient({
      async invoke() {
        throw new Error('subscription query failed');
      }
    });

    await expect(loadSubscriptions()).rejects.toThrow('subscription query failed');
    await expect(deleteSubscription('sub-1')).rejects.toThrow('subscription query failed');
  });
});
