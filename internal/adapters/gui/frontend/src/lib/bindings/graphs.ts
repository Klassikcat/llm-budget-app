import { getBindingClient } from './client';

export interface ModelTokenUsage {
  modelName: string;
  totalTokens: number;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
}

export interface ModelCost {
  modelName: string;
  totalCostUsd: number;
}

export interface ModelDailyTokens {
  modelName: string;
  totalTokens: number;
}

export interface DailyTokenTrend {
  date: string;
  modelBreakdown: ModelDailyTokens[];
}

export interface ModelTokenBreakdown {
  modelName: string;
  inputTokens: number;
  outputTokens: number;
  cacheReadTokens: number;
  cacheWriteTokens: number;
  totalTokens: number;
}

export const graphTimeRanges = ['7 days', '30 days', 'All'] as const;

export type GraphTimeRange = (typeof graphTimeRanges)[number];

export interface GraphResponse {
  modelTokenUsages: ModelTokenUsage[];
  modelCosts: ModelCost[];
  dailyTokenTrends: DailyTokenTrend[];
  modelTokenBreakdowns: ModelTokenBreakdown[];
}

export async function loadGraphs(month = '', timeRange: GraphTimeRange = 'All'): Promise<GraphResponse> {
  return getBindingClient().invoke<GraphResponse>('GraphsBinding', 'LoadGraphs', month, timeRange);
}
