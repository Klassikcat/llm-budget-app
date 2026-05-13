<script lang="ts">
  let {
    label,
    value,
    trend = 'none',
    trendValue = '',
    class: className = ''
  }: {
    label: string;
    value: string | number;
    trend?: 'up' | 'down' | 'none';
    trendValue?: string;
    class?: string;
  } = $props();
</script>

<div class="flex flex-col bg-card border border-panel-border rounded-compact p-panel-padding shadow-sm {className}">
  <div class="text-sm font-medium text-text-muted mb-xs">{label}</div>
  <div class="flex items-baseline justify-between">
    <div class="text-xl font-bold text-text">{value}</div>
    
    {#if trend !== 'none'}
      <div class="flex items-center text-xs font-medium" 
           class:text-status-success={trend === 'up'} 
           class:text-status-danger={trend === 'down'}>
        {#if trend === 'up'}
          <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-label="Trend up">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 10l7-7m0 0l7 7m-7-7v18"></path>
          </svg>
        {:else if trend === 'down'}
          <svg class="w-3 h-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg" aria-label="Trend down">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3"></path>
          </svg>
        {/if}
        {#if trendValue}
          <span>{trendValue}</span>
        {/if}
      </div>
    {/if}
  </div>
</div>
