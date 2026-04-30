import { describe, it, expect, vi, beforeAll, afterAll } from 'vitest';
import { render } from '@testing-library/svelte';
import PieChart from './PieChart.svelte';

describe('PieChart', () => {
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
    const { container } = render(PieChart, {
      data: [{ name: 'Model A', value: 100 }, { name: 'Model B', value: 200 }],
      name: 'Usage'
    });
    expect(container).toBeTruthy();
  });
});
