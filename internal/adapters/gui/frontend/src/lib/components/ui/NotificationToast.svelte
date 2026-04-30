<script lang="ts">
  import type { GuiNotification } from '$lib/types/notifications';
  import { dismissNotification } from '$lib/services/notification';

  let {
    notification,
    class: className = ''
  }: {
    notification: GuiNotification;
    class?: string;
  } = $props();

  let variantClasses = $derived.by(() => {
    switch (notification.severity) {
      case 'critical':
      case 'danger':
        return 'border-status-danger bg-status-danger/10 text-status-danger';
      case 'warning':
        return 'border-status-warning bg-status-warning/10 text-status-warning';
      case 'success':
        return 'border-status-success bg-status-success/10 text-status-success';
      case 'info':
      default:
        return 'border-status-normal bg-status-normal/10 text-status-normal';
    }
  });

  function handleDismiss() {
    dismissNotification(notification.id);
  }
</script>

<div class="flex w-full min-w-0 items-start gap-md overflow-hidden rounded-compact border bg-panel p-panel-padding shadow-md {variantClasses} {className}" role="alert">
  <div class="flex-shrink-0 mt-0.5">
    {#if notification.severity === 'success'}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    {:else if notification.severity === 'warning'}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path>
      </svg>
    {:else if notification.severity === 'critical' || notification.severity === 'danger'}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    {:else}
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
      </svg>
    {/if}
  </div>

  <div class="min-w-0 flex-1 overflow-hidden">
    <h4 class="m-0 break-words text-sm font-semibold leading-snug">{notification.title}</h4>
    {#if notification.subtitle}
      <p class="mt-0.5 mb-0 break-words text-xs font-medium leading-snug opacity-80">{notification.subtitle}</p>
    {/if}
    <p class="mt-1 mb-0 break-words text-sm leading-snug opacity-90">{notification.body}</p>
  </div>

  <button 
    class="ml-auto flex-shrink-0 rounded p-1 transition-colors hover:bg-background-hover" 
    type="button"
    onclick={handleDismiss}
    aria-label="Dismiss notification"
  >
    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" xmlns="http://www.w3.org/2000/svg">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
    </svg>
  </button>
</div>
