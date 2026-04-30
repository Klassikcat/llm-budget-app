import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest';
import { render } from '@testing-library/svelte';
import BarChart from './BarChart.svelte';

describe('BarChart', () => {
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
    const { container } = render(BarChart, {
      xAxisData: ['Model A', 'Model B'],
      series: [{ name: 'Usage', data: [100, 200] }]
    });
    expect(container).toBeTruthy();
  });
});
