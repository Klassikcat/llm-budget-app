<script lang="ts">
  let {
    checked = $bindable(false),
    id = '',
    name = '',
    disabled = false,
    onchange = undefined
  }: {
    checked?: boolean;
    id?: string;
    name?: string;
    disabled?: boolean;
    onchange?: (e: Event) => void;
  } = $props();
</script>

<label class="relative inline-flex items-center cursor-pointer" class:opacity-50={disabled} class:cursor-not-allowed={disabled}>
  <input
    type="checkbox"
    {id}
    {name}
    bind:checked
    {disabled}
    onchange={onchange}
    class="sr-only peer"
  />
  <div class="w-11 h-6 bg-background-active peer-focus:outline-none peer-focus:ring-2 peer-focus:ring-primary/50 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-card after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-card after:border-panel-border after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-primary border border-panel-border"></div>
</label>
