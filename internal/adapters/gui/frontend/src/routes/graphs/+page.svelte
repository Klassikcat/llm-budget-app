<script lang="ts">
  import { onMount } from 'svelte';
  import { appTitle } from '$lib/scaffold-readiness';
  import { graphTimeRanges, loadGraphs, type GraphResponse, type GraphTimeRange } from '$lib/bindings';
  import Panel from '$lib/components/ui/Panel.svelte';
  import SelectInput from '$lib/components/forms/SelectInput.svelte';
  import type { SelectOption } from '$lib/components/forms/SelectInput.svelte';
  import BarChart from '$lib/components/charts/BarChart.svelte';
  import LineChart from '$lib/components/charts/LineChart.svelte';
  import StackedBarChart from '$lib/components/charts/StackedBarChart.svelte';

  let loading = true;
  let error: string | null = null;
  let graphData: GraphResponse | null = null;

  let activeTab: 'Model Token Usage' | 'Model Cost' | 'Daily Token Trend' | 'Model Token Breakdown' = 'Model Token Usage';
  const tabs = ['Model Token Usage', 'Model Cost', 'Daily Token Trend', 'Model Token Breakdown'] as const;

  let timeRange: GraphTimeRange = '30 days';
  const timeRangeOptions: SelectOption[] = graphTimeRanges.map(range => ({ value: range, label: range }));
  let hasMounted = false;
  let latestRequestId = 0;

  const trendDateFormatter = new Intl.DateTimeFormat(undefined, {
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC'
  });

  async function loadGraphData() {
    const requestId = latestRequestId + 1;
    const requestedTimeRange = timeRange;
    latestRequestId = requestId;
    loading = true;
    error = null;

    try {
      const nextGraphData = await loadGraphs('', requestedTimeRange);
      if (requestId !== latestRequestId) {
        return;
      }

      graphData = nextGraphData;
    } catch (e) {
      if (requestId !== latestRequestId) {
        return;
      }

      error = e instanceof Error ? e.message : 'Failed to load graphs';
      graphData = null;
    } finally {
      if (requestId === latestRequestId) {
        loading = false;
      }
    }
  }

  function handleTimeRangeChange(event: Event) {
    const selectedRange = (event.currentTarget as HTMLSelectElement).value;
    if (!graphTimeRanges.includes(selectedRange as GraphTimeRange)) {
      return;
    }

    timeRange = selectedRange as GraphTimeRange;
    if (hasMounted) {
      loadGraphData();
    }
  }

  onMount(() => {
    hasMounted = true;
    loadGraphData();
  });

  $: tokenUsageData = {
    xAxis: graphData?.modelTokenUsages?.map(m => m.modelName) || [],
    series: [
      {
        name: 'Total Tokens',
        data: graphData?.modelTokenUsages?.map(m => m.totalTokens) || []
      }
    ]
  };

  $: costData = {
    xAxis: graphData?.modelCosts?.map(m => m.modelName) || [],
    series: [
      {
        name: 'Cost (USD)',
        data: graphData?.modelCosts?.map(m => m.totalCostUsd) || []
      }
    ]
  };

  $: trendData = (() => {
    if (!graphData?.dailyTokenTrends) return { xAxis: [], series: [] };
    
    const trends = graphData.dailyTokenTrends;

    const xAxis = trends.map(t => {
      const d = new Date(t.date);
      return Number.isNaN(d.getTime()) ? t.date : trendDateFormatter.format(d);
    });

    const modelNames = new Set<string>();
    trends.forEach(t => {
      t.modelBreakdown.forEach(b => {
        modelNames.add(b.modelName);
      });
    });

    const series = Array.from(modelNames).map(modelName => {
      return {
        name: modelName,
        data: trends.map(t => {
          const breakdown = t.modelBreakdown.find(b => b.modelName === modelName);
          return breakdown ? breakdown.totalTokens : 0;
        })
      };
    });

    return { xAxis, series };
  })();

  $: breakdownData = (() => {
    if (!graphData?.modelTokenBreakdowns) return { xAxis: [], series: [] };
    
    const xAxis = graphData.modelTokenBreakdowns.map(m => m.modelName);
    
    return {
      xAxis,
      series: [
        {
          name: 'Input',
          data: graphData.modelTokenBreakdowns.map(m => m.inputTokens)
        },
        {
          name: 'Output',
          data: graphData.modelTokenBreakdowns.map(m => m.outputTokens)
        },
        {
          name: 'Cache Read',
          data: graphData.modelTokenBreakdowns.map(m => m.cacheReadTokens)
        },
        {
          name: 'Cache Write',
          data: graphData.modelTokenBreakdowns.map(m => m.cacheWriteTokens)
        }
      ]
    };
  })();

