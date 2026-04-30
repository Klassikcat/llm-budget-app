<script lang="ts">
  import { onMount } from 'svelte';
  import { appTitle } from '$lib/scaffold-readiness';
  import Panel from '$lib/components/ui/Panel.svelte';
  import SelectInput from '$lib/components/forms/SelectInput.svelte';
  import type { SelectOption } from '$lib/components/forms/SelectInput.svelte';
  import Toggle from '$lib/components/forms/Toggle.svelte';
  import { resolvedTheme, setTheme, theme, themes, type ThemeMode } from '$lib/stores/theme';
  import { settings, loadSettingsState, saveSettingsAndRefresh } from '$lib/stores/settings';
  import { notificationStore } from '$lib/stores/notification';
  import type { SettingsFormState } from '$lib/bindings';

  const themeOptions: SelectOption[] = [
    { value: 'system', label: 'System' },
    { value: 'dark', label: 'Dark' },
    { value: 'light', label: 'Light' }
  ];

  let notificationsEnabled = $state(true);
  let systemPermission = $state('default');
  let lastRefreshTime = $state('Loading settings...');
  let databasePath = $state('Loading database path...');

  function getErrorMessage(error: unknown, fallback: string): string {
    return error instanceof Error ? error.message : fallback;
  }

  function applySettings(data: SettingsFormState): void {
    notificationsEnabled = data.notifications.budgetWarnings;
    databasePath = data.databasePath || 'Database path unavailable';
    lastRefreshTime = new Date().toLocaleString();
  }

  onMount(async () => {
    if (typeof window !== 'undefined' && 'Notification' in window) {
      systemPermission = Notification.permission;
    } else {
      systemPermission = 'unsupported';
    }

    try {
      const data = await loadSettingsState();
      applySettings(data);
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: getErrorMessage(error, 'Failed to load settings'),
        kind: 'error',
        severity: 'critical'
      });
    }
  });

  function handleThemeChange(event: Event) {
    const selectedTheme = (event.currentTarget as HTMLSelectElement).value;
    if (themes.includes(selectedTheme as ThemeMode)) {
      setTheme(selectedTheme as ThemeMode);
    }
  }

  async function handleNotificationToggle() {
    const current = $settings.data;
    if (!current) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: 'Settings are still loading',
        kind: 'error',
        severity: 'critical'
      });
      return;
    }

    const nextSettings: SettingsFormState = {
      ...current,
      notifications: {
        ...current.notifications,
        desktopEnabled: notificationsEnabled,
        budgetWarnings: notificationsEnabled
      }
    };

    try {
      const data = await saveSettingsAndRefresh(nextSettings);
      applySettings(data);
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Success',
        body: 'Settings saved',
        kind: 'info',
        severity: 'info'
      });
    } catch (error) {
      notificationsEnabled = current.notifications.budgetWarnings;
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: getErrorMessage(error, 'Failed to save settings'),
        kind: 'error',
        severity: 'critical'
      });
    }
  }
</script>

<svelte:head>
  <title>{appTitle} - Settings</title>
</svelte:head>

<div class="p-lg max-w-4xl mx-auto space-y-lg relative">
  <div>
    <h1 class="text-3xl font-bold text-text">Settings</h1>
    <p class="mt-sm text-text-muted">Manage your application preferences and configuration.</p>
  </div>

  {#if $settings.error}
    <div class="p-md bg-status-danger/10 border border-status-danger/30 rounded-compact text-status-danger text-sm">
      {$settings.error}
    </div>
  {/if}

  {#if $settings.loading && !$settings.data}
    <div class="p-md bg-card border border-panel-border rounded-compact text-sm text-text-muted" data-testid="settings-loading">
      Loading settings...
    </div>
  {/if}

  <div class="grid grid-cols-2 gap-lg">
    <Panel title="Appearance">
      <div class="flex items-center justify-between gap-md py-sm">
        <div>
          <h4 class="text-sm font-medium text-text">Theme</h4>
          <p class="text-xs text-text-muted mt-xs">
            Current: <span class="font-semibold capitalize">{$theme}</span>
            {#if $theme === 'system'}
              <span>({$resolvedTheme})</span>
            {:else}
              <span>mode</span>
            {/if}
          </p>
        </div>
        <div class="w-28">
          <SelectInput
            id="theme-select"
            value={$theme}
            options={themeOptions}
            onchange={handleThemeChange}
            required={true}
            testId="theme-select"
          />
        </div>
      </div>
    </Panel>

    <Panel title="Notifications">
      <div class="space-y-md">
        <div class="flex items-center justify-between py-sm">
          <div>
            <h4 class="text-sm font-medium text-text">Budget Alerts</h4>
            <p class="text-xs text-text-muted mt-xs">Enable threshold notifications</p>
          </div>
          <Toggle 
            id="notification-toggle" 
            bind:checked={notificationsEnabled} 
            onchange={handleNotificationToggle}
            disabled={$settings.loading || $settings.saving || !$settings.data}
          />
        </div>
        
        <div class="pt-md border-t border-panel-border">
          <h4 class="text-sm font-medium text-text">System Permission</h4>
          <p class="text-xs text-text-muted mt-xs">
            Status: <span class="font-semibold capitalize" data-testid="system-permission">{systemPermission}</span>
          </p>
        </div>
      </div>
    </Panel>

    <Panel title="Data Management">
      <div class="space-y-md">
        <div>
          <h4 class="text-sm font-medium text-text">Database Path</h4>
          <div class="mt-sm p-sm bg-background-hover rounded border border-panel-border overflow-x-auto">
            <code class="text-xs text-text-muted whitespace-nowrap">{databasePath}</code>
          </div>
        </div>
        
        <div class="pt-md border-t border-panel-border">
          <h4 class="text-sm font-medium text-text">Last Update</h4>
          <p class="text-xs text-text-muted mt-xs" data-testid="last-refresh">{lastRefreshTime}</p>
        </div>
      </div>
    </Panel>

    <Panel title="About">
      <div class="space-y-md">
        <div>
          <h4 class="text-sm font-medium text-text">Application</h4>
          <p class="text-sm text-text-muted mt-xs">{appTitle}</p>
        </div>
        
        <div class="pt-md border-t border-panel-border">
          <h4 class="text-sm font-medium text-text">Version</h4>
          <p class="text-sm text-text-muted mt-xs">v1.0.0 (Local Build)</p>
        </div>
      </div>
    </Panel>
  </div>
</div>
