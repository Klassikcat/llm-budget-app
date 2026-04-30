import { writable } from 'svelte/store';

import {
  loadDashboard,
  loadGraphs,
  saveManualEntry,
  type DashboardProviderSummary,
  type DashboardRecentSession,
  type DashboardResponse,
  type GraphResponse
} from '$lib/bindings';
import { invalidateDashboardData } from './dashboard';
import type { ManualEntryInput, ManualEntryMutationResponse } from '$lib/types/forms';

export interface UsageStoreData {
  dashboard: DashboardResponse | null;
  graphs: GraphResponse | null;
  providerSummaries: DashboardProviderSummary[];
  recentSessions: DashboardRecentSession[];
}

export interface UsageStoreState {
  data: UsageStoreData;
  loading: boolean;
  error: string | null;
}

const initialData: UsageStoreData = {
  dashboard: null,
  graphs: null,
  providerSummaries: [],
  recentSessions: []
};

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Failed to load usage data';
}

function createUsageStore() {
  const store = writable<UsageStoreState>({
    data: initialData,
    loading: false,
    error: null
  });

  let activeMonth = '';

  async function load(month = activeMonth): Promise<UsageStoreData> {
    activeMonth = month;
    store.update((state) => ({ ...state, loading: true, error: null }));

    try {
      const [dashboard, graphs] = await Promise.all([loadDashboard(month), loadGraphs(month)]);
      const data: UsageStoreData = {
        dashboard,
        graphs,
        providerSummaries: dashboard.providerSummaries,
        recentSessions: dashboard.recentSessions
      };
      store.set({ data, loading: false, error: null });
      return data;
    } catch (error) {
      store.update((state) => ({ ...state, loading: false, error: getErrorMessage(error) }));
      throw error;
    }
  }

  async function save(input: ManualEntryInput): Promise<ManualEntryMutationResponse> {
    const response = await saveManualEntry(input);
    await Promise.all([load(activeMonth), invalidateDashboardData(activeMonth)]);
    return response;
  }

  return {
    subscribe: store.subscribe,
    load,
    refresh: load,
    save
  };
}

export const usage = createUsageStore();
export const loadUsage = usage.load;
export const refreshUsage = usage.refresh;
export const saveManualEntryAndRefresh = usage.save;
