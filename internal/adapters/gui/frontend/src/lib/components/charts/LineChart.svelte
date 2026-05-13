<script lang="ts">
  import type { EChartsOption } from 'echarts';
  import { chartAction } from './echartsAction';

  export let xAxisData: string[] = [];
  export let series: { name: string; data: number[] }[] = [];
  export let yAxisName: string = '';

  $: options = {
    tooltip: {
      trigger: 'axis'
    },
    legend: {
      data: series.map(s => s.name),
      bottom: 0
    },
    grid: {
      left: '3%',
      right: '4%',
      bottom: '15%',
      top: '10%',
      containLabel: true
    },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: xAxisData
    },
    yAxis: {
      type: 'value',
      name: yAxisName
    },
    series: series.map(s => ({
      name: s.name,
      type: 'line',
      data: s.data,
      smooth: true,
      showSymbol: false
    }))
  } as EChartsOption;
</script>

<div class="w-full h-full" use:chartAction={options}></div>
