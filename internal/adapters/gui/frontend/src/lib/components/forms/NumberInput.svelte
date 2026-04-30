<script lang="ts">
  let {
    value = $bindable(null),
    id = '',
    name = '',
    placeholder = '',
    disabled = false,
    required = false,
    min = undefined,
    max = undefined,
    step = undefined,
    error = false,
    onchange = undefined,
    oninput = undefined,
    onblur = undefined,
    onfocus = undefined
  }: {
    value?: number | null;
    id?: string;
    name?: string;
    placeholder?: string;
    disabled?: boolean;
    required?: boolean;
    min?: number;
    max?: number;
    step?: number | string;
    error?: boolean;
    onchange?: (e: Event) => void;
    oninput?: (e: Event) => void;
    onblur?: (e: FocusEvent) => void;
    onfocus?: (e: FocusEvent) => void;
  } = $props();
</script>

<input
  {id}
  {name}
  type="number"
  bind:value
  {placeholder}
  {disabled}
  {required}
  {min}
  {max}
  {step}
  onchange={onchange}
  oninput={oninput}
  onblur={onblur}
  onfocus={onfocus}
  class="w-full px-md py-sm bg-background border rounded-compact text-sm text-text placeholder:text-text-muted focus:outline-none focus:ring-2 focus:ring-primary/50 transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
  class:border-danger={error}
  class:border-panel-border={!error}
  class:focus:border-primary={!error}
  class:focus:border-danger={error}
/>
