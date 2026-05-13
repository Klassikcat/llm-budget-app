<script lang="ts">
  export let value: number;
  
  $: formattedValue = new Intl.NumberFormat('en-US', {
    notation: 'compact',
    compactDisplay: 'short',
    maximumFractionDigits: 1
  }).format(value);
</script>

<span class="font-mono text-sm text-text">{formattedValue}</span>
