<script lang="ts">
  import { page } from '$app/stores';
  import {
    LayoutDashboard,
    Terminal,
    CreditCard,
    Wallet,
    Lightbulb,
    LineChart,
    Settings,
    ChevronLeft,
    ChevronRight,
    Sun,
    Moon
  } from 'lucide-svelte';
  import { resolvedTheme, theme, toggleTheme } from '$lib/stores/theme';

  export let collapsed = false;

  const navItems = [
    { href: '/', label: 'Dashboard', icon: LayoutDashboard },
    { href: '/usage', label: 'Usage', icon: Terminal },
    { href: '/subscriptions', label: 'Subscriptions', icon: CreditCard },
    { href: '/budgets', label: 'Budgets', icon: Wallet },
    { href: '/insights', label: 'Insights', icon: Lightbulb },
    { href: '/graphs', label: 'Graphs', icon: LineChart },
    { href: '/settings', label: 'Settings', icon: Settings }
  ];

  function toggleCollapse() {
    collapsed = !collapsed;
  }
</script>

<aside
  class="flex flex-col h-screen bg-card border-r border-panel-border transition-all duration-300 ease-in-out relative {collapsed ? 'w-16' : 'w-60'}"
  aria-label="Sidebar Navigation"
>
  <div class="flex items-center justify-between h-16 px-4 border-b border-panel-border">
    {#if !collapsed}
      <span class="font-bold text-lg text-text truncate">LLM Budget</span>
    {/if}
    <button
      class="p-1 rounded-md text-text-muted hover:text-text hover:bg-background-hover transition-colors"
      on:click={toggleCollapse}
      aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      aria-expanded={!collapsed}
    >
      {#if collapsed}
        <ChevronRight size={20} />
      {:else}
        <ChevronLeft size={20} />
      {/if}
    </button>
  </div>

  <nav class="flex-1 overflow-y-auto py-4 px-2 space-y-1">
    {#each navItems as item}
      {@const active = $page.url.pathname === item.href}
      <a
        href={item.href}
        class="flex items-center px-2 py-2 rounded-md transition-colors group {active ? 'bg-background-active text-primary font-medium' : 'text-text-muted hover:bg-background-hover hover:text-text'}"
        aria-current={active ? 'page' : undefined}
        title={collapsed ? item.label : undefined}
      >
        <svelte:component this={item.icon} size={20} class="flex-shrink-0 {active ? 'text-primary' : 'text-text-muted group-hover:text-text'}" />
        {#if !collapsed}
          <span class="ml-3 truncate">{item.label}</span>
        {/if}
      </a>
    {/each}
  </nav>

  <div class="p-4 border-t border-panel-border">
    <button
      class="flex items-center w-full px-2 py-2 rounded-md text-text-muted hover:text-text hover:bg-background-hover transition-colors"
      on:click={toggleTheme}
      aria-label="Toggle theme"
      title={collapsed ? 'Toggle theme' : undefined}
    >
      {#if $resolvedTheme === 'dark'}
        <Sun size={20} class="flex-shrink-0" />
      {:else}
        <Moon size={20} class="flex-shrink-0" />
      {/if}
      {#if !collapsed}
        <span class="ml-3 truncate">
          {#if $theme === 'system'}
            System ({$resolvedTheme})
          {:else}
            {$resolvedTheme === 'dark' ? 'Light Mode' : 'Dark Mode'}
          {/if}
        </span>
      {/if}
    </button>
  </div>
</aside>
