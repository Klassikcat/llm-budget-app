<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title = '',
    class: className = '',
    action,
    children
  }: {
    title?: string;
    class?: string;
    action?: Snippet;
    children?: Snippet;
  } = $props();
</script>

<div class="flex flex-col bg-card border border-panel-border rounded-compact shadow-sm {className}">
  {#if title || action}
    <div class="flex items-center justify-between px-panel-padding py-sm border-b border-panel-border bg-background-hover rounded-t-compact">
      {#if title}
        <h3 class="text-sm font-semibold text-text m-0">{title}</h3>
      {:else}
        <div></div>
      {/if}
      
      {#if action}
        <div class="flex items-center gap-sm">
          {@render action()}
        </div>
      {/if}
    </div>
  {/if}
  
  <div class="p-panel-padding flex-1">
    {@render children?.()}
  </div>
</div>
