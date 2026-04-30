import type { DetectorCategory, AlertSeverity } from '$lib/types/domain';

import type { DashboardPeriod } from './dashboard';
import { getBindingClient } from './client';

export type InsightSeverity = 'low' | 'medium' | 'high' | string;

export interface WasteByDetector {
  category: DetectorCategory | string;
  attributedCostUsd: number;
  insightCount: number;
}

export interface WasteTrendPoint {
  day: string;
  wasteCostUsd: number;
}

export interface WasteSummaryResponse {
  period: DashboardPeriod;
  totalWasteCostUsd: number;
  totalSpendCostUsd: number;
  wastePercent: number;
  weeklyWasteCostUsd: number;
  monthlyWasteCostUsd: number;
  projectedMonthEndWasteUsd: number;
  byDetector: WasteByDetector[];
  topCauses: WasteByDetector[];
  dailyTrend: WasteTrendPoint[];
  generatedAt: string;
}

export interface InsightHash {
  kind: string;
  value: string;
}

export interface InsightCount {
  key: string;
  value: number;
}

export interface InsightMetric {
  key: string;
  unit: string;
  value: number;
}

export interface InsightPayload {
  sessionIds: string[];
  usageEntryIds: string[];
  hashes: InsightHash[];
  counts: InsightCount[];
  metrics: InsightMetric[];
}

export interface InsightState {
  insightId: string;
  category: DetectorCategory | string;
  severity: InsightSeverity | AlertSeverity;
  detectedAt: string;
  period: DashboardPeriod;
  payload: InsightPayload;
}

export interface InsightListResponse {
  items: InsightState[];
  empty: boolean;
}

export async function loadWasteSummary(month = ''): Promise<WasteSummaryResponse> {
  return getBindingClient().invoke<WasteSummaryResponse>('InsightsBinding', 'LoadWasteSummary', month);
}

export async function loadInsights(month = ''): Promise<InsightListResponse> {
  return getBindingClient().invoke<InsightListResponse>('InsightsBinding', 'LoadInsights', month);
}
