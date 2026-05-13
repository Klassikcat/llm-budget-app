<script lang="ts">
  import { onMount } from 'svelte';
  import { z } from 'zod';
  import { appTitle } from '$lib/scaffold-readiness';
  import { usage, loadUsage, saveManualEntryAndRefresh } from '$lib/stores/usage';
  import { notificationStore } from '$lib/stores/notification';
  import Form from '$lib/components/forms/Form.svelte';
  import FormField from '$lib/components/forms/FormField.svelte';
  import TextInput from '$lib/components/forms/TextInput.svelte';
  import NumberInput from '$lib/components/forms/NumberInput.svelte';
  import SelectInput from '$lib/components/forms/SelectInput.svelte';
  import DatePicker from '$lib/components/forms/DatePicker.svelte';
  import DataTable from '$lib/components/tables/DataTable.svelte';
  import DateCell from '$lib/components/tables/DateCell.svelte';
  import CurrencyCell from '$lib/components/tables/CurrencyCell.svelte';
  import type { ManualEntryInput, ManualEntryState } from '$lib/types/forms';
  import type { DashboardRecentSession } from '$lib/bindings';
  import type { Column } from '$lib/components/tables/DataTable.svelte';
  import type { Component } from 'svelte';

  const manualEntrySchema = z.object({
    provider: z.string().min(1, 'Provider is required'),
    modelId: z.string().min(1, 'Model ID is required'),
    occurredAt: z.string().min(1, 'Date is required'),
    inputTokens: z.number().min(0, 'Input tokens cannot be negative'),
    outputTokens: z.number().min(0, 'Output tokens cannot be negative'),
    cachedTokens: z.number().min(0, 'Cached tokens cannot be negative'),
    cacheWriteTokens: z.number().min(0, 'Cache write tokens cannot be negative'),
    projectName: z.string().min(1, 'Project Name is required')
  });

  type UsageRow = Record<string, unknown> & {
    occurredAt: string;
    provider: string;
    modelId: string;
    inputTokens: number;
    outputTokens: number;
    totalCostUsd: number;
    currency?: string;
  };

  let formData: ManualEntryInput = $state({
    provider: '',
    modelId: '',
    occurredAt: new Date().toISOString().split('T')[0],
    inputTokens: 0,
    outputTokens: 0,
    cachedTokens: 0,
    cacheWriteTokens: 0,
    projectName: '',
    metadata: {}
  });

  let errors: Record<string, string> = $state({});
  let isSubmitting = $state(false);
  let manualEntries: ManualEntryState[] = $state([]);

  const providerOptions = [
    { value: 'claude', label: 'Claude Code' },
    { value: 'codex', label: 'Codex' },
    { value: 'gemini', label: 'Gemini' },
    { value: 'opencode', label: 'OpenCode' },
    { value: 'other', label: 'Other' }
  ];

  onMount(async () => {
    try {
      await loadUsage();
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: error instanceof Error ? error.message : 'Failed to load usage data',
        kind: 'error',
        severity: 'critical'
      });
    }
  });

  async function handleSubmit() {
    const result = manualEntrySchema.safeParse(formData);
    if (!result.success) {
      errors = {};
      result.error.issues.forEach((issue) => {
        if (issue.path[0]) {
          errors[issue.path[0].toString()] = issue.message;
        }
      });
      return;
    }

    errors = {};
    isSubmitting = true;

    try {
      const response = await saveManualEntryAndRefresh({
        ...formData,
        occurredAt: new Date(formData.occurredAt).toISOString()
      });

      if (response.result.success) {
        manualEntries = [response.entry, ...manualEntries];
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Success',
          body: 'Usage entry saved',
          kind: 'info',
          severity: 'info'
        });
        formData = {
          provider: '',
          modelId: '',
          occurredAt: new Date().toISOString().split('T')[0],
          inputTokens: 0,
          outputTokens: 0,
          cachedTokens: 0,
          cacheWriteTokens: 0,
          projectName: '',
          metadata: {}
        };
      } else {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Error',
          body: response.result.error?.message || 'Failed to save usage entry',
          kind: 'error',
          severity: 'critical'
        });
      }
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: error instanceof Error ? error.message : 'An unexpected error occurred',
        kind: 'error',
        severity: 'critical'
      });
    } finally {
      isSubmitting = false;
    }
  }

  function handleCancel() {
    formData = {
      provider: '',
      modelId: '',
      occurredAt: new Date().toISOString().split('T')[0],
      inputTokens: 0,
      outputTokens: 0,
      cachedTokens: 0,
      cacheWriteTokens: 0,
      projectName: '',
      metadata: {}
    };
    errors = {};
  }

  function sessionToUsageRow(session: DashboardRecentSession): UsageRow {
    return {
      occurredAt: session.startedAt,
      provider: session.provider,
      modelId: session.modelId,
      inputTokens: session.totalTokens,
      outputTokens: 0,
      totalCostUsd: session.totalCostUsd,
      currency: session.currency
    };
  }

  function entryToUsageRow(entry: ManualEntryState): UsageRow {
    return {
      occurredAt: entry.occurredAt,
      provider: entry.provider,
      modelId: entry.modelId,
      inputTokens: entry.inputTokens,
      outputTokens: entry.outputTokens,
      totalCostUsd: entry.totalCostUsd
    };
  }

  const columns: Column<UsageRow>[] = [
    {
      key: 'occurredAt',
      label: 'Date',
      component: DateCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row) => ({ value: row.occurredAt, format: 'short' }),
      sortable: true
    },
    {
      key: 'provider',
      label: 'Provider',
      sortable: true
    },
    {
      key: 'modelId',
      label: 'Model',
      sortable: true
    },
    {
      key: 'inputTokens',
      label: 'Input Tokens',
      format: (value) => String(value ?? 0)
    },
    {
      key: 'outputTokens',
      label: 'Output Tokens',
      format: (value) => String(value ?? 0)
    },
    {
      key: 'totalCostUsd',
      label: 'Cost',
      component: CurrencyCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row) => ({ value: row.totalCostUsd, currency: row.currency }),
      align: 'right' as const,
      sortable: true
    }
  ];

  let tableData = $derived([
    ...manualEntries.map(entryToUsageRow),
    ...$usage.data.recentSessions.map(sessionToUsageRow)
  ]);
