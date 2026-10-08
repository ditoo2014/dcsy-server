<template>
  <div :id="id" :class="className" :style="{height:height,width:width}" />
</template>

<script>
// 多 Y 轴示例 https://echarts.apache.org/examples/zh/editor.html?c=multiple-y-axis
import resize from './mixins/resize.js'
import { markRaw } from 'vue'
import { safeEchartsResize, safeEchartsSetOption, normalizeChartSeries } from '@/utils/echarts-safe.js'

const dataZoomLinkGroups = new Map()

export default {
  mixins: [resize],
  props: {
    className: {
      type: String,
      default: 'chart'
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
          colors: ['#5470C6', '#EE6666'],
          legend: {
            show: true,
            data: ['Evaporation', 'Temperature'] // 曲线标题
          },
          xAxis: { // 横轴
            data: ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec']
          },
          yAxis: [{ // 纵轴
            type: 'value',
            name: 'Evaporation',
            position: 'right',
            alignTicks: true,
            axisLine: {
              show: true,
              lineStyle: {
                color: ''
              }
            },
            axisLabel: {
              formatter: '{value} ml'
            }
          }, {
            type: 'value',
            name: '温度',
            position: 'left',
            alignTicks: true,
            axisLine: {
              show: true,
              lineStyle: {
                color: ''
              }
            },
            axisLabel: {
              formatter: '{value} °C'
            }
          }],
          showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
          series: [{
            name: '',
            type: 'bar',
            label: {
              show: true,
              position: 'top',
              color: '#fff'
            },
            data: [
              2.0, 4.9, 7.0, 23.2, 25.6, 76.7, 135.6, 162.2, 32.6, 20.0, 6.4, 3.3
            ]
          }, {
            name: '',
            type: 'line',
            yAxisIndex: 1,
            data: [2.0, 2.2, 3.3, 4.5, 6.3, 10.2, 20.3, 23.4, 23.0, 16.5, 12.0, 6.2]
          }]
        }
      }
    },
    /** 开启时间轴缩放（滚轮/拖拽） */
    enableDataZoom: {
      type: Boolean,
      default: false
    },
    /** 是否显示底部时间轴滑块 */
    showDataZoomSlider: {
      type: Boolean,
      default: false
    },
    /** 多图联动分组名，相同组名图表缩放/滚动同步 */
    linkGroup: {
      type: String,
      default: ''
    },
    /** 根据当前缩放窗口内的数据量自动显示或隐藏折线点 */
    dynamicSymbolOnZoom: {
      type: Boolean,
      default: false
    },
    /** 当前可视区单条曲线不超过该数量时显示折线点 */
    symbolVisibleThreshold: {
      type: Number,
      default: 80
    }
  },
  data() {
    return {
      chart: null,
      option: {}
    }
  },
  watch: {
    dataChangeKey() {
      this.resetYAxisSnapshot()
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
    this.teardownYAxisLockHandlers()
    this.unregisterDataZoomLink()
    this.chart.dispose()
    this.chart = null
  },
  methods: {
    safeChartResize() {
      if (!safeEchartsResize(this.chart, document.getElementById(this.id))) {
        this.reSetOption()
      }
    },
    buildDataZoomOption(xAxisType, seriesList) {
      if (!this.enableDataZoom || xAxisType !== 'time') {
        return null
      }
      const hasData = (seriesList || []).some(
        s => Array.isArray(s.data) && s.data.length > 0
      )
      if (!hasData) {
        return null
      }
      if (this.chartData.dataZoom) {
        return this.chartData.dataZoom
      }
      const sliderStyle = {
        id: 'linkedDayTimeSlider',
        show: this.showDataZoomSlider,
        xAxisIndex: [0],
        filterMode: 'none',
        height: 20,
        bottom: 5,
        start: 0,
        end: 100,
        handleIcon: 'path://M306.1,413c0,2.2-1.8,4-4,4h-59.8c-2.2,0-4-1.8-4-4V200.8c0-2.2,1.8-4,4-4h59.8c2.2,0,4,1.8,4,4V413z',
        handleSize: '110%',
        handleStyle: {
          color: '#d3dee5'
        },
        textStyle: {
          color: '#666'
        },
        borderColor: '#90979c'
      }
      // 联动图须保持相同 dataZoom 结构与 id，否则 connect 无法双向同步
      const zoomList = [sliderStyle, {
        id: 'linkedDayTimeInside',
        type: 'inside',
        show: true,
        xAxisIndex: [0],
        filterMode: 'none',
        start: 0,
        end: 100,
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
        moveOnMouseWheel: false
      }]
      return zoomList
    },
    registerDataZoomLink() {
      const groupName = this.enableDataZoom ? this.linkGroup : ''
      if (this._registeredDataZoomGroup === groupName) {
        return
      }
      this.unregisterDataZoomLink()
      if (!groupName) {
        return
      }
      if (!dataZoomLinkGroups.has(groupName)) {
        dataZoomLinkGroups.set(groupName, new Set())
      }
      dataZoomLinkGroups.get(groupName).add(this)
      this._registeredDataZoomGroup = groupName
    },
    unregisterDataZoomLink() {
      const groupName = this._registeredDataZoomGroup
      if (!groupName) {
        return
      }
      const group = dataZoomLinkGroups.get(groupName)
      if (group) {
        group.delete(this)
        if (!group.size) {
          dataZoomLinkGroups.delete(groupName)
        }
      }
      this._registeredDataZoomGroup = ''
    },
    syncLinkedDataZoom(event) {
      if (this._receivingLinkedDataZoom || !this._registeredDataZoomGroup) {
        return
      }
      const range = Array.isArray(event?.batch) ? event.batch[0] : event
      if (!range) {
        return
      }
      const zoomRange = {}
      ;['start', 'end', 'startValue', 'endValue'].forEach(key => {
        if (range[key] != null) {
          zoomRange[key] = range[key]
        }
      })
      if (!Object.keys(zoomRange).length) {
        return
      }
      const group = dataZoomLinkGroups.get(this._registeredDataZoomGroup)
      group?.forEach(target => {
        if (target !== this) {
          target.applyLinkedDataZoom(zoomRange)
        }
      })
    },
    applyLinkedDataZoom(zoomRange) {
      if (!this.chart || this.chart.isDisposed()) {
        return
      }
      this._receivingLinkedDataZoom = true
      try {
        this.chart.dispatchAction({
          type: 'dataZoom',
          ...zoomRange
        })
      } finally {
        this._receivingLinkedDataZoom = false
      }
    },
    getSeriesPointTime(point) {
      const value = point && typeof point === 'object' && !Array.isArray(point)
        ? point.value
        : point
      if (!Array.isArray(value) || value.length < 2) {
        return null
      }
      const time = Number(value[0])
      return Number.isFinite(time) ? time : null
    },
    getVisibleTimeRange(seriesList) {
      let minTime = Infinity
      let maxTime = -Infinity
      seriesList.forEach(series => {
        const seriesData = series.data || []
        seriesData.forEach(point => {
          const time = this.getSeriesPointTime(point)
          if (time == null) {
            return
          }
          minTime = Math.min(minTime, time)
          maxTime = Math.max(maxTime, time)
        })
      })
      if (!Number.isFinite(minTime) || !Number.isFinite(maxTime)) {
        return null
      }

      const dataZoomOption = this.chart.getOption().dataZoom
      const dataZoom = Array.isArray(dataZoomOption) ? dataZoomOption[0] : dataZoomOption
      if (!dataZoom) {
        return { min: minTime, max: maxTime }
      }
      const start = Number.isFinite(Number(dataZoom.start)) ? Number(dataZoom.start) : 0
      const end = Number.isFinite(Number(dataZoom.end)) ? Number(dataZoom.end) : 100
      const totalRange = maxTime - minTime
      return {
        min: minTime + totalRange * start / 100,
        max: minTime + totalRange * end / 100
      }
    },
    updateDynamicSymbols() {
      if (!this.dynamicSymbolOnZoom || !this.chart || this.chart.isDisposed()) {
        return
      }
      const optionSeries = this.chart.getOption().series
      const seriesList = (Array.isArray(optionSeries) ? optionSeries : [optionSeries])
        .filter(series => series && series.type === 'line')
      const visibleRange = this.getVisibleTimeRange(seriesList)
      if (!visibleRange) {
        return
      }
      const maxVisiblePointCount = seriesList.reduce((maxCount, series) => {
        const visibleCount = (series.data || []).reduce((count, point) => {
          const time = this.getSeriesPointTime(point)
          return time != null && time >= visibleRange.min && time <= visibleRange.max
            ? count + 1
            : count
        }, 0)
        return Math.max(maxCount, visibleCount)
      }, 0)
      const showSymbol = maxVisiblePointCount <= this.symbolVisibleThreshold
      if (this._dynamicSymbolsVisible === showSymbol) {
        return
      }
      this._dynamicSymbolsVisible = showSymbol
      const sourceSeries = (this.chartData.series || [])
        .filter(series => series && series.type === 'line')
      this.chart.setOption({
        series: seriesList.map((series, index) => {
          const source = sourceSeries[index] || {}
          const sourceItemStyle = source.itemStyle || {}
          const sourceEmphasis = source.emphasis || {}
          const sourceEmphasisItemStyle = sourceEmphasis.itemStyle || {}
          if (showSymbol) {
            return {
              showSymbol: true,
              symbolSize: source.symbolSize || 4,
              itemStyle: {
                ...sourceItemStyle,
                opacity: 1
              },
              emphasis: sourceEmphasis
            }
          }
          return {
            showSymbol: true,
            symbolSize: 8,
            itemStyle: {
              ...sourceItemStyle,
              opacity: 0
            },
            emphasis: {
              ...sourceEmphasis,
              focus: 'none',
              scale: 0.5,
              itemStyle: {
                ...sourceEmphasisItemStyle,
                opacity: 1
              }
            }
          }
        })
      }, { notMerge: false })
    },
    resetYAxisSnapshot() {
      if (!this.enableDataZoom) {
        return
      }
      this._yAxisSnapshot = null
      this._pendingYAxisCapture = true
    },
    hasSeriesData(seriesList) {
      return (seriesList || []).some(series => {
        return Array.isArray(series.data) && series.data.some(point => {
          const value = point && typeof point === 'object' && !Array.isArray(point)
            ? point.value
            : point
          const rawValue = Array.isArray(value) ? value[1] : value
          return rawValue !== null && rawValue !== '' && Number.isFinite(Number(rawValue))
        })
      })
    },
    applyEmptyYAxisRange(yAxis, seriesList) {
      const hasData = this.hasSeriesData(seriesList)
      yAxis.forEach(axis => {
        const emptyRange = axis.emptyRange
        delete axis.emptyRange
        if (!hasData && emptyRange) {
          axis.min = emptyRange.min
          axis.max = emptyRange.max
          axis.interval = emptyRange.interval
        } else if (emptyRange) {
          axis.min = null
          axis.max = null
          axis.interval = null
        }
      })
    },
    captureYAxisSnapshotFromChart() {
      if (!this.chart || this.chart.isDisposed()) {
        return null
      }
      const ecModel = this.chart.getModel()
      if (!ecModel) {
        return null
      }
      const yAxisOpt = this.chart.getOption().yAxis
      const axisCount = Array.isArray(yAxisOpt) ? yAxisOpt.length : (yAxisOpt ? 1 : 0)
      if (!axisCount) {
        return null
      }
      const snapshot = []
      for (let i = 0; i < axisCount; i++) {
        const comp = ecModel.getComponent('yAxis', i)
        const scale = comp && comp.axis && comp.axis.scale
        if (!scale) {
          return null
        }
        const extent = scale.getExtent()
        const min = extent[0]
        const max = extent[1]
        if (!Number.isFinite(min) || !Number.isFinite(max)) {
          return null
        }
        snapshot.push({ min, max })
      }
      return snapshot
    },
    buildLockedYAxisOption() {
      if (!this._yAxisSnapshot?.length) {
        return null
      }
      return this._yAxisSnapshot.map(snap => ({
        min: snap.min,
        max: snap.max,
        scale: false
      }))
    },
    applyLockedYAxisRange(yAxis) {
      if (!this.enableDataZoom || this._pendingYAxisCapture || !this._yAxisSnapshot?.length) {
        return
      }
      yAxis.forEach((axis, index) => {
        const snap = this._yAxisSnapshot[index]
        if (!snap) {
          return
        }
        axis.min = snap.min
        axis.max = snap.max
        axis.scale = false
      })
    },
    scheduleRestoreLockedYAxis() {
      if (!this.enableDataZoom || !this._yAxisSnapshot?.length || this._yAxisRestoring) {
        return
      }
      if (this._yAxisRestoreTimer) {
        return
      }
      this._yAxisRestoreTimer = requestAnimationFrame(() => {
        this._yAxisRestoreTimer = null
        this.restoreLockedYAxis()
      })
    },
    restoreLockedYAxis() {
      if (!this.enableDataZoom || !this._yAxisSnapshot?.length || this._yAxisRestoring) {
        return
      }
      if (!this.chart || this.chart.isDisposed()) {
        return
      }
      const lockedYAxis = this.buildLockedYAxisOption()
      if (!lockedYAxis) {
        return
      }
      this._yAxisRestoring = true
      // 仅合并 min/max，保留 axisLabel.formatter 等单位配置
      this.chart.setOption({ yAxis: lockedYAxis }, { notMerge: false })
      this._yAxisRestoring = false
    },
    setupYAxisLockHandlers() {
      if (!this.enableDataZoom || this._yAxisLockHandlersReady) {
        return
      }
      this._onChartFinishedForYAxis = () => {
        if (!this._pendingYAxisCapture) {
          return
        }
        const optionSeries = this.chart.getOption().series
        const seriesArr = Array.isArray(optionSeries) ? optionSeries : [optionSeries]
        if (!this.hasSeriesData(seriesArr)) {
          return
        }
        const snapshot = this.captureYAxisSnapshotFromChart()
        if (snapshot) {
          this._yAxisSnapshot = snapshot
          this._pendingYAxisCapture = false
        }
      }
      this._onDataZoomForYAxis = (event) => {
        this.scheduleRestoreLockedYAxis()
        this.syncLinkedDataZoom(event)
        this.updateDynamicSymbols()
      }
      this.chart.on('finished', this._onChartFinishedForYAxis)
      this.chart.on('datazoom', this._onDataZoomForYAxis)
      this._yAxisLockHandlersReady = true
    },
    teardownYAxisLockHandlers() {
      if (this._yAxisRestoreTimer) {
        cancelAnimationFrame(this._yAxisRestoreTimer)
        this._yAxisRestoreTimer = null
      }
      if (!this.chart || this.chart.isDisposed()) {
        return
      }
      if (this._onChartFinishedForYAxis) {
        this.chart.off('finished', this._onChartFinishedForYAxis)
      }
      if (this._onDataZoomForYAxis) {
        this.chart.off('datazoom', this._onDataZoomForYAxis)
      }
      this._onChartFinishedForYAxis = null
      this._onDataZoomForYAxis = null
      this._yAxisLockHandlersReady = false
    },
    formatTimeAxisLabel(ts) {
      const d = new Date(ts)
      if (Number.isNaN(d.getTime())) {
        return ''
      }
      const pad = (n) => String(n).padStart(2, '0')
      return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
    },
    buildTooltipOption(xAxisType) {
      if (this.chartData.tooltip) {
        const tip = { ...this.chartData.tooltip }
        if (xAxisType === 'time' && !tip.formatter) {
          tip.formatter = (params) => this.formatTimeTooltip(params)
        }
        return tip
      }
      if (xAxisType === 'time') {
        return {
          trigger: 'item',
          triggerOn: 'mousemove',
          confine: true,
          formatter: (params) => this.formatTimeTooltip(params)
        }
      }
      return {
        trigger: 'axis',
        axisPointer: {
          type: 'shadow',
          shadowStyle: {
            color: 'rgba(255,255,255,0.3)'
          }
        }
      }
    },
    formatTimeTooltip(params) {
      if (!params) {
        return ''
      }
      if (!Array.isArray(params)) {
        const timeTs = Array.isArray(params.value) ? params.value[0] : params.name
        const val = Array.isArray(params.value) ? params.value[1] : params.value
        if (val == null || val === '') {
          return ''
        }
        return `${this.formatTimeAxisLabel(timeTs)}<br/>${params.marker}${params.seriesName}：${val}`
      }
      const first = params[0]
      let timeTs = first.axisValue
      if (Array.isArray(first.value)) {
        timeTs = first.value[0]
      }
      let html = `${this.formatTimeAxisLabel(timeTs)}<br/>`
      params.forEach(p => {
        const val = Array.isArray(p.value) ? p.value[1] : p.value
        if (val != null && val !== '') {
          html += `${p.marker}${p.seriesName}：${val}<br/>`
        }
      })
      return html
    },
    buildXAxisOption(xAxisType, xAxisColor, xAxisData) {
      const base = {
        type: xAxisType,
        axisLine: {
          onZero: false,
          lineStyle: {
            color: xAxisColor
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
        }
      }
      if (xAxisType === 'time') {
        const xAxisExtra = this.chartData.xAxis || {}
        const axisLabelExtra = { ...(xAxisExtra.axisLabel || {}) }
        const axisTickExtra = { ...(xAxisExtra.axisTick || {}) }
        if (this.enableDataZoom) {
          delete axisLabelExtra.customValues
          delete axisTickExtra.customValues
        }
        return {
          ...base,
          min: xAxisExtra.min,
          max: xAxisExtra.max,
          axisTick: {
            show: axisTickExtra.show !== false,
            alignWithLabel: true,
            lineStyle: { color: xAxisColor },
            ...axisTickExtra
          },
          axisLabel: {
            color: xAxisColor,
            fontSize: 12,
            rotate: 0,
            ...axisLabelExtra,
            formatter: axisLabelExtra.formatter || ((value) => {
              const d = new Date(value)
              const pad = (n) => String(n).padStart(2, '0')
              const time = `${pad(d.getHours())}:${pad(d.getMinutes())}`
              const seconds = `${time}:${pad(d.getSeconds())}`
              if (d.getMilliseconds() !== 0) {
                return `${seconds}.${String(d.getMilliseconds()).padStart(3, '0')}`
              }
              return d.getSeconds() === 0 ? time : seconds
            })
          }
        }
      }
      return {
        ...base,
        axisLabel: {
          formatter: (value) => {
            const text = value && typeof value === 'object'
              ? (value.value != null ? String(value.value) : '')
              : String(value != null ? value : '')
            return text.split(' ').join('\n')
          }
        },
        data: xAxisData
      }
    },
    reSetOption() {
      const colors = this.chartData.colors
      let legendColor = this.chartData.legend.color
      legendColor = legendColor || '#F1F1F3'
      const legendData = this.chartData.legend.data
      const legendShow = this.chartData.legend.show
      let legendFontSize = this.chartData.legend.fontSize
      legendFontSize = legendFontSize || 12
      let legendFontWeight = this.chartData.legend.fontWeight
      legendFontWeight = legendFontWeight || 'normal'

      let xAxisColor = this.chartData.xAxis.color
      xAxisColor = xAxisColor || '#fff'
      const xAxisData = this.chartData.xAxis.data
      const xAxisType = this.chartData.xAxis.type || 'category'
      const yAxis = this.chartData.yAxis.map(axis => ({
        ...axis,
        axisLine: {
          onZero: false,
          ...(axis.axisLine || {}),
          lineStyle: {
            ...((axis.axisLine && axis.axisLine.lineStyle) || {}),
            color: (axis.axisLine && axis.axisLine.lineStyle && axis.axisLine.lineStyle.color) || ''
          }
        }
      }))
      yAxis.forEach((item, index) => {
        colors.forEach((color, colorIndex) => {
          if (index === colorIndex) {
            if (item.axisLine.lineStyle.color === '') {
              item.axisLine.lineStyle.color = color
            }
          }
        })
      })
      this.applyLockedYAxisRange(yAxis)
      const seriesList = normalizeChartSeries(
        JSON.parse(JSON.stringify(this.chartData.series || []))
      )
      if (!seriesList.length) {
        return
      }
      this.applyEmptyYAxisRange(yAxis, seriesList)
      seriesList.forEach((item, index) => {
        legendData.forEach((legend, legendIndex) => {
          if (index === legendIndex) {
            item.name = legend
          }
        })
        if (xAxisType === 'time' && item.type === 'line') {
          item.showSymbol = item.showSymbol !== false
          item.symbol = item.symbol || 'emptyCircle'
          item.symbolSize = item.symbolSize || 4
          item.sampling = false
          item.emphasis = item.emphasis || {
            focus: 'self',
            scale: 1.2,
            itemStyle: {
              borderWidth: 2
            }
          }
          item.connectNulls = false
        }
      })

      let grid = {
        top: 50,
        left: '10%',
        right: '10%',
        bottom: 60,
        textStyle: {
          color: '#fff'
        }
      }
      if (this.chartData.grid) {
        grid = { ...this.chartData.grid }
      }
      if (this.enableDataZoom && this.showDataZoomSlider) {
        grid.bottom = Math.max(grid.bottom || 0, 48)
      }
      const tooltip = this.buildTooltipOption(xAxisType)
      const dataZoom = this.buildDataZoomOption(xAxisType, seriesList)

      this.option = {
        animation: this.chartData.animation !== false,
        color: colors,
        tooltip,
        legend: { // 图例组件
          show: legendShow,
          top: 0,
          itemWidth: 14,
          itemHeight: 5,
          itemGap: 13,
          data: legendData,
          right: '25%',
          textStyle: {
            fontSize: legendFontSize,
            fontWeight: legendFontWeight,
            color: legendColor
          }
        },
        grid,
        xAxis: this.buildXAxisOption(xAxisType, xAxisColor, xAxisData),
        yAxis,
        series: seriesList
      }
      if (dataZoom) {
        this.option.dataZoom = dataZoom
      }
      if (!this.chart || this.chart.isDisposed()) {
        return
      }
      safeEchartsSetOption(this.chart, this.option, { notMerge: false })
      this.setupYAxisLockHandlers()
      this.registerDataZoomLink()
      this._dynamicSymbolsVisible = null
      this.updateDynamicSymbols()
    },
    initChart() {
      const dom = document.getElementById(this.id)
      this.chart = markRaw(this.$echarts.init(dom))
      this._yAxisSnapshot = null
      this._pendingYAxisCapture = false
      this._yAxisRestoring = false
      this._yAxisRestoreTimer = null
      this._yAxisLockHandlersReady = false
      this._registeredDataZoomGroup = ''
      this._receivingLinkedDataZoom = false
      this._dynamicSymbolsVisible = null
      this.reSetOption()
    }
  }
}
</script>
