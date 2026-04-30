<script lang="ts">
  export let status: string;
  
  $: normalizedStatus = status.toLowerCase();
  
  $: colorClass = (() => {
    if (normalizedStatus.includes('expired') || normalizedStatus.includes('inactive') || normalizedStatus.includes('disabled')) {
      return 'bg-status-inactive/20 text-status-inactive border-status-inactive/30';
    }
    if (normalizedStatus.includes('active') || normalizedStatus.includes('success') || normalizedStatus.includes('ok')) {
      return 'bg-status-success/20 text-status-success border-status-success/30';
    }
    if (normalizedStatus.includes('over') || normalizedStatus.includes('danger') || normalizedStatus.includes('error') || normalizedStatus.includes('failed')) {
      return 'bg-status-danger/20 text-status-danger border-status-danger/30';
    }
    if (normalizedStatus.includes('warning') || normalizedStatus.includes('pending')) {
      return 'bg-status-warning/20 text-status-warning border-status-warning/30';
    }
    return 'bg-status-normal/20 text-status-normal border-status-normal/30';
  })();
</script>

<span class="inline-flex items-center px-2 py-0.5 rounded-compact text-xs font-medium border {colorClass}">
  {status}
</span>
