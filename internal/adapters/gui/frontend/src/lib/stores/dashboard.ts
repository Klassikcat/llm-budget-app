import { writable } from 'svelte/store';

import {
  loadDashboard,
  loadGraphs,
  loadWasteSummary,
  type DashboardResponse,
  type GraphResponse,
  type WasteSummaryResponse
} from '$lib/bindings';
import { checkBudgetThresholds } from '$lib/services/notification';

export interface DashboardStoreData {
  dashboard: DashboardResponse | null;
  graphs: GraphResponse | null;
  waste: WasteSummaryResponse | null;
}

export interface DashboardStoreState {
  data: DashboardStoreData;
  loading: boolean;
  error: string | null;
}

const initialData: DashboardStoreData = {
  dashboard: null,
  graphs: null,
  waste: null
};

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Failed to load dashboard data';
}

function createDashboardStore() {
  const store = writable<DashboardStoreState>({
    data: initialData,
    loading: false,
    error: null
  });

  let activeMonth = '';

  async function load(month = activeMonth): Promise<DashboardStoreData> {
    activeMonth = month;
    store.update((state) => ({ ...state, loading: true, error: null }));

    try {
      const [dashboard, graphs, waste] = await Promise.all([
        loadDashboard(month),
        loadGraphs(month),
        loadWasteSummary(month)
      ]);
      const data: DashboardStoreData = { dashboard, graphs, waste };
      store.set({ data, loading: false, error: null });
      checkBudgetThresholds(dashboard.budgets, dashboard.period.month);
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

export const dashboardStore = createDashboardStore();
export const loadDashboardData = dashboardStore.load;
export const refreshDashboardData = dashboardStore.refresh;
export const invalidateDashboardData = dashboardStore.refresh;
