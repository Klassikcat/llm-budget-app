<script lang="ts">
  import { onMount } from 'svelte';
  import type { Component } from 'svelte';
  import { RefreshCw } from 'lucide-svelte';
  import { dashboardStore, loadDashboardData, refreshDashboardData } from '$lib/stores/dashboard';
  import Panel from '$lib/components/ui/Panel.svelte';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import BarChart from '$lib/components/charts/BarChart.svelte';
  import LineChart from '$lib/components/charts/LineChart.svelte';
  import DataTable from '$lib/components/tables/DataTable.svelte';
  import type { Column } from '$lib/components/tables/DataTable.svelte';
  import CurrencyCell from '$lib/components/tables/CurrencyCell.svelte';
  import DateCell from '$lib/components/tables/DateCell.svelte';
  import TokenCell from '$lib/components/tables/TokenCell.svelte';
  import StatusBadge from '$lib/components/tables/StatusBadge.svelte';
  import type { DashboardRecentSession, DashboardBudget } from '$lib/bindings';

  onMount(() => {
    loadDashboardData();
  });

  function handleRefresh() {
    refreshDashboardData();
  }

  const formatCurrency = (value: number, currency = 'USD') => {
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: currency,
      minimumFractionDigits: 2,
      maximumFractionDigits: 2
    }).format(value);
  };

  const formatPercent = (value: number) => {
    return new Intl.NumberFormat('en-US', {
      style: 'percent',
      minimumFractionDigits: 1,
      maximumFractionDigits: 1
    }).format(value / 100);
  };

  const formatNumber = (value: number) => {
    return new Intl.NumberFormat('en-US').format(value);
  };

  type SessionRow = Record<string, unknown> & DashboardRecentSession;
  type BudgetRow = Record<string, unknown> & DashboardBudget;

  const sessionColumns: Column<SessionRow>[] = [
    { key: 'projectName', label: 'Project' },
    { key: 'agentName', label: 'Agent' },
    { key: 'provider', label: 'Provider' },
    { key: 'modelId', label: 'Model' },
    { 
      key: 'startedAt', 
      label: 'Date',
      component: DateCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: SessionRow) => ({ value: row.startedAt })
    },
    { 
      key: 'totalTokens', 
      label: 'Tokens',
      align: 'right',
      component: TokenCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: SessionRow) => ({ value: row.totalTokens })
    },
    { 
      key: 'totalCostUsd', 
      label: 'Cost',
      align: 'right',
      component: CurrencyCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: SessionRow) => ({ value: row.totalCostUsd, currency: row.currency })
    }
  ];

  const budgetColumns: Column<BudgetRow>[] = [
    { key: 'name', label: 'Budget' },
    { key: 'provider', label: 'Provider' },
    { 
      key: 'limitUsd', 
      label: 'Limit',
      align: 'right',
      component: CurrencyCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: BudgetRow) => ({ value: row.limitUsd, currency: row.currency })
    },
    { 
      key: 'currentSpendUsd', 
      label: 'Spend',
      align: 'right',
      component: CurrencyCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: BudgetRow) => ({ value: row.currentSpendUsd, currency: row.currency })
    },
    { 
      key: 'remainingUsd', 
      label: 'Remaining',
      align: 'right',
      component: CurrencyCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: BudgetRow) => ({ value: row.remainingUsd, currency: row.currency })
    },
    {
      key: 'status',
      label: 'Status',
      align: 'center',
      component: StatusBadge as unknown as Component<Record<string, unknown>>,
      componentProps: (row: BudgetRow) => {
        const percent = (row.currentSpendUsd / row.limitUsd) * 100;
        let status: 'success' | 'warning' | 'danger' | 'info' = 'success';
        if (row.budgetOverrunActive || percent >= 100) status = 'danger';
        else if (percent >= 80) status = 'warning';
        
        return {
          status,
          label: formatPercent(percent)
        };
      }
    }
  ];

  $: providerChartData = {
    xAxis: $dashboardStore.data.dashboard?.providerSummaries?.map(p => p.provider) || [],
    series: [
      {
        name: 'Cost',
        data: $dashboardStore.data.dashboard?.providerSummaries?.map(p => p.totalSpendUsd) || []
      }
    ]
  };

  $: dailyCostTrendData = (() => {
    const sessions = $dashboardStore.data.dashboard?.recentSessions || [];
    if (sessions.length === 0) return { xAxis: [], series: [] };

    const dailyCosts = new Map<string, number>();
    
    sessions.forEach(session => {
      const date = session.startedAt.split('T')[0];
      const current = dailyCosts.get(date) || 0;
      dailyCosts.set(date, current + session.totalCostUsd);
    });

    const sortedDates = Array.from(dailyCosts.keys()).sort();
    
    const xAxis = sortedDates.map(d => new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }));
    const data = sortedDates.map(d => dailyCosts.get(d) || 0);

    return {
      xAxis,
      series: [
        {
          name: 'Cost (USD)',
          data
        }
      ]
    };
  })();
