import { get } from 'svelte/store';
import { afterEach, describe, expect, it, vi } from 'vitest';

import type { BindingName, WailsBindingClient } from '$lib/bindings';
import type {
  MutationResponse,
  SubscriptionInput,
  SubscriptionListResponse,
  SubscriptionMutationResponse
} from '$lib/types/forms';

function createSubscriptions(planName: string): SubscriptionListResponse {
  return {
    items: [
      {
        subscriptionId: `anthropic-${planName.toLowerCase().replace(/\s+/g, '-')}-2026-04-01`,
        provider: 'anthropic',
        planName,
        renewalDay: 3,
        startsAt: '2026-04-01',
        feeUsd: 20,
        isActive: true
      }
    ],
    empty: false
  };
}

async function loadSubscriptionStore(client: WailsBindingClient) {
  vi.resetModules();
  const bindings = await import('$lib/bindings');
  bindings.setBindingClient(client);
  return import('./subscription');
}

describe('subscription store', () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('loads subscriptions into typed readable state', async () => {
    const response = createSubscriptions('Claude Pro');
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const { subscription } = await loadSubscriptionStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        return response as T;
      }
    });

    await expect(subscription.load()).resolves.toEqual({ response, items: response.items });

    expect(get(subscription)).toEqual({
      data: { response, items: response.items },
      loading: false,
      error: null
    });
    expect(calls).toEqual([{ binding: 'SubscriptionLookupBinding', method: 'LoadSubscriptions', args: [] }]);
  });

  it('sets loading while subscriptions are pending', async () => {
    let resolveLoad: (value: SubscriptionListResponse) => void = () => undefined;
    const pendingSubscriptions = new Promise<SubscriptionListResponse>((resolve) => {
      resolveLoad = resolve;
    });
    const { subscription } = await loadSubscriptionStore({
      async invoke<T>() {
        return pendingSubscriptions as Promise<T>;
      }
    });

    const loadPromise = subscription.load();

    expect(get(subscription).loading).toBe(true);
    resolveLoad(createSubscriptions('Claude Pro'));
    await loadPromise;
    expect(get(subscription).loading).toBe(false);
  });

  it('refreshes subscriptions by loading again', async () => {
    const responses = [createSubscriptions('Claude Pro'), createSubscriptions('Claude Max')];
    const { subscription } = await loadSubscriptionStore({
      async invoke<T>() {
        return responses.shift() as T;
      }
    });

    await subscription.load();
    await subscription.refresh();

    expect(get(subscription).data.items[0]?.planName).toBe('Claude Max');
  });

  it('tracks errors without clearing existing subscriptions', async () => {
    const response = createSubscriptions('Claude Pro');
    let shouldReject = false;
    const { subscription } = await loadSubscriptionStore({
      async invoke<T>() {
        if (shouldReject) {
          throw new Error('subscriptions unavailable');
        }
        return response as T;
      }
    });
    await subscription.load();

    shouldReject = true;
    await expect(subscription.refresh()).rejects.toThrow('subscriptions unavailable');

    expect(get(subscription).data.items).toEqual(response.items);
    expect(get(subscription).error).toBe('subscriptions unavailable');
  });

  it('refreshes after successful save and delete mutations', async () => {
    const calls: { binding: BindingName; method: string; args: readonly unknown[] }[] = [];
    const input: SubscriptionInput = {
      presetKey: 'claude-pro',
      provider: 'anthropic',
      planName: 'Claude Pro',
      renewalDay: 3,
      startsAt: '2026-04-01',
      endsAt: '',
      feeUsd: 20,
      isActive: true
    };
    const mutation: SubscriptionMutationResponse = {
      result: { success: true },
      subscription: { subscriptionId: 'anthropic-claude-pro-2026-04-01', ...input }
    };
    const deleted: MutationResponse = { success: true };
    const { subscription } = await loadSubscriptionStore({
      async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
        calls.push({ binding, method, args });
        if (method === 'SaveSubscription') {
          return mutation as T;
        }
        if (method === 'DeleteSubscription') {
          return deleted as T;
        }
        return createSubscriptions('Claude Pro') as T;
      }
    });

    await expect(subscription.save(input)).resolves.toBe(mutation);
    await expect(subscription.remove('sub-1')).resolves.toBe(deleted);

    expect(calls.map((call) => call.method)).toEqual([
      'SaveSubscription',
      'LoadSubscriptions',
      'DeleteSubscription',
      'LoadSubscriptions'
    ]);
  });
});
