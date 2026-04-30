import * as echarts from 'echarts';
import type { EChartsOption, ECharts } from 'echarts';
import { resolvedTheme } from '$lib/stores/theme';

export function chartAction(node: HTMLElement, options: EChartsOption) {
  let chart: ECharts | undefined;
  let resizeObserver: ResizeObserver | undefined;
  let unsubscribeTheme: () => void;
  let currentTheme = 'dark';

  function initChart() {
    if (chart) {
      chart.dispose();
    }
    chart = echarts.init(node, currentTheme, { renderer: 'canvas' });
    chart.setOption(options);
  }

  unsubscribeTheme = resolvedTheme.subscribe((t) => {
    if (currentTheme !== t) {
      currentTheme = t;
      initChart();
    }
  });

  initChart();

  resizeObserver = new ResizeObserver(() => {
    if (chart) {
      chart.resize();
    }
  });
  resizeObserver.observe(node);

  return {
    update(newOptions: EChartsOption) {
      if (chart) {
        chart.setOption(newOptions, true);
      }
    },
    destroy() {
      if (unsubscribeTheme) unsubscribeTheme();
      if (resizeObserver) resizeObserver.disconnect();
      if (chart) chart.dispose();
    }
  };
}
