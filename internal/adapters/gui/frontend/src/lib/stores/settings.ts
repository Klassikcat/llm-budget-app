import { writable } from 'svelte/store';

import { loadSettings, saveSettings, type SettingsFormInput, type SettingsFormState } from '$lib/bindings';

export interface SettingsStoreState {
  data: SettingsFormState | null;
  loading: boolean;
  saving: boolean;
  error: string | null;
}

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Failed to load settings';
}

function createSettingsStore() {
  const store = writable<SettingsStoreState>({
    data: null,
    loading: false,
    saving: false,
    error: null
  });

  async function load(): Promise<SettingsFormState> {
    store.update((state) => ({ ...state, loading: true, error: null }));

    try {
      const response = await loadSettings();
      store.set({ data: response.settings, loading: false, saving: false, error: null });
      return response.settings;
    } catch (error) {
      store.update((state) => ({ ...state, loading: false, error: getErrorMessage(error) }));
      throw error;
    }
  }

  async function save(input: SettingsFormInput): Promise<SettingsFormState> {
    store.update((state) => ({ ...state, saving: true, error: null }));

    try {
      await saveSettings(input);
      return await load();
    } catch (error) {
      store.update((state) => ({ ...state, saving: false, error: getErrorMessage(error) }));
      throw error;
    }
  }

  return {
    subscribe: store.subscribe,
    load,
    refresh: load,
    save
  };
}

export const settings = createSettingsStore();
export const loadSettingsState = settings.load;
export const refreshSettingsState = settings.refresh;
export const saveSettingsAndRefresh = settings.save;
