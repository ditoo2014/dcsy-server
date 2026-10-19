<template>
  <div :id="id" :class="className" :style="{ height: height, width: width }" />
</template>

<script>
// 基础饼状图 https://echarts.apache.org/examples/zh/editor.html?c=pie-simple
import resize from './mixins/resize.js'
import { markRaw } from 'vue'
import { linearGradientColor, normalizeSeriesItem } from '@/utils/echartsGradient.js'
import { safeEchartsSetOption } from '@/utils/echartsSafe.js'

export default {
  mixins: [resize],
  props: {
    className: {
      type: String,
      default: 'chart'
    },
    backgroundColor: { // 注意: 设置透明度或设置为null时，会导致图表中的文字显示异常。此时需要在 series 中的 label 设置 color 属性。
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
      default: () => {
        return {
          legend: {
            data: ['字段1', '字段2'] // 曲线标题
          },
          xAxis: { // 横轴
            data: ['8:00', '9:00', '10:00', '11:00', '12:00', '13:00', '14:00', '15:00', '16:00', '17:00', '18:00', '19:00', '20:00', '21:00', '22:00']
          },
          yAxis: [ // 纵轴
            { name: '(%)' }
          ],
          showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
          series: [
            {
              name: '字段1',
              type: 'line',
              smooth: true,
              symbol: 'circle',
              symbolSize: 5,
              showSymbol: false,
              lineStyle: {
                color: 'rgba(228, 228, 126, 1)',
                width: 2
              },
              areaStyle: {
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              },
              itemStyle: {
                color: '#dddc6b',
                borderColor: 'rgba(221, 220, 107, .1)',
                borderWidth: 12
              },
              data: [600, 200, 600, 200, 400, 200, 400, 300, 400, 300, 400, 300, 200, 400, 200]
            }, {
              name: '字段2',
              type: 'line',
              smooth: true,
              symbol: 'circle',
              symbolSize: 5,
              showSymbol: false,
              lineStyle: {
                color: 'rgba(255, 128, 128, 1)',
                width: 2
              },
              areaStyle: {
                shadowColor: 'rgba(0, 0, 0, 0.1)'
              },
              itemStyle: {
                color: '#dddc6b',
                borderColor: 'rgba(221, 220, 107, .1)',
                borderWidth: 12
              },
              data: [300, 100, 300, 400, 300, 400, 300, 600, 200, 400, 200, 400, 100, 300, 400]
            }
          ]
        }
      }
    }
  },
  data() {
    return {
      chart: null,
      option: {},
      legendSelected: {}
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
      deep: true // 设为true时监听对象属性值变化
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
    if (!this.chart) {
      return
    }
    this.chart.off('legendselectchanged', this.handleLegendSelectChanged)
    this.chart.dispose()
    this.chart = null
  },
  methods: {
    safeChartResize() {
      this.$_scheduleChartResize()
    },
    buildSeries() {
      const source = this.chartData.series || []
      return source.map((item, index) => {
        const series = normalizeSeriesItem(item)
        if (!series.type) {
          series.type = 'line'
        }
        if (!series.areaStyle) {
          series.areaStyle = {}
        }
        if (!series.areaStyle.color) {
          if (index === 0) {
            series.areaStyle.color = linearGradientColor([{
              offset: 0,
              color: 'rgba(228, 228, 126, .8)'
            }, {
              offset: 0.8,
              color: 'rgba(228, 228, 126, 0.1)'
            }])
          } else if (index === 1) {
            series.areaStyle.color = linearGradientColor([{
              offset: 0,
              color: 'rgba(255, 128, 128,.8)'
            }, {
              offset: 0.8,
              color: 'rgba(255, 128, 128, .1)'
            }])
          }
        }
        return series
      }).filter(item => item && item.type)
    },
    buildLegendSelected() {
      const legend = this.chartData.legend || {}
      const configuredSelected = legend.selected || {}
      const selected = {}
      ;(legend.data || []).forEach(item => {
        const name = typeof item === 'string' ? item : item && item.name
        if (!name) return
        if (Object.prototype.hasOwnProperty.call(this.legendSelected, name)) {
          selected[name] = this.legendSelected[name]
        } else if (Object.prototype.hasOwnProperty.call(configuredSelected, name)) {
          selected[name] = configuredSelected[name] !== false
        } else {
          selected[name] = true
        }
      })
      this.legendSelected = selected
      return selected
    },
    handleLegendSelectChanged(params) {
      this.legendSelected = { ...(params && params.selected ? params.selected : {}) }
    },
    formatTooltip(params) {
      const items = Array.isArray(params) ? params : [params]
      const visibleItems = items.filter(Boolean)
      if (!visibleItems.length) return ''

      const title = visibleItems[0].axisValueLabel || visibleItems[0].name || ''
      const lines = title ? [title] : []
      visibleItems.forEach(item => {
        const rawValue = Array.isArray(item.value) ? item.value[item.value.length - 1] : item.value
        lines.push(`${item.marker || ''}${item.seriesName || ''}: ${rawValue == null ? '-' : rawValue}`)
      })
      return lines.join('<br/>')
    },
    reSetOption() {
      if (!this.chart || this.chart.isDisposed() || !this.chartData) {
        return
      }

      const legendConfig = this.chartData.legend || {}
      const legendSelected = this.buildLegendSelected()
      const xAxisData = (this.chartData.xAxis && this.chartData.xAxis.data) || []
      const yAxisName = (this.chartData.yAxis && this.chartData.yAxis[0] && this.chartData.yAxis[0].name) || ''
      const series = this.buildSeries()
      if (!series.length) {
        return
      }
      const visibleSeries = series.filter(item => legendSelected[item.name] !== false)
      const hasVisibleData = visibleSeries.some(item => {
        const data = Array.isArray(item.data) ? item.data : []
        return data.some(value => {
          const rawValue = value && typeof value === 'object' ? value.value : value
          const numericValue = Number(rawValue)
          return Number.isFinite(numericValue) && numericValue !== 0
        })
      })

      this.option = {
        animation: this.chartData.animation !== false,
        tooltip: {
          show: true,
          trigger: 'axis',
          triggerOn: 'mousemove|click',
          confine: true,
          formatter: params => this.formatTooltip(params),
          axisPointer: {
            lineStyle: {
              color: '#dddc6b'
            }
          }
        },
        grid: {
          top: 50,
          left: '10%',
          right: '5%',
          bottom: 60
        },
        legend: {
          ...legendConfig,
          data: legendConfig.data || [],
          selected: legendSelected,
          selectedMode: legendConfig.selectedMode === undefined ? true : legendConfig.selectedMode,
          top: legendConfig.top === undefined ? 0 : legendConfig.top,
          inactiveColor: legendConfig.inactiveColor || 'rgba(143, 170, 196, 0.45)',
          inactiveBorderColor: legendConfig.inactiveBorderColor || 'rgba(143, 170, 196, 0.45)',
          textStyle: {
            color: '#fff',
            ...(legendConfig.textStyle || {})
          },
          itemWidth: legendConfig.itemWidth || 12,
          itemHeight: legendConfig.itemHeight || 10
        },
        graphic: hasVisibleData ? [] : [{
          type: 'text',
          left: 'center',
          top: 'middle',
          silent: true,
          style: {
            text: '近7天暂无数据',
            fill: '#8FAAC4',
            fontSize: 14
          }
        }],
        xAxis: {
          type: 'category',
          boundaryGap: false,
          axisLine: {
            lineStyle: {
              color: '#fff'
            }
          },
          splitLine: {
            show: false
          },
          axisTick: {
            show: false
          },
          splitArea: {
            show: false
          },
          axisLabel: {
            formatter: (value) => String(value).split(' ').join('\n')
          },
          data: xAxisData
        },
        yAxis: {
          type: 'value',
          name: yAxisName,
          nameLocation: 'end',
          axisLine: {
            onZero: false,
            lineStyle: {
              color: '#fff'
            }
          },
          splitLine: {
            show: false
          },
          axisTick: {
            show: false
          },
          axisLabel: {
            interval: 0
          },
          splitArea: {
            show: false
          }
        },
        series
      }

      safeEchartsSetOption(this.chart, this.option, { notMerge: true })
    },
    initChart() {
      const el = this.$el
      if (!el) {
        return
      }
      this.chart = markRaw(this.$echarts.init(el))
      this.chart.on('legendselectchanged', this.handleLegendSelectChanged)
      this.reSetOption()
    }
  }
}
</script>
