type RuntimeCall = {
  name: string;
  payload: readonly unknown[];
};

type RuntimeCallback = (...payload: readonly unknown[]) => void;

export type WailsRuntimeMock = {
  calls: RuntimeCall[];
  events: Map<string, Set<RuntimeCallback>>;
  WindowReload: () => void;
  WindowSetTitle: (title: string) => void;
  EventsOn: (eventName: string, callback: RuntimeCallback) => () => void;
  EventsEmit: (eventName: string, ...payload: readonly unknown[]) => void;
  reset: () => void;
};

export function createWailsRuntimeMock(): WailsRuntimeMock {
  const calls: RuntimeCall[] = [];
  const events = new Map<string, Set<RuntimeCallback>>();

  const record = (name: string, payload: readonly unknown[] = []) => {
    calls.push({ name, payload });
  };

  return {
    calls,
    events,
    WindowReload: () => {
      record('WindowReload');
    },
    WindowSetTitle: (title: string) => {
      record('WindowSetTitle', [title]);
    },
    EventsOn: (eventName: string, callback: RuntimeCallback) => {
      const callbacks = events.get(eventName) ?? new Set<RuntimeCallback>();
      callbacks.add(callback);
      events.set(eventName, callbacks);

      return () => {
        callbacks.delete(callback);
        if (callbacks.size === 0) {
          events.delete(eventName);
        }
      };
    },
    EventsEmit: (eventName: string, ...payload: readonly unknown[]) => {
      record('EventsEmit', [eventName, ...payload]);
      events.get(eventName)?.forEach((callback) => callback(...payload));
    },
    reset: () => {
      calls.length = 0;
      events.clear();
    }
  };
}

export const wailsRuntimeMock = createWailsRuntimeMock();
