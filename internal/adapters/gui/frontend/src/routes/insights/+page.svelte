<script lang="ts">
  import { onMount } from 'svelte';
  import type { Component } from 'svelte';
  import { appTitle } from '$lib/scaffold-readiness';
  import { waste, loadWaste, refreshWaste } from '$lib/stores/waste';
  import StatCard from '$lib/components/ui/StatCard.svelte';
  import Panel from '$lib/components/ui/Panel.svelte';
  import BarChart from '$lib/components/charts/BarChart.svelte';
  import LineChart from '$lib/components/charts/LineChart.svelte';
  import DataTable from '$lib/components/tables/DataTable.svelte';
  import type { Column } from '$lib/components/tables/DataTable.svelte';
  import StatusBadge from '$lib/components/tables/StatusBadge.svelte';
  import DateCell from '$lib/components/tables/DateCell.svelte';
  import type { InsightState } from '$lib/bindings/insights';

  onMount(() => {
    loadWaste();
  });

  function handleRefresh() {
    refreshWaste();
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

  function getCategoryDescription(category: string): string {
    const map: Record<string, string> = {
      'context_avalanche': 'Excessive context window usage detected',
      'repeated_file_reads': 'Same files read multiple times across turns',
      'retry_amplification': 'High number of retries for the same prompt',
      'over_qualified_model_choice': 'Expensive model used for simple task',
      'tool_schema_bloat': 'Large tool schemas consuming context',
      'planning_tax': 'Excessive planning overhead before execution',
      'zombie_loops': 'Agent stuck in a loop without progress',
      'missed_prompt_caching': 'Prompt caching opportunities missed'
    };
    return map[category] || 'Waste detected';
  }

  function formatCategoryName(category: string): string {
    return category.split('_').map(word => word.charAt(0).toUpperCase() + word.slice(1)).join(' ');
  }

  type InsightRow = Record<string, unknown> & InsightState;

  const columns: Column<InsightRow>[] = [
    {
      key: 'category',
      label: 'Detector',
      sortable: true,
      format: (value: unknown) => formatCategoryName(value as string)
    },
    {
      key: 'severity',
      label: 'Severity',
      sortable: true,
      component: StatusBadge as unknown as Component<Record<string, unknown>>,
      componentProps: (row: InsightRow) => {
        let displayStatus = 'Normal';
        if (row.severity === 'high') displayStatus = 'High (Danger)';
        else if (row.severity === 'medium') displayStatus = 'Medium (Warning)';
        else if (row.severity === 'low') displayStatus = 'Low (Success)';
        
        return {
          status: displayStatus
        };
      }
    },
    {
      key: 'description',
      label: 'Description',
      format: (_: unknown, row: InsightRow) => getCategoryDescription(row.category as string)
    },
    {
      key: 'detectedAt',
      label: 'Detected At',
      sortable: true,
      component: DateCell as unknown as Component<Record<string, unknown>>,
      componentProps: (row: InsightRow) => ({ value: row.detectedAt })
    }
  ];

  const insightPageSize = 10;

  let selectedInsight: InsightState | null = null;
  let visibleInsightCount = insightPageSize;
  let previousInsights: InsightRow[] = [];

  function handleRowClick(row: Record<string, unknown>) {
    selectedInsight = row as unknown as InsightState;
  }

  function closeModal() {
    selectedInsight = null;
  }

  function showMoreInsights() {
    visibleInsightCount += insightPageSize;
  }

  $: summary = $waste.data.summary;
  $: insights = ($waste.data.insights?.items || []) as unknown as InsightRow[];
  $: if (insights !== previousInsights) {
    previousInsights = insights;
    visibleInsightCount = insightPageSize;
  }
  $: visibleInsights = insights.slice(0, visibleInsightCount);
  $: hasMoreInsights = visibleInsights.length < insights.length;
  
  $: topCausesXAxis = summary?.topCauses.map(c => formatCategoryName(c.category as string)) || [];
  $: topCausesSeries = [{
    name: 'Cost (USD)',
    data: summary?.topCauses.map(c => c.attributedCostUsd) || []
  }];

  $: dailyTrendXAxis = summary?.dailyTrend.map(t => new Date(t.day).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })) || [];
  $: dailyTrendSeries = [{
    name: 'Cost (USD)',
    data: summary?.dailyTrend.map(t => t.wasteCostUsd) || []
  }];

  $: wasteHeadline = summary?.topCauses && summary.topCauses.length > 0 
    ? formatCategoryName(summary.topCauses[0].category as string) 
    : 'No Waste Detected';
