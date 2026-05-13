<script lang="ts">
  let {
    label,
    value,
    data = [],
    color = 'var(--color-primary)',
    class: className = ''
  }: {
    label: string;
    value: string | number;
    data?: number[];
    color?: string;
    class?: string;
  } = $props();

  // Calculate SVG path for sparkline
  let pathD = $derived.by(() => {
    if (!data || data.length === 0) return '';
    
    const min = Math.min(...data);
    const max = Math.max(...data);
    const range = max - min || 1; // Prevent division by zero
    
    const width = 100;
    const height = 30;
    
    const points = data.map((val, i) => {
      const x = (i / (data.length - 1)) * width;
      const y = height - ((val - min) / range) * height;
      return `${x},${y}`;
    });
    
    return `M ${points.join(' L ')}`;
  });
</script>

<div class="flex flex-col bg-card border border-panel-border rounded-compact p-panel-padding shadow-sm {className}">
  <div class="flex justify-between items-start mb-sm">
    <div>
      <div class="text-sm font-medium text-text-muted mb-xs">{label}</div>
      <div class="text-lg font-bold text-text">{value}</div>
    </div>
  </div>
  
  <div class="h-8 w-full mt-auto">
    {#if data && data.length > 1}
      <svg class="w-full h-full overflow-visible" preserveAspectRatio="none" viewBox="0 0 100 30">
        <path 
          d={pathD} 
          fill="none" 
          stroke={color} 
          stroke-width="2" 
          stroke-linecap="round" 
          stroke-linejoin="round" 
          vector-effect="non-scaling-stroke"
        />
      </svg>
    {:else}
      <div class="w-full h-full flex items-center justify-center text-xs text-text-muted">
        No data
      </div>
    {/if}
  </div>
</div>
