<script lang="ts">
  export let onDelete: () => void;
  export let disabled: boolean = false;

  function handleClick(e: MouseEvent) {
    e.stopPropagation();
    if (confirm('Are you sure you want to delete this subscription?')) {
      onDelete();
    }
  }
</script>

<button
  type="button"
  class="text-danger hover:text-danger/80 transition-colors disabled:opacity-50"
  on:click={handleClick}
  {disabled}
  aria-label="Delete"
>
  <svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M3 6h18"></path>
    <path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"></path>
    <path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"></path>
  </svg>
</button>
