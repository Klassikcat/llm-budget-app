import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import UsagePage from './+page.svelte';
import { saveManualEntryAndRefresh } from '$lib/stores/usage';
import { notificationStore } from '$lib/stores/notification';

vi.mock('$lib/scaffold-readiness', () => ({
  appTitle: 'Test App'
}));

vi.mock('$lib/stores/usage', () => {
  const { writable } = require('svelte/store');
  const store = writable({
    data: {
      recentSessions: [
        {
          sessionId: '1',
          provider: 'claude',
          modelId: 'claude-3.5-sonnet',
          startedAt: '2024-01-01T12:00:00Z',
          totalCostUsd: 0.05,
          totalTokens: 1000
        }
      ]
    },
    loading: false,
    error: null
  });

  return {
    usage: store,
    loadUsage: vi.fn(),
    saveManualEntryAndRefresh: vi.fn()
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

describe('Usage Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders the manual entry form and history table', () => {
    render(UsagePage);
    
    expect(screen.getByText('Usage Tracking')).toBeInTheDocument();
    expect(screen.getByText('Manual Entry')).toBeInTheDocument();
    expect(screen.getByText('History')).toBeInTheDocument();
    
    expect(screen.getByLabelText(/Provider/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Model ID/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Occurred At/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Input Tokens/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Output Tokens/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Cached Tokens/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Cache Write Tokens/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/Project Name/i)).toBeInTheDocument();
  });

  it('shows validation errors for missing required fields', async () => {
    render(UsagePage);
    
    const submitButton = screen.getByRole('button', { name: /Save Entry/i });
    await fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText('Provider is required')).toBeInTheDocument();
      expect(screen.getByText('Model ID is required')).toBeInTheDocument();
      expect(screen.getByText('Project Name is required')).toBeInTheDocument();
    });
    
    expect(saveManualEntryAndRefresh).not.toHaveBeenCalled();
  });

  it('shows validation errors for negative token values', async () => {
    render(UsagePage);
    
    const inputTokens = screen.getByLabelText(/Input Tokens/i);
    await fireEvent.input(inputTokens, { target: { value: '-10' } });
    
    const submitButton = screen.getByRole('button', { name: /Save Entry/i });
    await fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(screen.getByText(/Input tokens cannot be negative|Expected number/i)).toBeInTheDocument();
    });
  });

  it('submits the form successfully and shows a notification', async () => {
    vi.mocked(saveManualEntryAndRefresh).mockResolvedValue({
      result: { success: true },
      entry: {
        entryId: '1',
        provider: 'claude',
        modelId: 'claude-3.5-sonnet',
        occurredAt: '2024-01-01T12:00:00Z',
        inputTokens: 1000,
        outputTokens: 500,
        cachedTokens: 0,
        cacheWriteTokens: 0,
        projectName: 'test-project',
        metadata: {},
        totalCostUsd: 0.05
      }
    });
    
    render(UsagePage);
    
    const provider = screen.getByLabelText(/Provider/i);
    await fireEvent.change(provider, { target: { value: 'claude' } });
    
    const modelId = screen.getByLabelText(/Model ID/i);
    await fireEvent.input(modelId, { target: { value: 'claude-3.5-sonnet' } });
    
    const inputTokens = screen.getByLabelText(/Input Tokens/i);
    await fireEvent.input(inputTokens, { target: { value: '1000' } });

    const outputTokens = screen.getByLabelText(/Output Tokens/i);
    await fireEvent.input(outputTokens, { target: { value: '500' } });

    const projectName = screen.getByLabelText(/Project Name/i);
    await fireEvent.input(projectName, { target: { value: 'test-project' } });
    
    const submitButton = screen.getByRole('button', { name: /Save Entry/i });
    await fireEvent.click(submitButton);
    
    await waitFor(() => {
      expect(saveManualEntryAndRefresh).toHaveBeenCalledWith(expect.objectContaining({
        provider: 'claude',
        modelId: 'claude-3.5-sonnet',
        projectName: 'test-project',
        inputTokens: 1000,
        outputTokens: 500,
        cachedTokens: 0,
        cacheWriteTokens: 0
      }));
      
      expect(notificationStore.addNotification).toHaveBeenCalledWith(expect.objectContaining({
        title: 'Success',
        body: 'Usage entry saved'
      }));

      expect(screen.getAllByText('1000').length).toBeGreaterThan(0);
      expect(screen.getByText('500')).toBeInTheDocument();
      expect(screen.getAllByText('$0.05').length).toBeGreaterThan(0);
    });
  });
});
