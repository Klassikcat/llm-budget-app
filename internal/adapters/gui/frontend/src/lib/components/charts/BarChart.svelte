<script lang="ts">
  import type { EChartsOption } from 'echarts';
  import { chartAction } from './echartsAction';

  export let xAxisData: string[] = [];
  export let series: { name: string; data: number[] }[] = [];
  export let yAxisName: string = '';
  export let colorByData = false;

  $: options = {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow'
      }
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
      data: xAxisData,
      axisLabel: {
        interval: 0,
        rotate: 30
      }
    },
    yAxis: {
      type: 'value',
      name: yAxisName
    },
    series: series.map(s => ({
      name: s.name,
      type: 'bar',
      colorBy: colorByData ? 'data' : 'series',
      data: s.data
    }))
  } as EChartsOption;
</script>

<div class="w-full h-full" use:chartAction={options}></div>
