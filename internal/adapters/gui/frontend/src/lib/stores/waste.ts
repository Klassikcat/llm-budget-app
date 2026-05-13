import { writable } from 'svelte/store';

import {
  loadAlerts,
  loadInsights,
  loadWasteSummary,
  type AlertListResponse,
  type InsightListResponse,
  type WasteSummaryResponse
} from '$lib/bindings';

export interface WasteStoreData {
  summary: WasteSummaryResponse | null;
  insights: InsightListResponse | null;
  alerts: AlertListResponse | null;
}

export interface WasteStoreState {
  data: WasteStoreData;
  loading: boolean;
  error: string | null;
}

const initialData: WasteStoreData = {
  summary: null,
  insights: null,
  alerts: null
};

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Failed to load waste data';
}

function createWasteStore() {
  const store = writable<WasteStoreState>({
    data: initialData,
    loading: false,
    error: null
  });

  let activeMonth = '';

  async function load(month = activeMonth): Promise<WasteStoreData> {
    activeMonth = month;
    store.update((state) => ({ ...state, loading: true, error: null }));

    try {
      const [summary, insights, alerts] = await Promise.all([
        loadWasteSummary(month),
        loadInsights(month),
        loadAlerts(month)
      ]);
      const data: WasteStoreData = { summary, insights, alerts };
      store.set({ data, loading: false, error: null });
      return data;
    } catch (error) {
      store.update((state) => ({ ...state, loading: false, error: getErrorMessage(error) }));
      throw error;
    }
  }

  return {
    subscribe: store.subscribe,
    load,
    refresh: load
  };
}

export const waste = createWasteStore();
export const loadWaste = waste.load;
export const refreshWaste = waste.refresh;
