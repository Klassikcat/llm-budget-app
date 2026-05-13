import { writable } from 'svelte/store';

import { loadDashboard, saveBudget, type DashboardBudget, type DashboardResponse } from '$lib/bindings';
import { invalidateDashboardData } from './dashboard';
import type { BudgetInput, BudgetMutationResponse } from '$lib/types/forms';

export interface BudgetStoreData {
  dashboard: DashboardResponse | null;
  budgets: DashboardBudget[];
}

export interface BudgetStoreState {
  data: BudgetStoreData;
  loading: boolean;
  error: string | null;
}

const initialData: BudgetStoreData = {
  dashboard: null,
  budgets: []
};

function getErrorMessage(error: unknown): string {
  return error instanceof Error ? error.message : 'Failed to load budget data';
}

function createBudgetStore() {
  const store = writable<BudgetStoreState>({
    data: initialData,
    loading: false,
    error: null
  });

  let activeMonth = '';

  async function load(month = activeMonth): Promise<BudgetStoreData> {
    activeMonth = month;
    store.update((state) => ({ ...state, loading: true, error: null }));

    try {
      const dashboard = await loadDashboard(month);
      const data: BudgetStoreData = {
        dashboard,
        budgets: dashboard.budgets
      };
      store.set({ data, loading: false, error: null });
      return data;
    } catch (error) {
      store.update((state) => ({ ...state, loading: false, error: getErrorMessage(error) }));
      throw error;
    }
  }

  async function save(input: BudgetInput): Promise<BudgetMutationResponse> {
    const response = await saveBudget(input);
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

export const budget = createBudgetStore();
export const loadBudget = budget.load;
export const refreshBudget = budget.refresh;
export const saveBudgetAndRefresh = budget.save;
