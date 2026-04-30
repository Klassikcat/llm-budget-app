<script lang="ts">
  import { onMount } from 'svelte';
  import '../lib/styles/tailwind.css';
  import Sidebar from '$lib/components/layout/Sidebar.svelte';
  import Header from '$lib/components/layout/Header.svelte';
  import NotificationCenter from '$lib/components/ui/NotificationCenter.svelte';
  import { wireWailsNotifications } from '$lib/services/wailsRuntime';

  let collapsed = false;

  onMount(() => {
    let cleanup: (() => void) | null = null;
    void wireWailsNotifications().then((dispose) => {
      cleanup = dispose;
    });

    return () => {
      cleanup?.();
    };
  });
</script>

<div class="flex h-screen w-full bg-background text-text overflow-hidden">
  <Sidebar bind:collapsed />
  
  <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
    <Header />
    
    <main class="flex-1 overflow-y-auto bg-background">
      <slot />
    </main>

    <NotificationCenter />
  </div>
</div>
