import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest';
import { render } from '@testing-library/svelte';
import LineChart from './LineChart.svelte';

describe('LineChart', () => {
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
    const { container } = render(LineChart, {
      xAxisData: ['Mon', 'Tue'],
      series: [{ name: 'Test', data: [10, 20] }]
    });
    expect(container).toBeTruthy();
  });
});
