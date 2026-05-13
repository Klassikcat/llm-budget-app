<script lang="ts">
  import { onMount } from 'svelte';
  import { appTitle } from '$lib/scaffold-readiness';
  import { subscription, loadSubscription, deleteSubscriptionAndRefresh } from '$lib/stores/subscription';
  import { notificationStore } from '$lib/stores/notification';
  import DataTable from '$lib/components/tables/DataTable.svelte';
  import DateCell from '$lib/components/tables/DateCell.svelte';
  import CurrencyCell from '$lib/components/tables/CurrencyCell.svelte';
  import StatusBadge from '$lib/components/tables/StatusBadge.svelte';
  import DeleteActionCell from '$lib/components/tables/DeleteActionCell.svelte';
  import type { SubscriptionState } from '$lib/types/forms';
  import type { Column } from '$lib/components/tables/DataTable.svelte';
  import type { Component } from 'svelte';

  let isDeleting = $state(false);

  onMount(async () => {
    try {
      await loadSubscription();
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: error instanceof Error ? error.message : 'Failed to load subscriptions',
        kind: 'error',
        severity: 'critical'
      });
    }
  });

  async function handleDelete(row: SubscriptionState) {
    isDeleting = true;
    try {
      const response = await deleteSubscriptionAndRefresh(row.subscriptionId);
      
      if (response.success) {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Success',
          body: 'Subscription deleted',
          kind: 'info',
          severity: 'info'
        });
      } else {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Error',
          body: response.error?.message || 'Failed to delete subscription',
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
      isDeleting = false;
    }
  }

  const columns: Column<Record<string, unknown>>[] = [
    {
      key: 'startsAt',
      label: 'Date',
      component: DateCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row) => ({ value: row.startsAt as string, format: 'short' }),
      sortable: true
    },
    {
      key: 'provider',
      label: 'Provider',
      sortable: true
    },
    {
      key: 'planName',
      label: 'Plan',
      sortable: true
    },
    {
      key: 'feeUsd',
      label: 'Fee',
      component: CurrencyCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row) => ({ value: row.feeUsd as number }),
      align: 'right' as const,
      sortable: true
    },
    {
      key: 'renewalDay',
      label: 'Renewal Day',
      align: 'right' as const,
      sortable: true
    },
    {
      key: 'isActive',
      label: 'Status',
      component: StatusBadge as unknown as Component<Record<string, unknown>>,
      componentProps: (row) => ({ 
        status: row.isActive ? 'Active' : 'Inactive',
      }),
      align: 'center' as const,
      sortable: true
    },
    {
      key: 'actions',
      label: '',
      component: DeleteActionCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row) => ({ 
        onDelete: () => handleDelete(row as unknown as SubscriptionState),
        disabled: isDeleting
      }),
      align: 'right' as const
    }
  ];

  let tableData = $derived($subscription.data.items as unknown as Record<string, unknown>[]);
</script>

<svelte:head>
  <title>{appTitle} - Subscriptions</title>
</svelte:head>

<div class="p-xl max-w-6xl mx-auto space-y-xl relative">
  <div class="flex justify-between items-center">
    <div>
      <h1 class="text-3xl font-bold text-text">Subscriptions</h1>
      <p class="mt-sm text-text-muted">Manage your API subscriptions and recurring costs.</p>
    </div>
    <a
      href="/subscriptions/new"
      class="px-md py-sm text-sm font-medium bg-primary text-primary-foreground rounded-compact hover:bg-primary/90 transition-colors"
    >
      Add Subscription
    </a>
  </div>

  {#if $subscription.error}
    <div class="p-md bg-status-danger/10 border border-status-danger/30 rounded-compact text-status-danger text-sm">
      {$subscription.error}
    </div>
  {/if}

  <div class="bg-card border border-panel-border rounded-lg p-lg shadow-sm">
    <DataTable
      data={tableData}
      {columns}
      loading={$subscription.loading}
      emptyMessage="No subscriptions found."
    />
  </div>
</div>
