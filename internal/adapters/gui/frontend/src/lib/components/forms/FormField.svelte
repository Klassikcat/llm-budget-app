<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    id = '',
    label = '',
    error = '',
    helpText = '',
    required = false,
    children
  }: {
    id?: string;
    label?: string;
    error?: string;
    helpText?: string;
    required?: boolean;
    children?: Snippet;
  } = $props();
</script>

<div class="flex flex-col gap-xs mb-md">
  {#if label}
    <label for={id} class="text-sm font-medium text-text">
      {label}
      {#if required}
        <span class="text-danger ml-1" aria-hidden="true">*</span>
      {/if}
    </label>
  {/if}
  
  <div class="relative">
    {@render children?.()}
  </div>
  
  {#if error}
    <p class="text-xs text-danger mt-1" role="alert">{error}</p>
  {:else if helpText}
    <p class="text-xs text-text-muted mt-1">{helpText}</p>
  {/if}
</div>