</script>

<svelte:head>
  <title>{appTitle} - Insights</title>
</svelte:head>

<div class="p-8 flex flex-col gap-lg h-full overflow-y-auto">
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-3xl font-bold text-text m-0">Insights</h1>
      <p class="text-text-muted mt-xs mb-0">Waste detection and optimization opportunities</p>
    </div>
    <button 
      class="px-md py-sm bg-primary text-primary-foreground rounded-compact font-medium hover:bg-primary-hover transition-colors disabled:opacity-50"
      on:click={handleRefresh}
      disabled={$waste.loading}
    >
      {$waste.loading ? 'Refreshing...' : 'Refresh'}
    </button>
  </div>

  {#if $waste.error}
    <div class="p-md bg-status-danger/10 border border-status-danger/30 rounded-compact text-status-danger">
      {$waste.error}
    </div>
  {/if}

  {#if summary}
    <div class="grid grid-cols-4 gap-md">
      <StatCard 
        label="Waste Headline" 
        value={wasteHeadline} 
      />
      <StatCard 
        label="Waste %" 
        value={formatPercent(summary.wastePercent)} 
        trend={summary.wastePercent > 10 ? 'down' : 'none'}
      />
      <StatCard 
        label="Projected Waste" 
        value={formatCurrency(summary.projectedMonthEndWasteUsd)} 
      />
      <StatCard 
        label="Weekly Waste" 
        value={formatCurrency(summary.weeklyWasteCostUsd)} 
      />
    </div>

    <div class="grid grid-cols-2 gap-md">
      <Panel title="Top Waste Causes">
        <div class="h-64">
          {#if topCausesXAxis.length > 0}
            <BarChart 
              xAxisData={topCausesXAxis} 
              series={topCausesSeries} 
              yAxisName="Cost (USD)" 
            />
          {:else}
            <div class="flex items-center justify-center h-full text-text-muted">
              No waste causes found.
            </div>
          {/if}
        </div>
      </Panel>

      <Panel title="Daily Waste Trend (30-day)">
        <div class="h-64">
          {#if dailyTrendXAxis.length > 0}
            <LineChart 
              xAxisData={dailyTrendXAxis} 
              series={dailyTrendSeries} 
              yAxisName="Cost (USD)" 
            />
          {:else}
            <div class="flex items-center justify-center h-full text-text-muted">
              No trend data available.
            </div>
          {/if}
        </div>
      </Panel>
    </div>
  {/if}

  <Panel title="Insights Log">
    <div class="flex flex-col gap-md">
      <DataTable 
        data={visibleInsights} 
        {columns} 
        loading={$waste.loading} 
        emptyMessage="No insights found for this period."
        onRowClick={handleRowClick}
      />

      {#if insights.length > 0}
        <div class="flex items-center justify-between text-sm text-text-muted">
          <span>Showing {visibleInsights.length} of {insights.length} insights</span>
          {#if hasMoreInsights}
            <button
              class="px-md py-sm bg-background-hover text-text border border-panel-border rounded-compact font-medium hover:bg-panel-border transition-colors"
              type="button"
              on:click={showMoreInsights}
            >
              Show 10 more
            </button>
          {/if}
        </div>
      {/if}
    </div>
  </Panel>
</div>

{#if selectedInsight}
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm p-4">
    <div class="bg-card border border-panel-border rounded-compact shadow-lg w-full max-w-2xl max-h-[90vh] flex flex-col overflow-hidden">
      <div class="flex items-center justify-between p-md border-b border-panel-border bg-background-hover">
        <h2 class="text-lg font-bold text-text m-0">Insight Details</h2>
        <button 
          class="text-text-muted hover:text-text transition-colors"
          on:click={closeModal}
          aria-label="Close modal"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>
      
      <div class="p-md overflow-y-auto flex-1 flex flex-col gap-md">
        <div class="grid grid-cols-2 gap-md">
          <div>
            <div class="text-xs font-medium text-text-muted mb-1">Detector</div>
            <div class="text-sm font-semibold text-text">{formatCategoryName(selectedInsight.category.toString())}</div>
          </div>
          <div>
            <div class="text-xs font-medium text-text-muted mb-1">Severity</div>
            <StatusBadge status={
              selectedInsight.severity === 'high' ? 'High (Danger)' :
              selectedInsight.severity === 'medium' ? 'Medium (Warning)' :
              selectedInsight.severity === 'low' ? 'Low (Success)' : 'Normal'
            } />
          </div>
          <div>
            <div class="text-xs font-medium text-text-muted mb-1">Detected At</div>
            <DateCell value={selectedInsight.detectedAt} />
          </div>
          <div>
            <div class="text-xs font-medium text-text-muted mb-1">Description</div>
            <div class="text-sm text-text">{getCategoryDescription(selectedInsight.category.toString())}</div>
          </div>
        </div>
        
        {#if selectedInsight.payload}
          <div class="mt-sm">
            <h3 class="text-sm font-semibold text-text mb-sm border-b border-panel-border pb-xs">Payload Details</h3>
            
            {#if selectedInsight.payload.metrics && selectedInsight.payload.metrics.length > 0}
              <div class="mb-sm">
                <div class="text-xs font-medium text-text-muted mb-1">Metrics</div>
                <div class="bg-background rounded p-sm border border-panel-border">
                  {#each selectedInsight.payload.metrics as metric}
                    <div class="flex justify-between text-sm py-1 border-b border-panel-border/50 last:border-0">
                      <span class="text-text-muted">{metric.key}</span>
                      <span class="font-mono text-text">{metric.value} {metric.unit}</span>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
            
            {#if selectedInsight.payload.counts && selectedInsight.payload.counts.length > 0}
              <div class="mb-sm">
                <div class="text-xs font-medium text-text-muted mb-1">Counts</div>
                <div class="bg-background rounded p-sm border border-panel-border">
                  {#each selectedInsight.payload.counts as count}
                    <div class="flex justify-between text-sm py-1 border-b border-panel-border/50 last:border-0">
                      <span class="text-text-muted">{count.key}</span>
                      <span class="font-mono text-text">{count.value}</span>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
            
            {#if selectedInsight.payload.hashes && selectedInsight.payload.hashes.length > 0}
              <div class="mb-sm">
                <div class="text-xs font-medium text-text-muted mb-1">Hashes</div>
                <div class="bg-background rounded p-sm border border-panel-border">
                  {#each selectedInsight.payload.hashes as hash}
                    <div class="flex justify-between text-sm py-1 border-b border-panel-border/50 last:border-0">
                      <span class="text-text-muted">{hash.kind}</span>
                      <span class="font-mono text-text truncate max-w-[200px]" title={hash.value}>{hash.value}</span>
                    </div>
                  {/each}
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </div>
      
      <div class="p-md border-t border-panel-border bg-background-hover flex justify-end">
        <button 
          class="px-md py-sm bg-background border border-panel-border text-text rounded-compact font-medium hover:bg-background-active transition-colors"
          on:click={closeModal}
        >
          Close
        </button>
      </div>
    </div>
  </div>
{/if}