</script>

<svelte:head>
  <title>{appTitle} - Graphs</title>
</svelte:head>

<div class="flex flex-col gap-lg p-lg max-w-7xl mx-auto w-full h-full">
  <div class="flex items-center justify-between">
    <h1 class="text-2xl font-bold text-text m-0">Graphs</h1>
    
    <div class="flex items-center gap-sm">
      <span class="text-sm text-text-muted">Time Range:</span>
      <div class="w-28">
        <SelectInput
          id="time-range-selector"
          bind:value={timeRange}
          options={timeRangeOptions}
          onchange={handleTimeRangeChange}
          required={true}
          testId="time-range-selector"
        />
      </div>
    </div>
  </div>

  {#if error}
    <div class="p-md bg-status-danger/10 border border-status-danger/20 rounded-compact text-status-danger text-sm">
      {error}
    </div>
  {/if}

  <div class="flex border-b border-panel-border mb-md">
    {#each tabs as tab}
      <button
        class="px-md py-sm text-sm font-medium border-b-2 transition-colors {activeTab === tab ? 'border-primary text-primary' : 'border-transparent text-text-muted hover:text-text hover:border-panel-border'}"
        onclick={() => activeTab = tab}
        data-testid={`tab-${tab.replace(/\s+/g, '-')}`}
      >
        {tab}
      </button>
    {/each}
  </div>

  <div class="flex-1 min-h-[500px]">
    {#if loading}
      <div class="flex items-center justify-center h-full" data-testid="loading-spinner">
        <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-primary"></div>
      </div>
    {:else if !graphData}
      <div class="flex items-center justify-center h-full text-text-muted" data-testid="empty-state">
        No data available
      </div>
    {:else}
      <Panel title={activeTab} class="h-full flex flex-col">
        <div class="flex-1 w-full h-full min-h-[400px]">
          {#if activeTab === 'Model Token Usage'}
            {#if tokenUsageData.xAxis.length > 0}
              <BarChart 
                xAxisData={tokenUsageData.xAxis}
                series={tokenUsageData.series}
                yAxisName="Tokens"
                colorByData={true}
              />
            {:else}
              <div class="flex items-center justify-center h-full text-text-muted" data-testid="empty-chart">No token usage data</div>
            {/if}
          {:else if activeTab === 'Model Cost'}
            {#if costData.xAxis.length > 0}
              <BarChart 
                xAxisData={costData.xAxis}
                series={costData.series}
                yAxisName="Cost (USD)"
                colorByData={true}
              />
            {:else}
              <div class="flex items-center justify-center h-full text-text-muted" data-testid="empty-chart">No cost data</div>
            {/if}
          {:else if activeTab === 'Daily Token Trend'}
            {#if trendData.xAxis.length > 0}
              <LineChart 
                xAxisData={trendData.xAxis}
                series={trendData.series}
                yAxisName="Tokens"
              />
            {:else}
              <div class="flex items-center justify-center h-full text-text-muted" data-testid="empty-chart">No trend data</div>
            {/if}
          {:else if activeTab === 'Model Token Breakdown'}
            {#if breakdownData.xAxis.length > 0}
              <StackedBarChart 
                xAxisData={breakdownData.xAxis}
                series={breakdownData.series}
                yAxisName="Tokens"
              />
            {:else}
              <div class="flex items-center justify-center h-full text-text-muted" data-testid="empty-chart">No breakdown data</div>
            {/if}
          {/if}
        </div>
      </Panel>
    {/if}
  </div>
</div>
