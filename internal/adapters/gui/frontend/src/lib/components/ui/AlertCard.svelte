<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    title,
    message = '',
    variant = 'info',
    class: className = '',
    action
  }: {
    title: string;
    message?: string;
    variant?: 'info' | 'success' | 'warning' | 'danger';
    class?: string;
    action?: Snippet;
  } = $props();

  let variantClasses = $derived.by(() => {
    switch (variant) {
      case 'success':
        return 'border-status-success bg-status-success/10 text-status-success';
      case 'warning':
        return 'border-status-warning bg-status-warning/10 text-status-warning';
      case 'danger':
        return 'border-status-danger bg-status-danger/10 text-status-danger';
      case 'info':
      default:
        return 'border-status-normal bg-status-normal/10 text-status-normal';
    }
  });
</script>

<div class="flex items-start gap-md p-panel-padding border rounded-compact {variantClasses} {className}">
  <div class="flex-shrink-0 mt-0.5">
    {#if variant === 'success'}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    {:else if variant === 'warning'}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
    {:else if variant === 'danger'}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    {:else}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    {/if}
  </div>
  
  <div class="flex-1 min-w-0">
    <h4 class="text-sm font-semibold m-0">{title}</h4>
    {#if message}
      <p class="text-sm mt-1 mb-0 opacity-90">{message}</p>
    {/if}
  </div>
  
  {#if action}
    <div class="flex-shrink-0 ml-auto">
      {@render action()}
    </div>
  {/if}
</div>
