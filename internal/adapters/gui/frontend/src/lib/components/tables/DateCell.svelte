<script lang="ts">
  export let value: Date | string | number;
  export let format: 'short' | 'long' = 'short';
  
  $: date = new Date(value);
  
  $: formattedValue = (() => {
    if (isNaN(date.getTime())) return 'Invalid Date';
    
    if (format === 'long') {
      return new Intl.DateTimeFormat('en-US', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit'
      }).format(date);
    }
    
    return new Intl.DateTimeFormat('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    }).format(date);
  })();
</script>

<span class="text-sm text-text-muted">{formattedValue}</span>
