import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import Page from './+page.svelte';
import * as bindings from '$lib/bindings';

vi.mock('$lib/bindings', () => ({
  graphTimeRanges: ['7 days', '30 days', 'All'],
  loadGraphs: vi.fn()
}));

vi.mock('$lib/scaffold-readiness', () => ({
  appTitle: 'Test App'
}));

vi.mock('$lib/components/charts/echartsAction', () => ({
  chartAction: () => ({
    destroy: vi.fn(),
    update: vi.fn()
  })
}));

describe('Graphs Page', () => {
  const mockGraphData: bindings.GraphResponse = {
    modelTokenUsages: [
      { modelName: 'gpt-4', totalTokens: 1000, inputTokens: 500, outputTokens: 500, cacheReadTokens: 0, cacheWriteTokens: 0 }
    ],
    modelCosts: [
      { modelName: 'gpt-4', totalCostUsd: 0.03 }
    ],
    dailyTokenTrends: Array.from({ length: 30 }, (_, i) => ({
      date: `2026-04-${(i + 1).toString().padStart(2, '0')}T00:00:00Z`,
      modelBreakdown: [{ modelName: 'gpt-4', totalTokens: 100 }]
    })),
    modelTokenBreakdowns: [
      { modelName: 'gpt-4', inputTokens: 500, outputTokens: 500, cacheReadTokens: 0, cacheWriteTokens: 0, totalTokens: 1000 }
    ]
  };

  function deferredGraphResponse() {
    let resolve!: (value: bindings.GraphResponse) => void;
    const promise = new Promise<bindings.GraphResponse>((resolver) => {
      resolve = resolver;
    });
    return { promise, resolve };
  }

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('shows loading state initially', () => {
    vi.mocked(bindings.loadGraphs).mockImplementation(() => new Promise(() => {}));
    render(Page);
    expect(screen.getByTestId('loading-spinner')).toBeInTheDocument();
  });

  it('shows error state if loading fails', async () => {
    vi.mocked(bindings.loadGraphs).mockRejectedValue(new Error('API Error'));
    render(Page);
    
    await waitFor(() => {
      expect(screen.getByText('API Error')).toBeInTheDocument();
    });
  });

  it('renders tabs and default chart', async () => {
    vi.mocked(bindings.loadGraphs).mockResolvedValue(mockGraphData);
    render(Page);
    
    await waitFor(() => {
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
    });

    expect(screen.getByTestId('tab-Model-Token-Usage')).toBeInTheDocument();
    expect(screen.getByTestId('tab-Model-Cost')).toBeInTheDocument();
    expect(screen.getByTestId('tab-Daily-Token-Trend')).toBeInTheDocument();
    expect(screen.getByTestId('tab-Model-Token-Breakdown')).toBeInTheDocument();
  });

  it('switches tabs correctly', async () => {
    vi.mocked(bindings.loadGraphs).mockResolvedValue(mockGraphData);
    render(Page);
    
    await waitFor(() => {
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
    });

    await fireEvent.click(screen.getByTestId('tab-Daily-Token-Trend'));
    expect(screen.getByTestId('tab-Daily-Token-Trend')).toHaveClass('border-primary');

    await fireEvent.click(screen.getByTestId('tab-Model-Token-Breakdown'));
    expect(screen.getByTestId('tab-Model-Token-Breakdown')).toHaveClass('border-primary');
  });

  it('filters daily token trend by time range', async () => {
    vi.mocked(bindings.loadGraphs).mockResolvedValue(mockGraphData);
    render(Page);
    
    await waitFor(() => {
      expect(screen.queryByTestId('loading-spinner')).not.toBeInTheDocument();
    });

    await fireEvent.click(screen.getByTestId('tab-Daily-Token-Trend'));
    
    const select = screen.getByTestId('time-range-selector');
    await fireEvent.change(select, { target: { value: '7 days' } });

    await waitFor(() => {
      expect((select as HTMLSelectElement).value).toBe('7 days');
      expect(bindings.loadGraphs).toHaveBeenLastCalledWith('', '7 days');
    });
  });

  it('keeps the latest time range result when graph loads overlap', async () => {
    const initialLoad = deferredGraphResponse();
    const latestLoad = deferredGraphResponse();
    const emptyGraphData = {
      modelTokenUsages: [],
      modelCosts: [],
      dailyTokenTrends: [],
      modelTokenBreakdowns: []
    };

    vi.mocked(bindings.loadGraphs)
      .mockReturnValueOnce(initialLoad.promise)
      .mockReturnValueOnce(latestLoad.promise);

    render(Page);

    const select = screen.getByTestId('time-range-selector');
    await fireEvent.change(select, { target: { value: '7 days' } });

    latestLoad.resolve(emptyGraphData);

    await waitFor(() => {
      expect(screen.getByTestId('empty-chart')).toBeInTheDocument();
    });

    initialLoad.resolve(mockGraphData);

    await waitFor(() => {
      expect(screen.getByTestId('empty-chart')).toBeInTheDocument();
      expect(bindings.loadGraphs).toHaveBeenLastCalledWith('', '7 days');
    });
  });

  it('shows empty state when no data', async () => {
    vi.mocked(bindings.loadGraphs).mockResolvedValue({
      modelTokenUsages: [],
      modelCosts: [],
      dailyTokenTrends: [],
      modelTokenBreakdowns: []
    });
    render(Page);
    
    await waitFor(() => {
      expect(screen.getByTestId('empty-chart')).toBeInTheDocument();
    });
  });
});
