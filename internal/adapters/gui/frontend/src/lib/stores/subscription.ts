import { writable } from 'svelte/store';

import { deleteSubscription, loadSubscriptions, saveSubscription } from '$lib/bindings';
import type {
  MutationResponse,
  SubscriptionInput,
  SubscriptionListResponse,
  SubscriptionMutationResponse,
  SubscriptionState
} from '$lib/types/forms';

export interface SubscriptionStoreData {
  response: SubscriptionListResponse | null;
  items: SubscriptionState[];
}

export interface SubscriptionStoreState {
  data: SubscriptionStoreData;
  loading: boolean;
  error: string | null;
}

const initialData: SubscriptionStoreData = {
  response: null,
  items: []
};

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Failed to load subscriptions';
}

function createSubscriptionStore() {
  const store = writable<SubscriptionStoreState>({
    data: initialData,
    loading: false,
    error: null
  });

  async function load(): Promise<SubscriptionStoreData> {
    store.update((state) => ({ ...state, loading: true, error: null }));

    try {
      const response = await loadSubscriptions();
      const data: SubscriptionStoreData = {
        response,
        items: response.items
      };
      store.set({ data, loading: false, error: null });
      return data;
    } catch (error) {
      store.update((state) => ({ ...state, loading: false, error: getErrorMessage(error) }));
      throw error;
    }
  }

  async function save(input: SubscriptionInput): Promise<SubscriptionMutationResponse> {
    const response = await saveSubscription(input);
    await load();
    return response;
  }

  async function remove(subscriptionId: string): Promise<MutationResponse> {
    const response = await deleteSubscription(subscriptionId);
    await load();
    return response;
  }

  return {
    subscribe: store.subscribe,
    load,
    refresh: load,
    save,
    remove
  };
}

export const subscription = createSubscriptionStore();
export const loadSubscription = subscription.load;
export const refreshSubscription = subscription.refresh;
export const saveSubscriptionAndRefresh = subscription.save;
export const deleteSubscriptionAndRefresh = subscription.remove;
