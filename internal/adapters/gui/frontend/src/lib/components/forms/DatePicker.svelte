<script lang="ts">
  let {
    value = $bindable(''),
    id = '',
    name = '',
    disabled = false,
    required = false,
    min = undefined,
    max = undefined,
    error = false,
    onchange = undefined,
    onblur = undefined,
    onfocus = undefined
  }: {
    value?: string;
    id?: string;
    name?: string;
    disabled?: boolean;
    required?: boolean;
    min?: string;
    max?: string;
    error?: boolean;
    onchange?: (e: Event) => void;
    onblur?: (e: FocusEvent) => void;
    onfocus?: (e: FocusEvent) => void;
  } = $props();
</script>

<input
  {id}
  {name}
  type="date"
  bind:value
  {disabled}
  {required}
  {min}
  {max}
  onchange={onchange}
  onblur={onblur}
  onfocus={onfocus}
  class="w-full px-md py-sm bg-background border rounded-compact text-sm text-text focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
  class:border-danger={error}
  class:border-panel-border={!error}
  class:focus:border-primary={!error}
  class:focus:border-danger={error}
/>
