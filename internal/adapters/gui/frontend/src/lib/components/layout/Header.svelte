<script lang="ts">
  import { page } from '$app/stores';
  import { RefreshCw } from 'lucide-svelte';

  const routeTitles: Record<string, string> = {
    '/': 'Dashboard',
    '/usage': 'Usage',
    '/subscriptions': 'Subscriptions',
    '/budgets': 'Budgets',
    '/insights': 'Insights',
    '/graphs': 'Graphs',
    '/settings': 'Settings'
  };

  $: currentTitle = routeTitles[$page.url.pathname] || 'LLM Budget Tracker';
</script>

<header class="h-16 flex items-center justify-between px-6 bg-background border-b border-panel-border">
  <h1 class="text-xl font-semibold text-text m-0">{currentTitle}</h1>
  
  <div class="flex items-center space-x-4">
    <button
      class="p-2 rounded-md text-text-muted hover:text-text hover:bg-background-hover transition-colors flex items-center"
      aria-label="Refresh data"
    >
      <RefreshCw size={18} />
    </button>
  </div>
</header>
