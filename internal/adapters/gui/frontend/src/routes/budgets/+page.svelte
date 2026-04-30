<script lang="ts">
  import { onMount } from 'svelte';
  import { z } from 'zod';
  import { appTitle } from '$lib/scaffold-readiness';
  import { budget, loadBudget, saveBudgetAndRefresh } from '$lib/stores/budget';
  import { notificationStore } from '$lib/stores/notification';
  import Form from '$lib/components/forms/Form.svelte';
  import FormField from '$lib/components/forms/FormField.svelte';
  import NumberInput from '$lib/components/forms/NumberInput.svelte';
  import Panel from '$lib/components/ui/Panel.svelte';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import AlertCard from '$lib/components/ui/AlertCard.svelte';
  import PieChart from '$lib/components/charts/PieChart.svelte';
  import LineChart from '$lib/components/charts/LineChart.svelte';
  import type { BudgetInput } from '$lib/types/forms';

  const budgetSchema = z.object({
    limitUsd: z.number().min(0.01, 'Limit must be greater than 0'),
    warningThresholdPercent: z.number().min(1, 'Must be between 1 and 99').max(99, 'Must be between 1 and 99'),
    criticalThresholdPercent: z.number().min(1, 'Must be between 1 and 100').max(100, 'Must be between 1 and 100')
  }).refine(data => data.warningThresholdPercent < data.criticalThresholdPercent, {
    message: "Warning threshold must be less than critical threshold",
    path: ["warningThresholdPercent"]
  });

  let formData: BudgetInput = $state({
    budgetId: crypto.randomUUID(),
    name: 'Monthly Budget',
    provider: '',
    projectHash: '',
    periodMonth: new Date().toISOString().substring(0, 7),
    limitUsd: 100,
    warningThresholdPercent: 80,
    criticalThresholdPercent: 95,
    currency: 'USD'
  });

  let errors: Record<string, string> = $state({});
  let isSubmitting = $state(false);

  onMount(async () => {
    try {
      await loadBudget();
    } catch (error) {
      notificationStore.addNotification({
        id: crypto.randomUUID(),
        title: 'Error',
        body: error instanceof Error ? error.message : 'Failed to load budget data',
        kind: 'error',
        severity: 'critical'
      });
      return;
    }
    
    if ($budget.data.budgets && $budget.data.budgets.length > 0) {
      const existing = $budget.data.budgets[0];
      formData = {
        budgetId: existing.budgetId,
        name: existing.name,
        provider: existing.provider,
        projectHash: existing.projectHash,
        periodMonth: $budget.data.dashboard?.period.month || new Date().toISOString().substring(0, 7),
        limitUsd: existing.limitUsd,
        warningThresholdPercent: existing.warningThresholdPercent || 80,
        criticalThresholdPercent: existing.criticalThresholdPercent || 100,
        currency: existing.currency
      };
    } else if ($budget.data.dashboard?.period.month) {
      formData.periodMonth = $budget.data.dashboard.period.month;
    }
  });

  async function handleSubmit() {
    const result = budgetSchema.safeParse(formData);
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
      const response = await saveBudgetAndRefresh({
        ...formData,
        warningThresholdPercent: formData.warningThresholdPercent,
        criticalThresholdPercent: formData.criticalThresholdPercent
      });

      if (response.result.success) {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Success',
          body: 'Budget saved',
          kind: 'info',
          severity: 'info'
        });
      } else {
        notificationStore.addNotification({
          id: crypto.randomUUID(),
          title: 'Error',
          body: response.result.error?.message || 'Failed to save budget',
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
    }).format(value);
  };

  let currentBudget = $derived($budget.data.budgets && $budget.data.budgets.length > 0 ? $budget.data.budgets[0] : null);
  
  let spendPercent = $derived(currentBudget ? (currentBudget.currentSpendUsd / currentBudget.limitUsd) : 0);
  
  let progressColor = $derived.by(() => {
    if (!currentBudget) return 'text-status-normal';
    if (currentBudget.budgetOverrunActive || spendPercent >= 1) return 'text-status-danger';
    
    const warningThreshold = formData.warningThresholdPercent / 100;
    if (spendPercent >= warningThreshold) return 'text-status-warning';
    
    return 'text-status-success';
  });

  let providerChartData = $derived.by(() => {
    if (!$budget.data.dashboard?.providerSummaries) return [];
    return $budget.data.dashboard.providerSummaries.map(p => ({
      name: p.provider,
      value: p.totalSpendUsd
    }));
  });

  let dailyCostTrendData = $derived.by(() => {
    const sessions = $budget.data.dashboard?.recentSessions || [];
    if (sessions.length === 0) return { xAxis: [], series: [] };

    const dailyCosts = new Map<string, number>();
    
    sessions.forEach(session => {
      const date = session.startedAt.split('T')[0];
      const current = dailyCosts.get(date) || 0;
      dailyCosts.set(date, current + session.totalCostUsd);
    });

    const sortedDates = Array.from(dailyCosts.keys()).sort();
    
    const xAxis = sortedDates.map(d => new Date(d).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }));
    
    let cumulative = 0;
    const data = sortedDates.map(d => {
      cumulative += dailyCosts.get(d) || 0;
      return cumulative;
    });

    return {
      xAxis,
      series: [
        {
          name: 'Cumulative Spend (USD)',
          data
        }
      ]
    };
  });
</script>

<svelte:head>
  <title>{appTitle} - Budgets</title>
</svelte:head>

