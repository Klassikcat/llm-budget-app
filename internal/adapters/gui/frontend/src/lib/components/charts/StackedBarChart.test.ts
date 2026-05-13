import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest';
import { render } from '@testing-library/svelte';
import StackedBarChart from './StackedBarChart.svelte';

describe('StackedBarChart', () => {
  beforeAll(() => {
    global.ResizeObserver = class ResizeObserver {
      observe() {}
      unobserve() {}
      disconnect() {}
    };
    
    vi.mock('echarts', () => ({
      init: vi.fn(() => ({
        setOption: vi.fn(),
        resize: vi.fn(),
        dispose: vi.fn()
      }))
    }));
  });

  afterAll(() => {
    vi.restoreAllMocks();
  });

  it('renders without crashing', () => {
    const { container } = render(StackedBarChart, {
      xAxisData: ['Model A', 'Model B'],
      series: [
        { name: 'Input', data: [50, 100] },
        { name: 'Output', data: [50, 100] }
      ]
    });
    expect(container).toBeTruthy();
  });
});
