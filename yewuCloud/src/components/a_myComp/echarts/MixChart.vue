<template>
  <div :id="id" :class="className" :style="{height:height,width:width}" />
</template>

<script>
// 普通折线图
import { markRaw } from 'vue'
import resize from './mixins/resize.js'

const DEFAULT_VISIBLE_POINT_COUNT = 12
const VALUE_LABEL_MIN_GAP = 64
const VALUE_LABEL_PADDING = 44
const VALUE_LABEL_MEASURE_LIMIT = 200

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
          title: {
            text: '标题名称' // 曲线图标题名称
          },
          legend: {
            data: ['曲线2'] // 曲线标题
          },
          xAxis: { // 横轴
            data: ['13:00', '13:05', '13:10', '13:15', '13:20', '13:25', '13:30', '13:35', '13:40', '13:45', '13:50', '13:55']
          },
          yAxis: [ // 纵轴
            { name: '(%)' }
          ],
          showSeriesLable: 'true', // 是否显示曲线上的文字，为字符串'true'时显示
          series: [{ // 纵坐标数据
            name: '',
            data: [120, 110, 125, 145, 122, 165, 122, 220, 182, 191, 134, 150],
            color: 'rgba(122,247,190)', // 线和文字颜色
            limitValue: 1, // 限值
            fixed: 2 // 显示数据小数点保留位数
          }]
        }
      }
    }
  },
  data() {
    return {
      chart: null,
      option: {},
      resetTimer: null
    }
  },
  watch: {
    dataChangeKey() {
      this.reSetOption()
    },
    chartData: {
      handler(newValue, oldValue) {
        this.reSetOption()
      },
      deep: true // 设为true时监听对象属性值变化
    }
  },
  mounted() {
    this.initChart()
    window.addEventListener('resize', this.handleResize)
  },
  beforeUnmount() {
    if (this.resetTimer) {
      clearTimeout(this.resetTimer)
      this.resetTimer = null
    }
    window.removeEventListener('resize', this.handleResize)
    if (!this.chart) {
      return
    }
    this.chart.off('datazoom', this.handleDataZoom)
    this.chart.dispose()
    this.chart = null
  },
  methods: {
    normalizePoint(raw) {
      if (raw == null) {
        return null
      }
      if (typeof raw === 'object') {
        if (raw.value !== undefined && raw.value !== null && raw.value !== '') {
          const num = Number(raw.value)
          return Number.isNaN(num) ? raw.value : num
        }
        return null
      }
      if (raw === '') {
        return null
      }
      const num = Number(raw)
      return Number.isNaN(num) ? raw : num
    },
    alignSeriesData(rawData, xAxisLen) {
      const source = Array.isArray(rawData) ? rawData : []
      if (xAxisLen <= 0) {
        return []
      }
      const aligned = []
      for (let i = 0; i < xAxisLen; i++) {
        aligned.push(this.normalizePoint(source[i]))
      }
      return aligned
    },
    getValidNumber(value, fallback) {
      const num = Number(value)
      return Number.isFinite(num) ? num : fallback
    },
    getDataZoomRange(xAxisLen) {
      if (!this.chart || xAxisLen <= 0) {
        return {
          start: 0,
          end: 100,
          startIndex: 0,
          endIndex: Math.max(xAxisLen - 1, 0)
        }
      }
      let option = null
      try {
        option = this.chart.getOption()
      } catch (e) {
        option = null
      }
      const dataZoomList = Array.isArray(option?.dataZoom) ? option.dataZoom : []
      const dataZoom = dataZoomList.find(item => {
        const axisIndex = item?.xAxisIndex
        return axisIndex == null || axisIndex === 0 || (Array.isArray(axisIndex) && axisIndex.includes(0))
      })
      const start = Math.max(0, Math.min(this.getValidNumber(dataZoom?.start, 0), 100))
      const end = Math.max(0, Math.min(this.getValidNumber(dataZoom?.end, 100), 100))
      const rangeStart = Math.min(start, end)
      const rangeEnd = Math.max(start, end)
      const startIndex = Math.max(0, Math.min(Math.floor(xAxisLen * rangeStart / 100), xAxisLen - 1))
      const endIndex = Math.max(startIndex, Math.min(Math.ceil(xAxisLen * rangeEnd / 100) - 1, xAxisLen - 1))
      return {
        start: rangeStart,
        end: rangeEnd,
        startIndex,
        endIndex
      }
    },
    getAdaptiveValueLabelGap(series, range) {
      const canvas = typeof document !== 'undefined' ? document.createElement('canvas') : null
      const context = canvas?.getContext('2d') || null
      if (context) {
        context.font = '12px Arial'
      }
      let maxLabelWidth = 0
      const visibleCount = Math.max(range.endIndex - range.startIndex + 1, 1)
      const measureStep = Math.max(Math.floor(visibleCount / VALUE_LABEL_MEASURE_LIMIT), 1)
      series.forEach(item => {
        const data = Array.isArray(item.data) ? item.data : []
        for (let index = range.startIndex; index <= range.endIndex; index += measureStep) {
          const labelText = this.formatSeriesValue(item, data[index])
          if (!labelText) {
            continue
          }
          const textWidth = context ? context.measureText(labelText).width : labelText.length * 7
          maxLabelWidth = Math.max(maxLabelWidth, textWidth)
        }
        const endLabelText = this.formatSeriesValue(item, data[range.endIndex])
        if (endLabelText) {
          const endTextWidth = context ? context.measureText(endLabelText).width : endLabelText.length * 7
          maxLabelWidth = Math.max(maxLabelWidth, endTextWidth)
        }
      })
      return Math.max(Math.ceil(maxLabelWidth) + VALUE_LABEL_PADDING, VALUE_LABEL_MIN_GAP)
    },
    getValueLabelDensity(xAxisLen, range = null, series = []) {
      range = range || this.getDataZoomRange(xAxisLen)
      const chartWidth = this.chart?.getWidth?.() || 0
      const plotWidth = chartWidth > 0 ? chartWidth * 0.9 : 900
      const labelGap = this.getAdaptiveValueLabelGap(series, range)
      const targetLabelCount = Math.max(Math.floor(plotWidth / labelGap), 2)
      return {
        ...range,
        targetLabelCount,
        chartWidth,
        labelGap
      }
    },
    getLabelIndexSet(series, xAxisLen, density, valueLabelMode) {
      const dataList = series.map(item => ({
        item,
        data: this.alignSeriesData(item.data, xAxisLen)
      }))
      const validIndices = []
      for (let index = density.startIndex; index <= density.endIndex; index++) {
        if (dataList.some(({ data }) => {
          const value = this.normalizePoint(data[index])
          return value !== null && value !== ''
        })) {
          validIndices.push(index)
        }
      }
      if (valueLabelMode === 'all' || validIndices.length <= density.targetLabelCount) {
        return new Set(validIndices)
      }

      const selected = new Set([validIndices[0], validIndices[validIndices.length - 1]])
      const targetCount = Math.min(density.targetLabelCount, validIndices.length)
      const minimumIndexGap = (validIndices.length - 1) / Math.max(targetCount - 1, 1)
      const semanticCandidates = []

      dataList.forEach(({ item, data }) => {
        for (let index = density.startIndex + 1; index < density.endIndex; index++) {
          const current = this.normalizePoint(data[index])
          if (current === null || current === '') {
            continue
          }
          let score = 0
          const previous = this.normalizePoint(data[index - 1])
          const next = this.normalizePoint(data[index + 1])
          const currentText = this.formatSeriesValue(item, current)
          const previousText = this.formatSeriesValue(item, previous)
          if (currentText && currentText !== previousText) {
            score += 2
          }
          const currentNumber = Number(current)
          const previousNumber = Number(previous)
          const nextNumber = Number(next)
          if (Number.isFinite(currentNumber) && Number.isFinite(previousNumber) && Number.isFinite(nextNumber) &&
            ((currentNumber >= previousNumber && currentNumber >= nextNumber) ||
              (currentNumber <= previousNumber && currentNumber <= nextNumber))) {
            score += 1
          }
          const currentColor = data[index]?.itemStyle?.color
          const previousColor = data[index - 1]?.itemStyle?.color
          if (currentColor && currentColor !== previousColor) {
            score += 3
          }
          if (score > 0) {
            semanticCandidates.push({ index, score })
          }
        }
      })

      const candidateScores = new Map()
      semanticCandidates.forEach(candidate => {
        candidateScores.set(candidate.index, Math.max(candidateScores.get(candidate.index) || 0, candidate.score))
      })
      const candidates = Array.from(candidateScores.entries())
        .map(([index, score]) => ({ index, score }))
        .sort((left, right) => right.score - left.score || left.index - right.index)
      candidates.forEach(candidate => {
        if (selected.size >= targetCount) {
          return
        }
        const farEnough = Array.from(selected).every(index => Math.abs(index - candidate.index) >= minimumIndexGap * 0.55)
        if (farEnough) {
          selected.add(candidate.index)
        }
      })

      for (let slot = 1; selected.size < targetCount - 1; slot++) {
        const desired = validIndices[Math.round(slot * (validIndices.length - 1) / Math.max(targetCount - 1, 1))]
        if (desired === undefined) {
          break
        }
        const nearest = validIndices.reduce((best, index) => {
          if (selected.has(index)) return best
          if (best === null || Math.abs(index - desired) < Math.abs(best - desired)) return index
          return best
        }, null)
        if (nearest === null) {
          break
        }
        selected.add(nearest)
      }
      return selected
    },
    formatSeriesValue(item, rawValue) {
      const limitValue = item.limitValue ? item.limitValue : 1
      const value = this.normalizePoint(rawValue)
      if (value == null || value === '') {
        return ''
      }
      if (typeof value === 'number' && value > 0) {
        if (item.fixed) {
          const formatted = (value / limitValue).toFixed(item.fixed)
          return formatted > 0 ? formatted : String(value / limitValue)
        }
        return String(value / limitValue)
      }
      return String(value)
    },
    buildLineSeries(item, xAxisLen, seriesLableShow, labelIndexSet) {
      const rawStyle = (Array.isArray(item.data) ? item.data : []).map((point, index) => {
        if (point && typeof point === 'object' && point.itemStyle) {
          return point.itemStyle
        }
        return null
      })
      const data = this.alignSeriesData(item.data, xAxisLen)
      const seriesData = data.map((value, index) => {
        const dataItem = { value }
        const style = rawStyle[index]
        if (style) {
          dataItem.itemStyle = style
        }
        if (seriesLableShow && labelIndexSet.has(index)) {
          const labelText = this.formatSeriesValue(item, value)
          if (labelText !== '') {
            dataItem.label = {
              show: true,
              position: 'top',
              formatter: labelText
            }
          }
        }
        return dataItem
      })
      return {
        name: item.name || '',
        type: 'line',
        coordinateSystem: 'cartesian2d',
        xAxisIndex: 0,
        yAxisIndex: 0,
        sampling: false,
        symbolSize: 10,
        symbol: 'circle',
        connectNulls: false,
        itemStyle: {
          color: item.color ? item.color : 'rgba(122,247,190)'
        },
        label: {
          show: false,
          position: 'top'
        },
        labelLayout: {
          hideOverlap: false
        },
        data: seriesData
      }
    },
    reSetOption() {
      if (!this.chart) {
        return
      }
      if (this.resetTimer) {
        clearTimeout(this.resetTimer)
      }
      this.resetTimer = setTimeout(() => {
        this.resetTimer = null
        this.applyOption()
      }, 30)
    },
    applyOption() {
      if (!this.chart) {
        return
      }
      const showSeriesLable = this.chartData.showSeriesLable ? this.chartData.showSeriesLable : 'true' // 是否显示曲线上的文字
      const seriesLableShow = showSeriesLable === 'true' // 是否显示曲线上的文字
      const valueLabelMode = this.chartData.valueLabelMode || 'adaptive'
      const xAxisData = (this.chartData.xAxis?.data || []).map(item => {
        if (item && typeof item === 'object' && item.value != null) {
          return item.value
        }
        return item
      }).filter(item => item != null && item !== '')
      const hasXAxisData = xAxisData.length > 0
      const displayXAxis = hasXAxisData ? xAxisData : ['暂无数据']
      const xAxisLen = displayXAxis.length
      const hasDataOverflow = hasXAxisData && xAxisLen > DEFAULT_VISIBLE_POINT_COUNT
      const zoomRange = hasDataOverflow
        ? this.getDataZoomRange(xAxisLen)
        : { start: 0, end: 100, startIndex: 0, endIndex: Math.max(xAxisLen - 1, 0) }
      const series = []
      const sourceSeries = this.chartData.series || []
      const labelDensity = this.getValueLabelDensity(xAxisLen, zoomRange, sourceSeries)
      const labelIndexSet = this.getLabelIndexSet(sourceSeries, xAxisLen, labelDensity, valueLabelMode)
      if (hasXAxisData && sourceSeries.length > 0) {
        sourceSeries.forEach(item => {
          series.push(this.buildLineSeries(item, xAxisLen, seriesLableShow, labelIndexSet))
        })
      }
      if (series.length === 0) {
        series.push({
          name: '',
          type: 'line',
          coordinateSystem: 'cartesian2d',
          xAxisIndex: 0,
          yAxisIndex: 0,
          sampling: false,
          symbolSize: 0,
          showSymbol: false,
          silent: true,
          lineStyle: { opacity: 0 },
          data: [null]
        })
      }

      this.option = {
        backgroundColor: this.backgroundColor,
        title: {
          top: 20,
          text: this.chartData.title.text,
          textStyle: {
            fontWeight: 'normal',
            fontSize: 16,
            color: '#F1F1F3'
          },
          left: '30'
        },
        tooltip: { // 提示框组件
          trigger: 'axis', // 触发类型
          axisPointer: { // 坐标轴指示器配置项
            lineStyle: {
              color: '#57617B' // 提示线的颜色
            }
          },
          formatter: (params) => {
            let res = // 字符串形式的html标签会被echarts转换渲染成数据，这个res主要是画的tooltip里的上部分的标题部分
              "<div style='margin-bottom:5px;padding:0 12px;width:100%;height:24px;line-height:24px;'><p>" +
              params[0].name +
              ' </p></div>'
            for (let i = 0; i < params.length; i++) {
              const item = params[i]
              // 因为是个数组，所以要遍历拿到里面的数据，并加入到tooltip的数据内容部分里面去
              res += `<div style="color: #000;font-size: 14px; padding:0 12px;line-height: 24px">
                  <span style="display:inline-block;margin-right:5px;border-radius:5px;width:10px;height:10px;background-color:${[
    item.color // 默认是小圆点，我们将其修改成有圆角的正方形，这里用的是模板字符串。并拿到对应颜色、名字、数据
  ]};"></span>
                  ${item.seriesName}：${item.value}
                </div>`
            }
            return res // 经过这么一加工，最终返回出去并渲染，最终就出现了我们所看的效果
          }
        },
        legend: { // 图例组件
          top: 20,
          itemWidth: 14,
          itemHeight: 5,
          itemGap: 13,
          data: this.chartData.legend.data,
          right: '4%',
          textStyle: {
            fontSize: 12,
            color: '#F1F1F3'
          }
        },
        grid: {
          top: 50,
          left: '5%',
          right: '5%',
          bottom: hasDataOverflow ? 62 : 40,
          textStyle: {
            color: '#fff'
          }
        },
        calculable: true,
        xAxis: [{
          type: 'category',
          axisLine: {
            lineStyle: {
              color: '#90979c'
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
          axisLabel: { // 横轴坐标点名称
            interval: 'auto',
            hideOverlap: true,
            formatter: (value) => {
              const label = value && typeof value === 'object' ? value.value : value
              if (!label || typeof label !== 'string') {
                return ''
              }
              return label.split(' ').join('\n')
            }
          },
          data: displayXAxis
        }],
        yAxis: [{
          type: 'value',
          name: (this.chartData.yAxis && this.chartData.yAxis[0] && this.chartData.yAxis[0].name) || '',
          nameLocation: 'end',
          splitLine: {
            show: false
          },
          axisLine: {
            lineStyle: {
              color: '#90979c'
            }
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
        }],
        dataZoom: hasDataOverflow ? [{
          type: 'slider',
          show: true,
          xAxisIndex: [0],
          filterMode: 'none',
          left: '5%',
          right: '5%',
          height: 16,
          bottom: 8,
          start: zoomRange.start,
          end: zoomRange.end,
          showDetail: false,
          showDataShadow: false,
          brushSelect: false,
          handleIcon: 'path://M306.1,413c0,2.2-1.8,4-4,4h-59.8c-2.2,0-4-1.8-4-4V200.8c0-2.2,1.8-4,4-4h59.8c2.2,0,4,1.8,4,4V413z',
          handleSize: '100%',
          handleStyle: {
            color: '#d3dee5'
          },
          backgroundColor: '#eef4fb',
          fillerColor: 'rgba(64, 158, 255, 0.2)',
          textStyle: {
            color: '#fff'
          },
          borderColor: '#b8c9db'
        }, {
          type: 'inside',
          xAxisIndex: [0],
          filterMode: 'none',
          start: zoomRange.start,
          end: zoomRange.end,
          zoomOnMouseWheel: true,
          moveOnMouseMove: true,
          moveOnMouseWheel: false
        }] : [],
        series
      }
      try {
        this.chart.clear()
        this.chart.setOption(this.option, true)
      } catch (e) {
        console.warn('[MixChart] setOption failed:', e)
      }
    },
    initChart() {
      this.chart = markRaw(this.$echarts.init(document.getElementById(this.id)))
      this.chart.on('datazoom', this.handleDataZoom)
      this.reSetOption()
    },
    handleDataZoom() {
      this.reSetOption()
    },
    handleResize() {
      if (!this.chart) {
        return
      }
      this.chart.resize()
      this.reSetOption()
    }
  }
}
</script>
