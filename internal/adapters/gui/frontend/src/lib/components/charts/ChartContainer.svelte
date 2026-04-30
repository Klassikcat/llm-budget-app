<script lang="ts">
  export let title: string = '';
  export let loading: boolean = false;
  export let error: string | null = null;
  export let empty: boolean = false;
</script>

<div class="flex flex-col w-full h-full bg-card border border-panel-border rounded-compact p-panel-padding">
  {#if title}
    <h3 class="text-sm font-semibold text-text mb-md">{title}</h3>
  {/if}

  <div class="relative flex-1 w-full min-h-[200px]">
    {#if loading}
      <div class="absolute inset-0 flex items-center justify-center bg-card/50 z-10">
        <span class="text-text-muted text-sm">Loading...</span>
      </div>
    {/if}

    {#if error}
      <div class="absolute inset-0 flex items-center justify-center bg-card z-10">
        <span class="text-danger text-sm">{error}</span>
      </div>
    {:else if empty}
      <div class="absolute inset-0 flex items-center justify-center bg-card z-10">
        <span class="text-text-muted text-sm">No data available</span>
      </div>
    {:else}
      <div class="absolute inset-0" class:opacity-50={loading}>
        <slot />
      </div>
    {/if}
  </div>
</div>
