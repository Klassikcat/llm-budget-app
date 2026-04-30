import { writable, get } from 'svelte/store';
import type { GuiNotification, NotificationState } from '$lib/types/notifications';

export interface NotificationStoreState extends NotificationState {
  sentAlertKeys: Set<string>;
}

const initialState: NotificationStoreState = {
  items: [],
  unreadCount: 0,
  lastDispatchedAt: null,
  permission: 'default',
  sentAlertKeys: new Set<string>()
};

function createNotificationStore() {
  const store = writable<NotificationStoreState>(initialState);
  const { subscribe, set, update } = store;

  return {
    subscribe,
    addNotification: (notification: GuiNotification) => {
      update((state) => {
        const items = [notification, ...state.items];
        return {
          ...state,
          items,
          unreadCount: state.unreadCount + 1,
          lastDispatchedAt: new Date().toISOString()
        };
      });
    },
    dismissNotification: (id: string) => {
      update((state) => {
        const items = state.items.filter((n) => n.id !== id);
        return {
          ...state,
          items
        };
      });
    },
    markAsRead: () => {
      update((state) => ({
        ...state,
        unreadCount: 0
      }));
    },
    clearAll: () => {
      update((state) => ({
        ...state,
        items: [],
        unreadCount: 0
      }));
    },
    addSentAlertKey: (key: string) => {
      update((state) => {
        const newSet = new Set(state.sentAlertKeys);
        newSet.add(key);
        return {
          ...state,
          sentAlertKeys: newSet
        };
      });
    },
    hasSentAlertKey: (key: string): boolean => {
      return get(store).sentAlertKeys.has(key);
    },
    reset: () => set(initialState)
  };
}

export const notificationStore = createNotificationStore();