</script>

<svelte:head>
  <title>Dashboard - LLM Budget Tracker</title>
</svelte:head>

<div class="flex flex-col gap-lg p-lg max-w-7xl mx-auto w-full">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold text-text m-0">Dashboard</h1>
    <button 
      class="flex items-center gap-xs px-md py-xs bg-card border border-panel-border rounded-compact text-sm font-medium text-text-muted hover:text-text hover:bg-background-hover transition-colors"
      on:click={handleRefresh}
      disabled={$dashboardStore.loading}
    >
      <RefreshCw size={16} class={$dashboardStore.loading ? 'animate-spin' : ''} />
      Refresh
    </button>
  </div>

  {#if $dashboardStore.error}
    <div class="p-md bg-status-danger/10 border border-status-danger/20 rounded-compact text-status-danger text-sm">
      {$dashboardStore.error}
    </div>
  {/if}

  {#if $dashboardStore.data.dashboard?.empty}
    <div class="flex flex-col items-center justify-center p-2xl bg-card border border-panel-border rounded-compact text-center">
      <div class="w-16 h-16 mb-md rounded-full bg-background-active flex items-center justify-center text-text-muted">
        <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"></path>
        </svg>
      </div>
      <h2 class="text-lg font-semibold text-text mb-xs">No Data Available</h2>
      <p class="text-sm text-text-muted max-w-md">
        There is no usage data or budgets configured for the current period. 
        Start using the CLI or configure a budget to see insights here.
      </p>
    </div>
  {:else}
    <div class="grid grid-cols-4 gap-md">
      <StatCard 
        label="Total Spend" 
        value={formatCurrency($dashboardStore.data.dashboard?.totals?.totalSpendUsd || 0, $dashboardStore.data.dashboard?.totals?.currency)} 
      />
      <StatCard 
        label="Total Tokens" 
        value={formatNumber($dashboardStore.data.graphs?.modelTokenUsages?.reduce((sum, m) => sum + m.totalTokens, 0) || 0)} 
      />
      <StatCard 
        label="Subscription Cost" 
        value={formatCurrency($dashboardStore.data.dashboard?.totals?.subscriptionSpendUsd || 0, $dashboardStore.data.dashboard?.totals?.currency)} 
      />
      <StatCard 
        label="Waste %" 
        value={formatPercent($dashboardStore.data.waste?.wastePercent || 0)} 
        trend={$dashboardStore.data.waste?.wastePercent && $dashboardStore.data.waste.wastePercent > 10 ? 'down' : 'none'}
      />
    </div>

    <div class="grid grid-cols-2 gap-md">
      <Panel title="Daily Cost Trend" class="h-80">
        {#if dailyCostTrendData.xAxis.length > 0}
          <LineChart 
            xAxisData={dailyCostTrendData.xAxis} 
            series={dailyCostTrendData.series} 
            yAxisName="Cost (USD)" 
          />
        {:else}
          <div class="flex items-center justify-center h-full text-sm text-text-muted">
            No trend data available
          </div>
        {/if}
      </Panel>

      <Panel title="Provider Costs" class="h-80">
        {#if providerChartData.xAxis.length > 0}
          <BarChart 
            xAxisData={providerChartData.xAxis}
            series={providerChartData.series} 
            yAxisName="Cost (USD)" 
          />
        {:else}
          <div class="flex items-center justify-center h-full text-sm text-text-muted">
            No provider data available
          </div>
        {/if}
      </Panel>
    </div>

    <div class="grid grid-cols-2 gap-md">
      <Panel title="Budgets">
        <DataTable 
          data={($dashboardStore.data.dashboard?.budgets || []) as unknown as BudgetRow[]} 
          columns={budgetColumns} 
          loading={$dashboardStore.loading}
          emptyMessage="No budgets configured"
        />
      </Panel>

      <Panel title="Recent Sessions">
        <DataTable 
          data={($dashboardStore.data.dashboard?.recentSessions || []) as unknown as SessionRow[]} 
          columns={sessionColumns} 
          loading={$dashboardStore.loading}
          emptyMessage="No recent sessions"
        />
      </Panel>
    </div>
  {/if}
</div>