</script>

<svelte:head>
  <title>{appTitle} - Usage</title>
</svelte:head>

<div class="p-xl max-w-6xl mx-auto space-y-xl relative">
  <div>
    <h1 class="text-3xl font-bold text-text">Usage Tracking</h1>
    <p class="mt-sm text-text-muted">Manually enter API usage or view recent session history.</p>
  </div>

  {#if $usage.error}
    <div class="p-md bg-status-danger/10 border border-status-danger/30 rounded-compact text-status-danger text-sm">
      {$usage.error}
    </div>
  {/if}

  <div class="grid grid-cols-3 gap-xl">
    <div>
      <div class="bg-card border border-panel-border rounded-lg p-lg shadow-sm">
        <h2 class="text-xl font-semibold text-text mb-lg">Manual Entry</h2>
        <Form onsubmit={handleSubmit}>
          <FormField id="provider" label="Provider" error={errors.provider} required>
            <SelectInput
              id="provider"
              bind:value={formData.provider}
              options={providerOptions}
              error={!!errors.provider}
            />
          </FormField>

          <FormField id="modelId" label="Model ID" error={errors.modelId} required>
            <TextInput
              id="modelId"
              bind:value={formData.modelId}
              placeholder="e.g. gpt-4o"
              error={!!errors.modelId}
            />
          </FormField>

          <FormField id="occurredAt" label="Occurred At" error={errors.occurredAt} required>
            <DatePicker
              id="occurredAt"
              bind:value={formData.occurredAt}
              error={!!errors.occurredAt}
            />
          </FormField>

          <div class="grid grid-cols-2 gap-md">
            <FormField id="inputTokens" label="Input Tokens" error={errors.inputTokens} required>
              <NumberInput
                id="inputTokens"
                bind:value={formData.inputTokens}
                error={!!errors.inputTokens}
              />
            </FormField>

            <FormField id="outputTokens" label="Output Tokens" error={errors.outputTokens} required>
              <NumberInput
                id="outputTokens"
                bind:value={formData.outputTokens}
                error={!!errors.outputTokens}
              />
            </FormField>
          </div>

          <div class="grid grid-cols-2 gap-md">
            <FormField id="cachedTokens" label="Cached Tokens" error={errors.cachedTokens} required>
              <NumberInput
                id="cachedTokens"
                bind:value={formData.cachedTokens}
                error={!!errors.cachedTokens}
              />
            </FormField>

            <FormField id="cacheWriteTokens" label="Cache Write Tokens" error={errors.cacheWriteTokens} required>
              <NumberInput
                id="cacheWriteTokens"
                bind:value={formData.cacheWriteTokens}
                error={!!errors.cacheWriteTokens}
              />
            </FormField>
          </div>

          <FormField id="projectName" label="Project Name" error={errors.projectName} required>
            <TextInput
              id="projectName"
              bind:value={formData.projectName}
              placeholder="e.g. my-project"
              error={!!errors.projectName}
            />
          </FormField>

          <div class="flex justify-end gap-md mt-xl">
            <button
              type="button"
              class="px-md py-sm text-sm font-medium text-text-muted hover:text-text transition-colors"
              onclick={handleCancel}
              disabled={isSubmitting}
            >
              Cancel
            </button>
            <button
              type="submit"
              class="px-md py-sm text-sm font-medium bg-primary text-primary-foreground rounded-compact hover:bg-primary/90 transition-colors disabled:opacity-50"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Saving...' : 'Save Entry'}
            </button>
          </div>
        </Form>
      </div>
    </div>

    <div class="col-span-2">
      <div class="bg-card border border-panel-border rounded-lg p-lg shadow-sm h-full">
        <h2 class="text-xl font-semibold text-text mb-lg">History</h2>
        <DataTable
          data={tableData}
          {columns}
          loading={$usage.loading}
          emptyMessage="No usage history found."
        />
      </div>
    </div>
  </div>
</div>
