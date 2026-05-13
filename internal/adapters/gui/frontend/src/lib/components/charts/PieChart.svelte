<script lang="ts">
  import type { EChartsOption } from 'echarts';
  import { chartAction } from './echartsAction';

  export let data: { name: string; value: number }[] = [];
  export let name: string = '';
  export let donut: boolean = false;

  $: options = {
    tooltip: {
      trigger: 'item'
    },
    legend: {
      bottom: 0,
      type: 'scroll'
    },
    series: [
      {
        name: name,
        type: 'pie',
        radius: donut ? ['40%', '70%'] : '70%',
        center: ['50%', '45%'],
        data: data
      }
    ]
  } as EChartsOption;
</script>

<div class="w-full h-full" use:chartAction={options}></div>
