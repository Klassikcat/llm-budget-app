<script lang="ts">
  export let value: Date | string | number;
  export let format: 'short' | 'long' = 'short';

  function parseDateValue(input: Date | string | number): Date {
    if (typeof input === 'string') {
      const dateOnly = /^(\d{4})-(\d{2})-(\d{2})$/.exec(input);

      if (dateOnly) {
        const [, year, month, day] = dateOnly;

        return new Date(Number(year), Number(month) - 1, Number(day));
      }
    }

    return new Date(input);
  }
  
  $: date = parseDateValue(value);
  
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
