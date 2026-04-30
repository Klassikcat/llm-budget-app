import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import NewSubscriptionPage from './+page.svelte';
import { saveSubscriptionAndRefresh } from '$lib/stores/subscription';
import { notificationStore } from '$lib/stores/notification';
import { listSubscriptionPresets } from '$lib/bindings/forms';

vi.mock('$app/navigation', () => ({
  goto: vi.fn()
}));

vi.mock('$lib/scaffold-readiness', () => ({
  appTitle: 'Test App'
}));

vi.mock('$lib/stores/subscription', () => ({
  saveSubscriptionAndRefresh: vi.fn()
}));

vi.mock('$lib/bindings/forms', () => ({
  listSubscriptionPresets: vi.fn()
}));

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

describe('New Subscription Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(listSubscriptionPresets).mockResolvedValue({
      items: [
        {
          key: 'chatgpt-plus',
          provider: 'openai',
          planName: 'ChatGPT Plus',
          renewalDay: 1,
          feeUsd: 20.00
        }
      ]
    });
  });

  it('renders the add subscription form', async () => {
    render(NewSubscriptionPage);
    
    expect(screen.getByText('Add Subscription')).toBeInTheDocument();
    
    await waitFor(() => {
      expect(screen.getByLabelText(/Preset/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Provider/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Plan Name/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Fee \(USD\)/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Renewal Day/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Starts At/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/Ends At/i)).toBeInTheDocument();
    });
  });

  it('auto-fills form when preset is selected', async () => {
    render(NewSubscriptionPage);
    
    await waitFor(() => {
      expect(listSubscriptionPresets).toHaveBeenCalled();
    });
    
    const presetSelect = screen.getByLabelText(/Preset/i);
    await fireEvent.change(presetSelect, { target: { value: 'chatgpt-plus' } });
    
    await waitFor(() => {
      expect(screen.getByLabelText(/Provider/i)).toHaveValue('openai');
      expect(screen.getByLabelText(/Plan Name/i)).toHaveValue('ChatGPT Plus');
      expect(screen.getByLabelText(/Fee \(USD\)/i)).toHaveValue(20);
      expect(screen.getByLabelText(/Renewal Day/i)).toHaveValue(1);
    });
  });

  it('shows validation errors for missing required fields', async () => {
    render(NewSubscriptionPage);
    
    const submitButton = screen.getByRole('button', { name: /Save Subscription/i });
    await fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText('Provider is required')).toBeInTheDocument();
      expect(screen.getByText('Plan Name is required')).toBeInTheDocument();
    });
    
    expect(saveSubscriptionAndRefresh).not.toHaveBeenCalled();
  });

  it('submits the form successfully and redirects', async () => {
    vi.mocked(saveSubscriptionAndRefresh).mockResolvedValue({
      result: { success: true },
      subscription: {
        subscriptionId: 'openai-chatgpt-plus-2024-01-01',
        provider: 'openai',
        planName: 'ChatGPT Plus',
        renewalDay: 1,
        startsAt: '2024-01-01T00:00:00Z',
        feeUsd: 20.00,
        isActive: true
      }
    });
    
    render(NewSubscriptionPage);
    
    const provider = screen.getByLabelText(/Provider/i);
    await fireEvent.input(provider, { target: { value: 'openai' } });
    
    const planName = screen.getByLabelText(/Plan Name/i);
    await fireEvent.input(planName, { target: { value: 'ChatGPT Plus' } });
    
    const feeUsd = screen.getByLabelText(/Fee \(USD\)/i);
    await fireEvent.input(feeUsd, { target: { value: '20' } });

    const renewalDay = screen.getByLabelText(/Renewal Day/i);
    await fireEvent.input(renewalDay, { target: { value: '15' } });
    
    const submitButton = screen.getByRole('button', { name: /Save Subscription/i });
    await fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(saveSubscriptionAndRefresh).toHaveBeenCalledWith(expect.objectContaining({
        provider: 'openai',
        planName: 'ChatGPT Plus',
        feeUsd: 20,
        renewalDay: 15,
        isActive: true
      }));
      
      expect(notificationStore.addNotification).toHaveBeenCalledWith(expect.objectContaining({
        title: 'Success',
        body: 'Subscription saved'
      }));
    });
  });
});