<div class="p-xl max-w-6xl mx-auto space-y-xl relative">
  <div>
    <h1 class="text-3xl font-bold text-text">Budget Management</h1>
    <p class="mt-sm text-text-muted">Set monthly limits and monitor your spending.</p>
  </div>

  {#if $budget.error}
    <div class="p-md bg-status-danger/10 border border-status-danger/30 rounded-compact text-status-danger text-sm">
      {$budget.error}
    </div>
  {/if}

  <div class="grid grid-cols-3 gap-xl">
    <div>
      <div class="bg-card border border-panel-border rounded-lg p-lg shadow-sm">
        <h2 class="text-xl font-semibold text-text mb-lg">Budget Settings</h2>
        
        <Form onsubmit={handleSubmit}>
          <FormField id="limitUsd" label="Monthly Limit (USD)" error={errors.limitUsd} required>
            <NumberInput
              id="limitUsd"
              bind:value={formData.limitUsd}
              error={!!errors.limitUsd}
              step="0.01"
            />
          </FormField>

          <FormField id="warningThresholdPercent" label="Warning Threshold (%)" error={errors.warningThresholdPercent} required>
            <NumberInput
              id="warningThresholdPercent"
              bind:value={formData.warningThresholdPercent}
              error={!!errors.warningThresholdPercent}
            />
          </FormField>

          <FormField id="criticalThresholdPercent" label="Critical Threshold (%)" error={errors.criticalThresholdPercent} required>
            <NumberInput
              id="criticalThresholdPercent"
              bind:value={formData.criticalThresholdPercent}
              error={!!errors.criticalThresholdPercent}
            />
          </FormField>

          <div class="flex justify-end mt-xl">
            <button
              type="submit"
              class="px-md py-sm text-sm font-medium bg-primary text-primary-foreground rounded-compact hover:bg-primary/90 transition-colors disabled:opacity-50"
              disabled={isSubmitting}
            >
              {isSubmitting ? 'Saving...' : 'Save Budget'}
            </button>
          </div>
        </Form>
      </div>
    </div>

    <div class="col-span-2 space-y-xl">
      {#if currentBudget}
        <div class="bg-card border border-panel-border rounded-lg p-lg shadow-sm">
          <h2 class="text-xl font-semibold text-text mb-lg">Current Month Progress</h2>
          
          <div class="mb-2 flex justify-between items-end">
            <div>
              <span class="text-3xl font-bold text-text">{formatCurrency(currentBudget.currentSpendUsd, currentBudget.currency)}</span>
              <span class="text-text-muted ml-2">of {formatCurrency(currentBudget.limitUsd, currentBudget.currency)}</span>
            </div>
            <div class="text-right">
              <span class="text-lg font-medium text-text">{formatPercent(spendPercent)}</span>
            </div>
          </div>
          
          <div class="w-full bg-background-active rounded-full h-4 mb-lg overflow-hidden">
            <svg class="w-full h-full" preserveAspectRatio="none" viewBox="0 0 100 100">
              <rect x="0" y="0" width={Math.min(spendPercent * 100, 100)} height="100" class="fill-current {progressColor} transition-all duration-500" />
            </svg>
          </div>
          
          <div class="grid grid-cols-2 gap-md">
            <StatCard 
              label="Remaining Budget" 
              value={formatCurrency(currentBudget.remainingUsd, currentBudget.currency)} 
            />
          </div>
          
          {#if currentBudget.budgetOverrunActive || spendPercent >= formData.warningThresholdPercent / 100}
            <div class="mt-lg space-y-md">
              {#if currentBudget.budgetOverrunActive || spendPercent >= 1}
                <AlertCard 
                  title="Budget Exceeded" 
                  message="You have exceeded your monthly budget limit." 
                  variant="danger" 
                />
              {:else if spendPercent >= formData.criticalThresholdPercent / 100}
                <AlertCard 
                  title="Critical Threshold Reached" 
                  message="Warning: {Math.round(spendPercent * 100)}% of budget used. You are approaching your limit." 
                  variant="danger" 
                />
              {:else if spendPercent >= formData.warningThresholdPercent / 100}
                <AlertCard 
                  title="Warning Threshold Reached" 
                  message="Warning: {Math.round(spendPercent * 100)}% of budget used." 
                  variant="warning" 
                />
              {/if}
            </div>
          {/if}
        </div>
        
        <div class="grid grid-cols-2 gap-lg">
          <Panel title="Provider Costs" class="h-80">
            {#if providerChartData.length > 0}
              <PieChart 
                data={providerChartData} 
                name="Cost (USD)" 
                donut={true}
              />
            {:else}
              <div class="flex items-center justify-center h-full text-sm text-text-muted">
                No provider data available
              </div>
            {/if}
          </Panel>
          
          <Panel title="Cumulative Spend" class="h-80">
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
        </div>
      {:else if !$budget.loading}
        <div class="flex flex-col items-center justify-center p-2xl bg-card border border-panel-border rounded-lg text-center h-full">
          <div class="w-16 h-16 mb-md rounded-full bg-background-active flex items-center justify-center text-text-muted">
            <svg class="w-8 h-8" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8c-1.657 0-3 .895-3 2s1.343 2 3 2 3 .895 3 2-1.343 2-3 2m0-8c1.11 0 2.08.402 2.599 1M12 8V7m0 1v8m0 0v1m0-1c-1.11 0-2.08-.402-2.599-1M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
          </div>
          <h2 class="text-lg font-semibold text-text mb-sm">No Budget Set</h2>
          <p class="text-sm text-text-muted max-w-md">
            Set a monthly limit using the form to start monitoring your spending.
          </p>
        </div>
      {/if}
    </div>
  </div>
</div>
