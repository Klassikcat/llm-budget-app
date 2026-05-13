import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import SettingsPage from './+page.svelte';
import { theme } from '$lib/stores/theme';
import { get } from 'svelte/store';
import { loadSettingsState, saveSettingsAndRefresh } from '$lib/stores/settings';
import { notificationStore } from '$lib/stores/notification';
import type { SettingsFormState } from '$lib/bindings';

const mockInitialSettings = vi.hoisted<SettingsFormState>(() => ({
  providers: {
    anthropicEnabled: true,
    openaiEnabled: true,
    geminiEnabled: false,
    openRouterEnabled: false
  },
  cliBillingDefaults: {
    claudeCode: 'direct_api',
    codex: 'direct_api',
    geminiCli: 'direct_api',
    openCode: 'direct_api'
  },
  subscriptionDefaults: {
    openai: {
      enabled: true,
      planCode: 'chatgpt-plus',
      planName: 'ChatGPT Plus',
      feeUsd: 20,
      renewalDay: 1,
      sourceUrl: ''
    },
    claude: {
      enabled: false,
      planCode: 'claude-pro',
      planName: 'Claude Pro',
      feeUsd: 20,
      renewalDay: 1,
      sourceUrl: ''
    },
    gemini: {
      enabled: false,
      planCode: 'gemini-pro',
      planName: 'Gemini Pro',
      feeUsd: 20,
      renewalDay: 1,
      sourceUrl: ''
    }
  },
  budgets: {
    monthlyBudgetUsd: 100,
    monthlySubscriptionBudgetUsd: 40,
    monthlyUsageBudgetUsd: 60,
    warningThresholdPercent: 80,
    criticalThresholdPercent: 95
  },
  notifications: {
    desktopEnabled: true,
    tuiEnabled: false,
    budgetWarnings: true,
    forecastWarnings: true,
    providerSyncFailure: true
  },
  databasePath: '/tmp/llmbudget.sqlite3'
}));

Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: vi.fn().mockImplementation(query => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(),
    removeListener: vi.fn(),
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  })),
});

vi.mock('$lib/stores/settings', () => {
  const { writable } = require('svelte/store');
  const store = writable({
    data: mockInitialSettings,
    loading: false,
    saving: false,
    error: null
  });

  return {
    settings: store,
    loadSettingsState: vi.fn().mockResolvedValue(mockInitialSettings),
    saveSettingsAndRefresh: vi.fn().mockResolvedValue(mockInitialSettings)
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

describe('Settings Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    theme.setTheme('dark');
    vi.mocked(loadSettingsState).mockResolvedValue(mockInitialSettings);
    vi.mocked(saveSettingsAndRefresh).mockResolvedValue(mockInitialSettings);
  });

  it('renders all settings sections', async () => {
    render(SettingsPage);
    
    expect(screen.getByText('Settings')).toBeInTheDocument();
    expect(screen.getByText('Appearance')).toBeInTheDocument();
    expect(screen.getAllByText('Notifications').length).toBeGreaterThan(0);
    expect(screen.getByText('Data Management')).toBeInTheDocument();
    expect(screen.getByText('About')).toBeInTheDocument();

    await waitFor(() => {
      expect(loadSettingsState).toHaveBeenCalled();
    });
  });

  it('selects system, dark, and light theme modes', async () => {
    render(SettingsPage);
    
    expect(get(theme)).toBe('dark');
    
    const themeSelect = screen.getByTestId('theme-select') as HTMLSelectElement;
    expect(themeSelect).toBeInTheDocument();
    
    await fireEvent.change(themeSelect, { target: { value: 'system' } });
    expect(get(theme)).toBe('system');

    await fireEvent.change(themeSelect, { target: { value: 'light' } });
    expect(get(theme)).toBe('light');
  });

  it('saves notification preference through the settings store', async () => {
    render(SettingsPage);
    
    const notifToggle = document.getElementById('notification-toggle') as HTMLInputElement;
    expect(notifToggle).toBeInTheDocument();
    expect(notifToggle.checked).toBe(true);
    
    await fireEvent.click(notifToggle);
    
    await waitFor(() => {
      expect(saveSettingsAndRefresh).toHaveBeenCalledWith(expect.objectContaining({
        notifications: expect.objectContaining({
          desktopEnabled: false,
          budgetWarnings: false
        })
      }));
      expect(notificationStore.addNotification).toHaveBeenCalledWith(expect.objectContaining({
        title: 'Success',
        body: 'Settings saved'
      }));
    });
  });

  it('displays read-only DB path and last refresh time', async () => {
    render(SettingsPage);
    
    await waitFor(() => {
      expect(screen.getByText('/tmp/llmbudget.sqlite3')).toBeInTheDocument();
    });
    expect(screen.getByTestId('last-refresh')).toBeInTheDocument();
  });

  it('shows binding load failures through notifications', async () => {
    vi.mocked(loadSettingsState).mockRejectedValue(new Error('settings unavailable'));

    render(SettingsPage);

    await waitFor(() => {
      expect(notificationStore.addNotification).toHaveBeenCalledWith(expect.objectContaining({
        title: 'Error',
        body: 'settings unavailable'
      }));
    });
  });

  it('displays app info', () => {
    render(SettingsPage);
    
    expect(screen.getByText('LLM Budget Tracker')).toBeInTheDocument();
    expect(screen.getByText(/v1\.0\.0/)).toBeInTheDocument();
  });
});
