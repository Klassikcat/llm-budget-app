import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/svelte';
import Page from './+page.svelte';
import { waste, type WasteStoreState } from '$lib/stores/waste';

vi.mock('$lib/stores/waste', () => ({
  waste: {
    subscribe: vi.fn(),
    load: vi.fn(),
    refresh: vi.fn()
  },
  loadWaste: vi.fn(),
  refreshWaste: vi.fn()
}));

vi.mock('$lib/components/charts/echartsAction', () => ({
  chartAction: () => ({
    destroy: vi.fn(),
    update: vi.fn()
  })
}));

function createInsight(index: number) {
  return {
    insightId: `ins-${index}`,
    category: `custom_detector_${index}`,
    severity: 'medium',
    detectedAt: `2026-04-${String(index).padStart(2, '0')}T10:00:00Z`,
    period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
    payload: {
      sessionIds: [],
      usageEntryIds: [],
      hashes: [],
      counts: [],
      metrics: []
    }
  };
}

describe('Insights Page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders empty state when no data is available', () => {
    const mockStore: WasteStoreState = {
      data: {
        summary: null,
        insights: null,
        alerts: null
      },
      loading: false,
      error: null
    };
    
    vi.mocked(waste.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('Insights')).toBeInTheDocument();
    expect(screen.getByText('Waste detection and optimization opportunities')).toBeInTheDocument();
    expect(screen.queryByText('Waste Headline')).not.toBeInTheDocument();
  });

  it('renders insights dashboard with data', () => {
    const mockStore: WasteStoreState = {
      data: {
        summary: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totalWasteCostUsd: 15.5,
          totalSpendCostUsd: 100,
          wastePercent: 15.5,
          weeklyWasteCostUsd: 5,
          monthlyWasteCostUsd: 15.5,
          projectedMonthEndWasteUsd: 20.5,
          byDetector: [],
          topCauses: [
            { category: 'context_avalanche', attributedCostUsd: 10.5, insightCount: 5 }
          ],
          dailyTrend: [
            { day: '2026-04-01T00:00:00Z', wasteCostUsd: 1.5 }
          ],
          generatedAt: '2026-04-30T12:00:00Z'
        },
        insights: {
          items: [
            {
              insightId: 'ins-1',
              category: 'context_avalanche',
              severity: 'high',
              detectedAt: '2026-04-30T10:00:00Z',
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              payload: {
                sessionIds: ['sess-1'],
                usageEntryIds: [],
                hashes: [],
                counts: [],
                metrics: []
              }
            }
          ],
          empty: false
        },
        alerts: null
      },
      loading: false,
      error: null
    };
    
    vi.mocked(waste.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('Waste Headline')).toBeInTheDocument();
    expect(screen.getAllByText('Context Avalanche').length).toBeGreaterThan(0);
    
    expect(screen.getByText('Waste %')).toBeInTheDocument();
    expect(screen.getByText('15.5%')).toBeInTheDocument();
    
    expect(screen.getByText('Projected Waste')).toBeInTheDocument();
    expect(screen.getByText('$20.50')).toBeInTheDocument();
    
    expect(screen.getByText('Weekly Waste')).toBeInTheDocument();
    expect(screen.getByText('$5.00')).toBeInTheDocument();
    
    expect(screen.getByText('Top Waste Causes')).toBeInTheDocument();
    expect(screen.getByText('Daily Waste Trend (30-day)')).toBeInTheDocument();
    expect(screen.getByText('Insights Log')).toBeInTheDocument();
    
    expect(screen.getByText('High (Danger)')).toBeInTheDocument();
    expect(screen.getByText('Excessive context window usage detected')).toBeInTheDocument();
  });

  it('shows insights 10 at a time with a load-more control', async () => {
    const mockStore: WasteStoreState = {
      data: {
        summary: null,
        insights: {
          items: Array.from({ length: 12 }, (_, index) => createInsight(index + 1)),
          empty: false
        },
        alerts: null
      },
      loading: false,
      error: null
    };

    vi.mocked(waste.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);

    expect(screen.getByText('Showing 10 of 12 insights')).toBeInTheDocument();
    expect(screen.getByText('Custom Detector 10')).toBeInTheDocument();
    expect(screen.queryByText('Custom Detector 11')).not.toBeInTheDocument();

    await fireEvent.click(screen.getByText('Show 10 more'));

    expect(screen.getByText('Showing 12 of 12 insights')).toBeInTheDocument();
    expect(screen.getByText('Custom Detector 11')).toBeInTheDocument();
    expect(screen.getByText('Custom Detector 12')).toBeInTheDocument();
    expect(screen.queryByText('Show 10 more')).not.toBeInTheDocument();
  });

  it('opens and closes insight detail modal', async () => {
    const mockStore: WasteStoreState = {
      data: {
        summary: {
          period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
          totalWasteCostUsd: 15.5,
          totalSpendCostUsd: 100,
          wastePercent: 15.5,
          weeklyWasteCostUsd: 5,
          monthlyWasteCostUsd: 15.5,
          projectedMonthEndWasteUsd: 20.5,
          byDetector: [],
          topCauses: [],
          dailyTrend: [],
          generatedAt: '2026-04-30T12:00:00Z'
        },
        insights: {
          items: [
            {
              insightId: 'ins-1',
              category: 'context_avalanche',
              severity: 'high',
              detectedAt: '2026-04-30T10:00:00Z',
              period: { month: '2026-04', startAt: '2026-04-01T00:00:00Z', endExclusive: '2026-05-01T00:00:00Z', currency: 'USD' },
              payload: {
                sessionIds: ['sess-1'],
                usageEntryIds: [],
                hashes: [{ kind: 'prompt_hash', value: 'abc123def456' }],
                counts: [{ key: 'turns', value: 5 }],
                metrics: [{ key: 'wasted_tokens', unit: 'tokens', value: 1000 }]
              }
            }
          ],
          empty: false
        },
        alerts: null
      },
      loading: false,
      error: null
    };
    
    vi.mocked(waste.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.queryByText('Insight Details')).not.toBeInTheDocument();
    
    const row = screen.getAllByText('Context Avalanche')[0].closest('tr');
    expect(row).not.toBeNull();
    await fireEvent.click(row!);
    
    expect(screen.getByText('Insight Details')).toBeInTheDocument();
    expect(screen.getByText('Payload Details')).toBeInTheDocument();
    expect(screen.getByText('wasted_tokens')).toBeInTheDocument();
    expect(screen.getByText('1000 tokens')).toBeInTheDocument();
    expect(screen.getByText('turns')).toBeInTheDocument();
    expect(screen.getByText('5')).toBeInTheDocument();
    expect(screen.getByText('prompt_hash')).toBeInTheDocument();
    expect(screen.getByText('abc123def456')).toBeInTheDocument();
    
    const closeButton = screen.getByText('Close');
    await fireEvent.click(closeButton);
    
    expect(screen.queryByText('Insight Details')).not.toBeInTheDocument();
  });

  it('shows error message when error exists', () => {
    const mockStore: WasteStoreState = {
      data: {
        summary: null,
        insights: null,
        alerts: null
      },
      loading: false,
      error: 'Failed to load insights data'
    };
    
    vi.mocked(waste.subscribe).mockImplementation((fn) => {
      fn(mockStore);
      return () => {};
    });

    render(Page);
    
    expect(screen.getByText('Failed to load insights data')).toBeInTheDocument();
  });
});
