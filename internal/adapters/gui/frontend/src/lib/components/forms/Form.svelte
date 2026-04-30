<script module lang="ts">
  export interface ValidationResult<T = unknown> {
    success: boolean;
    data?: T;
    error?: {
      issues: Array<{ path: (string | number)[]; message: string }>;
    };
  }

  export interface Schema<T = unknown> {
    safeParse: (data: unknown) => ValidationResult<T>;
  }
</script>

<script lang="ts">
  import type { Snippet } from 'svelte';

  let {
    schema = undefined,
    data = undefined,
    onvalidate = undefined,
    onsubmit = undefined,
    class: className = '',
    children
  }: {
    schema?: Schema;
    data?: unknown;
    onvalidate?: (result: ValidationResult) => void;
    onsubmit?: (e: SubmitEvent) => void;
    class?: string;
    children?: Snippet;
  } = $props();

  function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    
    if (schema && data !== undefined) {
      const result = schema.safeParse(data);
      if (onvalidate) {
        onvalidate(result);
      }
      if (!result.success) {
        return;
      }
    }

    if (onsubmit) {
      onsubmit(e);
    }
  }
</script>

<form onsubmit={handleSubmit} class={className}>
  {@render children?.()}
</form>
