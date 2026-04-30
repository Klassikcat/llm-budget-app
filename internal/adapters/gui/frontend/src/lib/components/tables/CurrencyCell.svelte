<script lang="ts">
  export let value: number;
  export let currency: string | null | undefined = 'USD';

  $: safeCurrency = currency && currency.trim() ? currency : 'USD';
  $: formattedValue = new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: safeCurrency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2
  }).format(value);
</script>

<span class="font-mono text-sm text-text">{formattedValue}</span>
