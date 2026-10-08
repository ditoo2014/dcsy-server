<template>
  <div :ref="refKey" :style="{ height: height, width: width }" />
</template>

<script>
import { markRaw } from 'vue'
import { safeEchartsResize, safeEchartsSetOption } from '@/utils/echarts-safe.js'

/**
 * 与原 visualMap.inRange 分段一致（见 setChartData 内 pieces）
 * 实际大屏上多为：0 → 深橄榄绿；1~20 → 亮黄绿（山西/广东等）
 * 数量更大时才会出现蓝、浅蓝、纯绿
 */
const MAP_AREA_COLORS = {
  zero: '#474503',
  tier1: '#C3E43B',
  tier2: '#0E94EB',
  tier3: '#6FBCF0',
  tier4: '#00CC00'
}

function getMapAreaColor(value) {
  const v = Number(value) || 0
  if (v === 0) return MAP_AREA_COLORS.zero
  if (v <= 20) return MAP_AREA_COLORS.tier1
  if (v <= 50) return MAP_AREA_COLORS.tier2
  if (v <= 100) return MAP_AREA_COLORS.tier3
  return MAP_AREA_COLORS.tier4
}

export default {
  props: {
    refKey: {
      type: String,
      default: 'refChina'
    },
    width: {
      type: String,
      default: '500px'
    },
    height: {
      type: String,
      default: '500px'
    },
    allEgoData: {
      type: Object,
      default: () => ({})
    },
    egoMsg: {
      type: Object,
      default: () => ({
        areaName: '中国',
        pingYinName: 'china',
        adcode: 100000
      })
    },
    chartData: {
      type: Object,
      default: () => ({
        seriesData: []
      })
    }
  },
  data() {
    return {
      chart: null,
      seriesMap: 'china',
      mapZoom: 1.2,
      refreshFrameId: null
    }
  },
  watch: {
    allEgoData() {
      this.scheduleChartRefresh()
    },
    chartData: {
      handler() {
        if (this.chartData.seriesData && this.chartData.seriesData.length > 0) {
          this.scheduleChartRefresh()
        }
      },
      deep: true,
      immediate: true
    },
    width() {
      this.safeChartResize()
    },
    height() {
      this.safeChartResize()
    }
  },
  mounted() {
    this.init()
    this.__resizeHandler = () => this.safeChartResize()
    window.addEventListener('resize', this.__resizeHandler)
  },
  beforeUnmount() {
    window.removeEventListener('resize', this.__resizeHandler)
    if (this.refreshFrameId !== null) {
      cancelAnimationFrame(this.refreshFrameId)
      this.refreshFrameId = null
    }
    this.disposeChart()
  },
  methods: {
    scheduleChartRefresh() {
      if (this.refreshFrameId !== null) return
      this.refreshFrameId = requestAnimationFrame(() => {
        this.refreshFrameId = null
        this.init()
      })
    },
    getChartEl() {
      return this.$refs[this.refKey]
    },
    disposeChart() {
      if (this.chart && !this.chart.isDisposed()) {
        this.chart.off('click')
        this.chart.dispose()
      }
      this.chart = null
    },
    safeChartResize() {
      if (!safeEchartsResize(this.chart, this.getChartEl())) {
        this.setChartData()
      }
    },
    init() {
      if (!this.allEgoData || !this.allEgoData.type) {
        return
      }
      const el = this.getChartEl()
      if (!el) {
        return
      }
      if (!this.chart || this.chart.isDisposed()) {
        this.chart = markRaw(this.$echarts.init(el))
      }
      if (this.chartData.seriesData && this.chartData.seriesData.length > 0) {
        this.setChartData()
      }
    },
    buildColoredSeriesData(seriesData) {
      return seriesData.map(item => ({
        ...item,
        itemStyle: {
          areaColor: getMapAreaColor(item.value),
          borderColor: 'rgba(255, 255, 255, 0.35)',
          borderWidth: 1
        }
      }))
    },
    async setChartData() {
      if (!this.allEgoData || !this.allEgoData.type || !this.chart || this.chart.isDisposed()) {
        return
      }

      const egoMsg = this.egoMsg
      this.seriesMap = egoMsg.pingYinName
      await this.$echarts.registerMap(egoMsg.pingYinName, this.allEgoData)
      const seriesData = this.buildColoredSeriesData(this.chartData.seriesData || [])

      const option = {
        animation: false,
        tooltip: {
          trigger: 'item',
          triggerOn: 'mousemove'
        },
        grid: {
          left: 0,
          right: 0,
          top: 0,
          bottom: 0
        },
        textStyle: { color: '#fff' },
        visualMap: {
          type: 'piecewise',
          min: 0,
          max: 1000,
          show: false,
          realtime: false,
          calculable: true,
          inRange: {
            color: [
              MAP_AREA_COLORS.zero,
              MAP_AREA_COLORS.tier1,
              MAP_AREA_COLORS.tier2,
              MAP_AREA_COLORS.tier3,
              '#F0F06F',
              MAP_AREA_COLORS.tier4
            ]
          },
          pieces: [
            { min: 0, max: 0 },
            { min: 1, max: 20 },
            { min: 20, max: 50 },
            { min: 50, max: 100 },
            { min: 100 }
          ]
        },
        series: [
          {
            name: '中国省份地图',
            type: 'map',
            map: this.seriesMap,
            zoom: this.mapZoom,
            roam: false,
            label: {
              show: true,
              formatter: [
                '{style|{b}({c})}'
              ].join('\n'),
              fontSize: 12,
              fontWeight: 'bold',
              rich: {
                style: {
                  color: '#ffffff',
                  textShadowColor: 'rgb(34 73 103)',
                  textShadowBlur: 10,
                  fontSize: 12,
                  fontWeight: 'bold'
                }
              }
            },
            tooltip: {
              show: false
            },
            emphasis: {
              itemStyle: {
                borderColor: '#FFD700',
                borderWidth: 3,
                // shadowColor: 'rgba(255, 215, 0, 0.8)',
                shadowBlur: 20
              },
              label: {
                color: '#FFD700',
                fontSize: 14,
                fontWeight: 'bold',
                textShadowColor: 'rgba(255, 215, 0, 0.8)',
                textShadowBlur: 15
              }
            },
            data: seriesData
          }
        ]
      }

      if (safeEchartsSetOption(this.chart, option, { notMerge: true })) {
        this.chart.off('click')
        this.chart.on('click', params => {
          this.$emit('choose', params)
        })
        this.$nextTick(() => this.safeChartResize())
      }
    }
  }
}
</script>
