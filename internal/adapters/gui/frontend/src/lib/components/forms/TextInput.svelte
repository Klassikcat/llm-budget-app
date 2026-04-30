<script lang="ts">
  let {
    value = $bindable(''),
    id = '',
    name = '',
    placeholder = '',
    disabled = false,
    required = false,
    type = 'text',
    error = false,
    onchange = undefined,
    oninput = undefined,
    onblur = undefined,
    onfocus = undefined
  }: {
    value?: string;
    id?: string;
    name?: string;
    placeholder?: string;
    disabled?: boolean;
    required?: boolean;
    type?: 'text' | 'password' | 'email' | 'search' | 'tel' | 'url';
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
  {type}
  bind:value
  {placeholder}
  {disabled}
  {required}
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
