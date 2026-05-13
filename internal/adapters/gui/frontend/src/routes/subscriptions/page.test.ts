import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import SubscriptionsPage from './+page.svelte';
import { deleteSubscriptionAndRefresh } from '$lib/stores/subscription';
import { notificationStore } from '$lib/stores/notification';

vi.mock('$lib/scaffold-readiness', () => ({
  appTitle: 'Test App'
}));

vi.mock('$lib/stores/subscription', () => {
  const { writable } = require('svelte/store');
  const store = writable({
    data: {
      items: [
        {
          subscriptionId: 'openai-chatgpt-plus-2024-01-01',
          provider: 'openai',
          planName: 'ChatGPT Plus',
          renewalDay: 15,
          startsAt: '2024-01-01T00:00:00Z',
          feeUsd: 20.00,
          isActive: true
        }
      ]
    },
    loading: false,
    error: null
  });

  return {
    subscription: store,
    loadSubscription: vi.fn(),
    deleteSubscriptionAndRefresh: vi.fn()
  };
});

vi.mock('$lib/stores/notification', () => {
  const { writable } = require('svelte/store');
  const store = writable({
    items: [],
    unreadCount: 0,
    lastDispatchedAt: null,
    permission: 'default',
    sentAlertKeys: new Set()
  });

  return {
    notificationStore: {
      ...store,
      addNotification: vi.fn()
    }
  };
});

describe('Subscriptions Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(window, 'confirm').mockImplementation(() => true);
  });

  it('renders the subscriptions list', () => {
    render(SubscriptionsPage);
    
    expect(screen.getByText('Subscriptions')).toBeInTheDocument();
    expect(screen.getByText('Add Subscription')).toBeInTheDocument();
    
    expect(screen.getByText('openai')).toBeInTheDocument();
    expect(screen.getByText('ChatGPT Plus')).toBeInTheDocument();
    expect(screen.getByText('$20.00')).toBeInTheDocument();
    expect(screen.getByText('15')).toBeInTheDocument();
    expect(screen.getByText('Active')).toBeInTheDocument();
  });

  it('calls deleteSubscriptionAndRefresh when delete button is clicked and confirmed', async () => {
    vi.mocked(deleteSubscriptionAndRefresh).mockResolvedValue({ success: true });
    
    render(SubscriptionsPage);
    
    const deleteButton = screen.getByRole('button', { name: /Delete/i });
    await fireEvent.click(deleteButton);
    
    expect(window.confirm).toHaveBeenCalledWith('Are you sure you want to delete this subscription?');
    
    await waitFor(() => {
      expect(deleteSubscriptionAndRefresh).toHaveBeenCalledWith('openai-chatgpt-plus-2024-01-01');
      expect(notificationStore.addNotification).toHaveBeenCalledWith(expect.objectContaining({
        title: 'Success',
        body: 'Subscription deleted'
      }));
    });
  });
});
