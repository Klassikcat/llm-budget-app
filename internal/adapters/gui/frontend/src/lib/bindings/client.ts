export type BindingName =
  | 'DashboardBinding'
  | 'FormsBinding'
  | 'SubscriptionLookupBinding'
  | 'GraphsBinding'
  | 'InsightsBinding'
  | 'AlertsBinding';

export type WailsBindingClient = {
  invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]): Promise<T>;
};

type BindingMethod = (...args: unknown[]) => unknown;
type WailsGuiBindings = Partial<Record<BindingName, Record<string, BindingMethod>>>;

declare global {
  interface Window {
    go?: {
      gui?: WailsGuiBindings;
    };
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function describeError(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  if (typeof error === 'string') {
    return error;
  }
  return 'unknown binding error';
}

export function getGeneratedBinding(binding: BindingName): Record<string, unknown> {
  if (typeof window === 'undefined') {
    throw new Error('Wails bindings are unavailable outside the browser runtime');
  }

  const module = window.go?.gui?.[binding];
  if (!isRecord(module)) {
    throw new Error(`Wails module ${binding} did not export a method map`);
  }
  return module;
}

export function createWailsBindingClient(): WailsBindingClient {
  return {
    async invoke<T>(binding: BindingName, method: string, ...args: readonly unknown[]) {
      try {
        const module = getGeneratedBinding(binding);
        const candidate = module[method];
        if (typeof candidate !== 'function') {
          throw new Error(`Wails method ${binding}.${method} is unavailable`);
        }
        const value = await (candidate as BindingMethod)(...[...args]);
        return value as T;
      } catch (error) {
        throw new Error(`${binding}.${method} failed: ${describeError(error)}`);
      }
    }
  };
}

let activeClient: WailsBindingClient = createWailsBindingClient();

export function getBindingClient(): WailsBindingClient {
  return activeClient;
}

export function setBindingClient(client: WailsBindingClient): void {
  activeClient = client;
}

export function resetBindingClient(): void {
  activeClient = createWailsBindingClient();
}
