<script module lang="ts">
  export interface SelectOption {
    value: string;
    label: string;
    disabled?: boolean;
  }
</script>

<script lang="ts">
  let {
    value = $bindable(''),
    options = [],
    id = '',
    name = '',
    disabled = false,
    required = false,
    error = false,
    onchange = undefined,
    onblur = undefined,
    onfocus = undefined,
    testId = undefined
  }: {
    value?: string;
    options?: SelectOption[];
    id?: string;
    name?: string;
    disabled?: boolean;
    required?: boolean;
    error?: boolean;
    onchange?: (e: Event) => void;
    onblur?: (e: FocusEvent) => void;
    onfocus?: (e: FocusEvent) => void;
    testId?: string;
  } = $props();
</script>

<div class="relative w-full">
  <select
    {id}
    {name}
    bind:value
    {disabled}
    {required}
    onchange={onchange}
    onblur={onblur}
    onfocus={onfocus}
    data-testid={testId}
    class="w-full px-md py-sm pr-8 bg-background border rounded-compact text-sm text-text appearance-none focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
    class:border-danger={error}
    class:border-panel-border={!error}
    class:focus:border-primary={!error}
    class:focus:border-danger={error}
  >
    {#if !required && !value}
      <option value="" disabled selected hidden></option>
    {/if}
    {#each options as option}
      <option value={option.value} disabled={option.disabled}>
        {option.label}
      </option>
    {/each}
  </select>
  <div class="absolute inset-y-0 right-0 flex items-center px-2 pointer-events-none text-text-muted">
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
    </svg>
  </div>
</div>
