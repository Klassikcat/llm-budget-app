<script lang="ts">
  import { notificationStore } from '$lib/stores/notification';
  import { clearAllNotifications, markNotificationsAsRead } from '$lib/services/notification';
  import NotificationToast from './NotificationToast.svelte';

  let {
    class: className = ''
  }: {
    class?: string;
  } = $props();

  let notifications = $derived($notificationStore.items);
  let unreadCount = $derived($notificationStore.unreadCount);

  function handleClearAll() {
    clearAllNotifications();
  }

  $effect(() => {
    if (unreadCount > 0) {
      markNotificationsAsRead();
    }
  });
</script>

{#if notifications.length > 0}
  <section
    class="fixed right-4 top-4 z-[80] flex max-h-[calc(100vh-2rem)] w-[min(calc(100vw-2rem),28rem)] flex-col overflow-hidden rounded-lg border border-panel-border bg-panel shadow-lg pointer-events-auto sm:right-6 sm:top-6 {className}"
    aria-label="Notifications"
  >
    <div class="flex items-center justify-between gap-md border-b border-panel-border bg-card p-panel-padding">
      <h3 class="m-0 min-w-0 truncate text-md font-semibold text-text">Notifications</h3>
      <button 
        class="shrink-0 text-xs text-muted hover:text-text transition-colors"
        type="button"
        onclick={handleClearAll}
      >
        Clear All
      </button>
    </div>

    <div class="flex min-h-0 flex-col gap-sm overflow-y-auto p-panel-padding">
      {#each notifications as notification (notification.id)}
        <NotificationToast {notification} />
      {/each}
    </div>
  </section>
{/if}
