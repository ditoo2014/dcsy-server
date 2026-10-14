<template>
  <div :id="id" :class="className" :style="{ height: height, width: width }" />
</template>

<script>
import { markRaw } from 'vue'
import resize from './mixins/resize.js'
import { safeEchartsSetOption } from '@/utils/echarts-safe.js'

export default {
  mixins: [resize],
  props: {
    className: {
      type: String,
      default: 'chart'
    },
    backgroundColor: {
      type: String,
      default: '#344b58'
    },
    id: {
      type: String,
      default: 'chart'
    },
    width: {
      type: String,
      default: '200px'
    },
    height: {
      type: String,
      default: '200px'
    },
    dataChangeKey: {
      type: Number,
      default: 0
    },
    chartData: {
      type: Object,
      default: () => ({
        tooltip: {
          trigger: 'item',
          formatter: '{a} <br/>{b} : {c} ({d}%)'
        },
        colors: ['red', 'green', 'yellow', 'blue'],
        series: [{
          type: 'pie',
          radius: '65%',
          center: ['50%', '50%'],
          selectedMode: 'single',
          label: {
            show: true,
            position: 'inside',
            formatter: '{b}'
          },
          data: []
        }]
      })
    }
  },
  data() {
    return {
      chart: null
    }
  },
  watch: {
    dataChangeKey() {
      this.reSetOption()
    },
    chartData: {
      handler() {
        this.reSetOption()
      },
      deep: true
    },
    width() {
      this.safeChartResize()
    },
    height() {
      this.safeChartResize()
    }
  },
  mounted() {
    this.initChart()
  },
  beforeUnmount() {
    if (this.chart && !this.chart.isDisposed()) {
      this.chart.dispose()
    }
    this.chart = null
  },
  methods: {
    safeChartResize() {
      this.$_scheduleChartResize()
    },
    buildOption() {
      const tooltip = this.chartData.tooltip
      const colors = this.chartData.colors
      const series = JSON.parse(JSON.stringify(this.chartData.series || []))
      const option = {
        animation: this.chartData.animation !== false,
        tooltip,
        series
      }
      if (colors) {
        option.color = colors
      }
      return option
    },
    reSetOption() {
      if (!this.chart || this.chart.isDisposed()) {
        return
      }
      safeEchartsSetOption(this.chart, this.buildOption(), { replaceMerge: ['series'] })
    },
    initChart() {
      const el = this.$el
      if (!el) {
        return
      }
      this.chart = markRaw(this.$echarts.init(el))
      this.reSetOption()
    }
  }
}
</script>
