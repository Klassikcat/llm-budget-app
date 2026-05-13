import type { MutationResponse, SubscriptionListResponse } from '$lib/types/forms';

import { getBindingClient } from './client';
import { BindingMutationError } from './forms';

export async function loadSubscriptions(): Promise<SubscriptionListResponse> {
  return getBindingClient().invoke<SubscriptionListResponse>('SubscriptionLookupBinding', 'LoadSubscriptions');
}

export async function deleteSubscription(subscriptionId: string): Promise<MutationResponse> {
  const response = await getBindingClient().invoke<MutationResponse>('SubscriptionLookupBinding', 'DeleteSubscription', subscriptionId);
  if (!response.success) {
    throw new BindingMutationError(response);
  }
  return response;
}
