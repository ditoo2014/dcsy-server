<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
// 采集云使用与原项目一致的图表版本，保留历史曲线的样式和交互。
import * as echarts from 'echarts-collection/core'
import { createContainerResizeWatcher } from '../../../utils/containerResize'

import { LineChart, ScatterChart } from 'echarts-collection/charts'
import {
  DataZoomComponent,
  GraphicComponent,
  GridComponent,
  MarkPointComponent,
  TooltipComponent
} from 'echarts-collection/components'
import { CanvasRenderer } from 'echarts-collection/renderers'
import { fetchGatewayHistoryCount, fetchGatewayHistoryPaged } from '../../api/gatewayRealtime'

echarts.use([
  LineChart,
  ScatterChart,
  GridComponent,
  TooltipComponent,
  DataZoomComponent,
  MarkPointComponent,
  GraphicComponent,
  CanvasRenderer
])

const props = defineProps({
  title: { type: String, required: true },
  gatewayId: { type: String, required: true },
  token: { type: String, required: true },
  dataId: { type: String, default: undefined },
  globalRefreshEnabled: { type: Boolean },
  darkMode: { type: Boolean }
})

const chartRef = ref(null)
let chart = null
const autoRefresh = ref(true)
const loading = ref(false)
const historyError = ref('')
let timer = null
let requestSequence = 0
let disposed = false

// 历史曲线只使用时序接口的数据，不能用实时值生成模拟历史。
const fullSeries = ref([])
const historyPage = ref(1)
const pageSize = ref(30)
const totalCount = ref(0)
const timeRange = ref([])
const suppressPageWatch = ref(false)
const isPollingBusy = ref(false)
const lastKnownTotalPages = ref(1)
const autoRefreshBeforeGlobalPause = ref(true)
const defaultPickerTime = [
  new Date(2000, 0, 1, 0, 0, 0),
  new Date(2000, 0, 1, 23, 59, 59)
]

const xData = computed(() => fullSeries.value.map(x => x.time))
const yData = computed(() => fullSeries.value.map(x => x.value))
const rangePopperClass = computed(() =>
  props.darkMode ? 'history-range-popper history-range-popper--dark' : 'history-range-popper'
)

function normalizeTime(ts) {
  const n = Number(ts)
  if (Number.isFinite(n)) {
    const ms = n < 10_000_000_000 ? n * 1000 : n
    const d = new Date(ms)
    const yyyy = d.getFullYear()
    const mm = String(d.getMonth() + 1).padStart(2, '0')
    const dd = String(d.getDate()).padStart(2, '0')
    const hh = String(d.getHours()).padStart(2, '0')
    const mi = String(d.getMinutes()).padStart(2, '0')
    const ss = String(d.getSeconds()).padStart(2, '0')
    return `${yyyy}-${mm}-${dd} ${hh}:${mi}:${ss}`
  }
  return String(ts ?? '')
}

function normalizeValue(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

function parseSeries(raw) {
  if (raw?.return === null) return { rows: [], total: 0 }
  const ret = raw?.return ?? raw
  const times = ret?.times ?? ret?.data?.times ?? ret?.result?.times
  const values = ret?.values ?? ret?.data?.values ?? ret?.result?.values
  const status =
    ret?.status ??
    ret?.statuses ??
    ret?.statuss ??
    ret?.data?.status ??
    ret?.data?.statuses ??
    ret?.data?.statuss ??
    ret?.result?.status ??
    ret?.result?.statuses ??
    ret?.result?.statuss
  // Go 的空切片可能返回 null；只有明确的零记录允许没有数组。
  const countValue = ret?.total ?? ret?.count ?? raw?.total ?? raw?.count
  if (Number(countValue) === 0 && times == null && values == null) return { rows: [], total: 0 }
  if (!Array.isArray(times) || !Array.isArray(values) || times.length !== values.length) {
    throw new Error('历史数据返回格式异常，请检查时序查询接口')
  }
  const rows = []
  const size = Math.min(times.length, values.length)
  for (let i = 0; i < size; i++) {
    const tNum = Number(times[i])
    const rawTs = Number.isFinite(tNum) ? (tNum < 10_000_000_000 ? tNum * 1000 : tNum) : Date.now() + i
    const st = Array.isArray(status)
      ? status[i]
      : (
        typeof values[i] === 'object'
          ? (values[i])?.status ?? (values[i])?.Status ?? ''
          : ''
      )
    rows.push({
      time: normalizeTime(times[i]),
      value: normalizeValue(values[i]),
      status: String(st ?? ''),
      rawTs
    })
  }
  rows.sort((a, b) => Number(a.rawTs ?? 0) - Number(b.rawTs ?? 0))
  const totalRaw = Number(
    ret?.total ?? ret?.count ?? raw?.total ?? raw?.count ?? rows.length
  )
  if (!Number.isInteger(totalRaw) || totalRaw < rows.length) {
    throw new Error('历史数据条数与返回明细不一致，请重新查询')
  }
  const total = totalRaw
  return { rows, total }
}

function parseHistoryCount(raw) {
  if (raw?.return === null) return 0
  const ret = raw?.return ?? raw
  const n = Number(ret?.total ?? ret?.count ?? ret?.num ?? raw?.total ?? raw?.count)
  if (!Number.isInteger(n) || n < 0) throw new Error('历史数据条数返回异常，请检查时序查询接口')
  return n
}

function ensureDefaultTimeRange() {
  if (timeRange.value.length === 2) return
  const end = Date.now()
  const start = end - 24 * 60 * 60 * 1000
  timeRange.value = [new Date(start), new Date(end)]
}

function toUnixSeconds(d) {
  return Math.floor(d.getTime() / 1000)
}

function normalizeRangeEndToDayLastSecond(range) {
  if (range.length !== 2) return range
  const [start, end] = range
  if (!(start instanceof Date) || !(end instanceof Date)) return range
  const isMidnight =
    end.getHours() === 0 &&
    end.getMinutes() === 0 &&
    end.getSeconds() === 0
  if (!isMidnight) return range
  const fixedEnd = new Date(end)
  fixedEnd.setHours(23, 59, 59, 0)
  return [start, fixedEnd]
}

async function fetchHistorySeries({ latest = false } = {}) {
  // 切换时间、分页或点位后，只允许最后一次请求更新当前曲线。
  const requestId = ++requestSequence
  if (disposed) return
  if (!props.token || !props.gatewayId || !props.dataId) {
    fullSeries.value = []
    totalCount.value = 0
    loading.value = false
    historyError.value = !props.dataId ? '缺少数据点编号，无法查询历史数据' : '缺少登录或网关信息，无法查询历史数据'
    renderChart()
    return
  }
  ensureDefaultTimeRange()
  const [startDt, endDt] = timeRange.value
  if (!(startDt instanceof Date) || !(endDt instanceof Date)) return
  const dateStart = toUnixSeconds(startDt)
  const dateEnd = toUnixSeconds(endDt)
  if (!Number.isFinite(dateStart) || !Number.isFinite(dateEnd)) return
  const requestedPage = historyPage.value
  const requestedSize = pageSize.value
  const query = {
    token: props.token,
    gatewayId: props.gatewayId,
    dataId: String(props.dataId),
    dateStart,
    dateEnd
  }
  loading.value = true
  historyError.value = ''
  renderChart()
  try {
    const requestPage = async(uiPageNo, pages) => {
      // 后端页序为“1=最新页”，前端页序为“1=最旧页，最后一页=最新页”
      const backendPageNo = Math.max(1, pages - uiPageNo + 1)
      return fetchGatewayHistoryPaged({
        ...query,
        pageNo: backendPageNo,
        pageSize: requestedSize
      })
    }

    const { data: countData } = await fetchGatewayHistoryCount(query)
    if (requestId !== requestSequence) return
    const count = parseHistoryCount(countData)
    const countPages = Math.max(1, Math.ceil(count / requestedSize))
    const followLatest = latest || (autoRefresh.value && countPages > lastKnownTotalPages.value)
    let uiPage = followLatest ? countPages : Math.min(Math.max(1, requestedPage), countPages)
    const { data } = await requestPage(uiPage, countPages)
    if (requestId !== requestSequence) return
    let parsed = parseSeries(data)
    let actualPages = Math.max(1, Math.ceil(parsed.total / requestedSize))
    // 两次查询之间可能新增或删除记录，以明细接口的实际总数校准页码。
    const correctedPage = followLatest ? actualPages : Math.min(uiPage, actualPages)
    if (parsed.total > 0 && countPages - uiPage !== actualPages - correctedPage) {
      const { data: correctedData } = await requestPage(correctedPage, actualPages)
      if (requestId !== requestSequence) return
      parsed = parseSeries(correctedData)
      actualPages = Math.max(1, Math.ceil(parsed.total / requestedSize))
    }
    uiPage = followLatest ? actualPages : Math.min(correctedPage, actualPages)
    if (parsed.total > 0 && parsed.rows.length === 0) {
      throw new Error('历史数据条数与当前页不一致，请重新查询')
    }
    // 零记录也是有效结果，必须同时清空曲线和数量，不能保留上次的计数。
    fullSeries.value = parsed.rows
    totalCount.value = parsed.total
    lastKnownTotalPages.value = actualPages
    suppressPageWatch.value = true
    historyPage.value = uiPage
    suppressPageWatch.value = false
  } catch (error) {
    if (requestId !== requestSequence) return
    fullSeries.value = []
    totalCount.value = 0
    lastKnownTotalPages.value = 1
    suppressPageWatch.value = true
    historyPage.value = 1
    suppressPageWatch.value = false
    historyError.value = error?.code === 'ECONNABORTED' || error?.code === 'ETIMEDOUT'
      ? '历史数据查询超时，请稍后重试'
      : error?.code === 'ERR_NETWORK' || error?.message === 'Network Error' || (error?.isAxiosError && !error?.response)
        ? '无法连接历史数据接口，请检查网络或服务状态'
        : error?.response?.data?.msg || error?.message || '历史数据查询失败，请稍后重试'
  } finally {
    if (requestId === requestSequence && !disposed) {
      loading.value = false
      await nextTick()
      if (requestId === requestSequence && !disposed) renderChart()
    }
  }
}

function stopPolling() {
  if (timer != null) {
    window.clearInterval(timer)
    timer = null
  }
}

function startPolling(refreshImmediately = true) {
  if (props.globalRefreshEnabled === false) {
    stopPolling()
    return
  }
  stopPolling()
  if (refreshImmediately) void refreshLatestByAuto()
  timer = window.setInterval(() => {
    void refreshLatestByAuto()
  }, 3000)
}

async function refreshLatestByAuto() {
  if (isPollingBusy.value || loading.value) return
  if (props.globalRefreshEnabled === false) return
  if (!props.dataId) return
  isPollingBusy.value = true
  try {
    // 自动刷新时把结束时间推进到“当前时刻”，否则会一直查询同一时间片
    if (timeRange.value.length === 2) {
      const [start, end] = timeRange.value
      if (start instanceof Date && end instanceof Date) {
        const now = new Date()
        if (end.getTime() < now.getTime() - 1000) {
          timeRange.value = [start, now]
        }
      }
    }
    // 自动刷新始终拉到最后一页，保证展示最新 30 条
    await resetToLatestPageAndFetch()
  } finally {
    isPollingBusy.value = false
  }
}

function renderChart() {
  if (disposed || !chartRef.value) return
  if (!chart) chart = echarts.init(chartRef.value)
  const isDark = props.darkMode === true
  const axisLabelColor = isDark ? '#a8bfd8' : '#8b95a7'
  const axisLineColor = isDark ? '#45607f' : '#d8dee9'
  const splitLineColor = isDark ? 'rgba(148, 163, 184, 0.35)' : '#edf2f7'
  const zoomBg = isDark ? 'rgba(15, 29, 48, 0.92)' : 'rgba(248, 251, 255, 0.92)'
  const zoomBorder = isDark ? 'rgba(71, 95, 124, 0.8)' : 'rgba(233, 238, 245, 0.9)'

  const pts = fullSeries.value
  const pointCount = yData.value.length
  const denseLevel = pointCount >= 5000 ? 3 : pointCount >= 1500 ? 2 : pointCount >= 600 ? 1 : 0
  const lineWidth = denseLevel === 3 ? 1.4 : denseLevel === 2 ? 1.8 : 2.4
  const markPointSize = denseLevel >= 2 ? 5 : 6
  const abnormalPointSize = denseLevel >= 2 ? 4 : 5
  const statusPointSize = denseLevel >= 2 ? 3 : 4
  const useSmoothLine = denseLevel <= 1
  const samplingMode = denseLevel >= 1 ? 'lttb' : undefined
  const showValueLabels = pointCount > 0 && pointCount <= 50
  const lastIdx = pts.length - 1
  const abnormalScatter = pts
    .map((p, idx) => ({ idx, p }))
    .filter(x => String(x.p.status ?? '') === '-1')
    .map(x => [xData.value[x.idx], yData.value[x.idx]])

  const statusOneScatter = pts
    .map((p, idx) => ({ idx, p }))
    .filter(x => String(x.p.status ?? '') === '1')
    .map(x => [xData.value[x.idx], yData.value[x.idx]])
  chart.setOption({
    /** 透明底，与 history-wrap 浅蓝渐变衔接，避免主图白底与滑块割裂 */
    backgroundColor: 'transparent',
    // 仅在原图表内部提示加载、空数据或失败，正常曲线的布局和配色保持原版。
    graphic: [{
      id: 'history-state',
      type: 'text',
      left: 'center',
      top: 'middle',
      silent: true,
      ignore: pointCount > 0,
      style: {
        text: loading.value ? '正在加载历史数据…' : historyError.value || '所选时间范围内暂无历史数据',
        fill: axisLabelColor,
        fontSize: 12,
        lineHeight: 20,
        width: Math.max(160, chartRef.value.clientWidth - 96),
        overflow: 'break',
        align: 'center'
      }
    }],
    grid: { left: 48, right: 22, top: showValueLabels ? 46 : 32, bottom: 56 },
    tooltip: {
      trigger: 'axis',
      backgroundColor: 'rgba(16, 24, 40, 0.92)',
      borderWidth: 0,
      textStyle: { color: '#e6edf7' }
    },
    dataZoom: [
      {
        type: 'slider',
        xAxisIndex: 0,
        filterMode: 'filter',
        realtime: true,
        zoomLock: false,
        /** 在滑条背景上按住拖动可框选放大区间 */
        brushSelect: true,
        brushStyle: {
          color: 'rgba(19, 194, 194, 0.08)',
          borderColor: 'rgba(19, 194, 194, 0.35)',
          borderWidth: 1
        },
        minSpan: 4,
        maxSpan: 100,
        start: 0,
        end: 100,
        showDetail: true,
        showDataShadow: 'auto',
        /** 与主图同色系的轻量条带，贴近 history-wrap 背景 */
        height: 22,
        bottom: 6,
        borderRadius: 10,
        borderColor: zoomBorder,
        backgroundColor: zoomBg,
        handleIcon: 'path://M-8,1 h3 v18 h-3 Z M5,1 h3 v18 h-3 Z',
        handleSize: 12,
        handleStyle: {
          borderRadius: 2,
          color: '#ffffff',
          borderColor: '#87e8de',
          borderWidth: 1,
          shadowBlur: 0
        },
        emphasis: {
          handleStyle: {
            color: '#ffffff',
            borderColor: '#13c2c2',
            shadowBlur: 4,
            shadowColor: 'rgba(19, 194, 194, 0.2)'
          },
          moveHandleStyle: {
            color: 'rgba(19, 194, 194, 0.35)',
            borderColor: 'rgba(19, 194, 194, 0.45)'
          }
        },
        moveHandleSize: 9,
        moveHandleStyle: {
          color: 'rgba(19, 194, 194, 0.22)',
          borderColor: 'rgba(19, 194, 194, 0.4)',
          borderWidth: 1
        },
        fillerColor: 'rgba(19, 194, 194, 0.1)',
        /** 缩略波形与主曲线同一套青绿→蓝渐变 */
        dataBackground: {
          lineStyle: {
            width: 1.2,
            cap: 'round',
            /** 与主折线同主色（缩略线用单色更稳，渐变见下方 area） */
            color: '#13c2c2'
          },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(19, 194, 194, 0.28)' },
              { offset: 1, color: 'rgba(22, 119, 255, 0.04)' }
            ])
          }
        },
        selectedDataBackground: {
          lineStyle: { color: '#1677ff', width: 1.2, cap: 'round' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(19, 194, 194, 0.38)' },
              { offset: 1, color: 'rgba(22, 119, 255, 0.1)' }
            ])
          }
        },
        textStyle: { fontSize: 11, color: axisLabelColor, fontWeight: 400 }
      },
      {
        type: 'inside',
        xAxisIndex: 0,
        filterMode: 'filter',
        /** 曲线区内滚轮缩放；按住左键拖动平移。底部滑条：拖左右手柄、拖中间选区、在背景上拖选框选放大 */
        zoomOnMouseWheel: true,
        moveOnMouseMove: true,
        moveOnMouseWheel: false
      }
    ],
    xAxis: {
      type: 'category',
      data: xData.value,
      boundaryGap: false,
      axisLabel: {
        color: axisLabelColor,
        fontSize: 11,
        formatter: (value) => {
          const s = String(value ?? '')
          return s.includes(' ') ? s.replace(' ', '\n') : s
        }
      },
      axisLine: { lineStyle: { color: axisLineColor }},
      axisTick: { show: false }
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: axisLabelColor, fontSize: 11 },
      axisLine: { show: false },
      splitLine: { lineStyle: { color: splitLineColor, type: 'dashed' }}
    },
    series: [
      {
        type: 'line',
        smooth: useSmoothLine,
        sampling: samplingMode,
        showSymbol: showValueLabels,
        symbol: 'circle',
        symbolSize: showValueLabels ? 4 : 0,
        data: yData.value,
        label: {
          show: showValueLabels,
          position: 'top',
          distance: 6,
          color: isDark ? '#c9d7e8' : '#52627a',
          fontSize: 10,
          fontWeight: 500,
          formatter: (params) => {
            const value = Array.isArray(params.value) ? params.value[1] : params.value
            if (value == null || value === '') return ''
            const num = Number(value)
            return Number.isFinite(num) ? String(Number(num.toFixed(4))) : String(value)
          }
        },
        labelLayout: {
          hideOverlap: true
        },
        lineStyle: {
          width: lineWidth,
          color: new echarts.graphic.LinearGradient(0, 0, 1, 0, [
            { offset: 0, color: '#13c2c2' },
            { offset: 1, color: '#1677ff' }
          ])
        },
        areaStyle: {
          opacity: 0.16,
          color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
            { offset: 0, color: 'rgba(19, 194, 194, 0.35)' },
            { offset: 1, color: 'rgba(22, 119, 255, 0.02)' }
          ])
        },
        markPoint: {
          symbol: 'circle',
          symbolSize: markPointSize,
          itemStyle: { color: '#16c28f', borderColor: '#ffffff', borderWidth: 2 },
          // 空结果时显式移除原有末点标记，避免 ECharts 合并配置后残留旧点。
          data: lastIdx >= 0 ? [{ coord: [xData.value[lastIdx], yData.value[lastIdx]] }] : []
        }
      },
      {
        type: 'scatter',
        data: abnormalScatter,
        tooltip: { show: false },
        symbolSize: abnormalPointSize,
        itemStyle: {
          color: '#ff4d4f',
          borderColor: '#ffffff',
          borderWidth: 1.5
        },
        z: 6
      },
      {
        type: 'scatter',
        data: statusOneScatter,
        tooltip: { show: false },
        symbolSize: statusPointSize,
        itemStyle: {
          color: '#13c2c2',
          borderColor: '#ffffff',
          borderWidth: 1.2
        },
        z: 5
      }
    ]
  })
}

// 折叠行、侧栏和分页换行都会改变容器尺寸，不能只依赖窗口 resize。
const chartResizeWatcher = createContainerResizeWatcher(() => chartRef.value, () => {
  if (disposed || !chart) return false
  chart.resize()
})

watch(historyPage, () => {
  if (suppressPageWatch.value) return
  void fetchHistorySeries()
}, { flush: 'sync' })

watch(
  () => autoRefresh.value,
  enabled => {
    if (enabled && props.globalRefreshEnabled !== false) startPolling()
    else stopPolling()
  }
)

watch(
  () => props.globalRefreshEnabled,
  enabled => {
    if (enabled === false) {
      autoRefreshBeforeGlobalPause.value = autoRefresh.value
      autoRefresh.value = false
      stopPolling()
      return
    }
    autoRefresh.value = autoRefreshBeforeGlobalPause.value
    if (autoRefresh.value) startPolling()
  }
)

watch(
  () => [props.dataId, props.gatewayId, props.token],
  () => {
    // 网关或点位切换后，不展示上一个点位的历史曲线。
    fullSeries.value = []
    totalCount.value = 0
    lastKnownTotalPages.value = 1
    void resetToLatestPageAndFetch()
  }
)

watch(
  () => props.darkMode,
  async() => {
    await nextTick()
    renderChart()
  }
)

watch(pageSize, () => {
  void resetToLatestPageAndFetch()
})

function onTimeRangeChange() {
  timeRange.value = normalizeRangeEndToDayLastSecond(timeRange.value)
  void resetToLatestPageAndFetch()
}

async function resetToLatestPageAndFetch() {
  await fetchHistorySeries({ latest: true })
}

onMounted(async() => {
  ensureDefaultTimeRange()
  // 折叠打开默认显示最新 30 条：最新在最后一页
  await resetToLatestPageAndFetch()
  // 每次异步等待后都确认组件仍在，防止离开页面后重启轮询或尺寸监听。
  if (disposed) return
  await nextTick()
  if (disposed) return
  renderChart()
  if (autoRefresh.value && props.globalRefreshEnabled !== false) {
    startPolling(false)
  }
  chartResizeWatcher.start()
})

onBeforeUnmount(() => {
  disposed = true
  requestSequence += 1
  stopPolling()
  chartResizeWatcher.stop()
  chart?.dispose()
  chart = null
})
</script>

<template>
  <div class="history-wrap" :class="{ 'is-dark': props.darkMode }">
    <div class="history-head">
      <div class="history-title">
        {{ title }} 历史曲线
        <span class="history-meta">{{ historyError ? '查询失败' : `共 ${totalCount} 条` }} · 每页 {{ pageSize }} 条</span>
      </div>
      <div class="history-switch">
        <span class="switch-label">自动刷新</span>
        <el-switch v-model="autoRefresh" :loading="loading" :disabled="props.globalRefreshEnabled === false" />
      </div>
    </div>
    <div ref="chartRef" class="history-chart" />
    <div class="history-pager">
      <el-date-picker
        v-model="timeRange"
        type="datetimerange"
        :default-time="defaultPickerTime"
        :popper-class="rangePopperClass"
        range-separator="至"
        start-placeholder="开始时间"
        end-placeholder="结束时间"
        format="YYYY-MM-DD HH:mm:ss"
        size="small"
        class="history-range"
        @change="onTimeRangeChange"
      />
      <el-pagination
        v-model:current-page="historyPage"
        v-model:page-size="pageSize"
        :total="totalCount"
        :page-sizes="[30, 60, 100, 200, 500, 1000, 5000, 10000]"
        layout="total, sizes, prev, pager, next, jumper"
        small
        background
      />
    </div>
  </div>
</template>

<style scoped>

.history-wrap {
  min-width: 0;
  padding: 8px 6px 2px;
  border: 1px solid #e9eef5;
  border-radius: 10px;
  background: linear-gradient(180deg, #f8fbff 0%, #ffffff 55%);
}

.history-title {
  font-size: 13px;
  font-weight: 600;
  color: #344054;
  padding-left: 2px;
}

.history-meta {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 400;
  color: #667085;
}

.history-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  margin-bottom: 4px;
}

.history-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.switch-label {
  font-size: 12px;
  color: #667085;
}

.history-chart {
  width: 100%;
  height: 268px;
}

.history-pager {
  display: flex;
  /* 时间筛选与分页在容器不足时分行，正常桌面保持原排布。 */
  flex-wrap: wrap;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 6px 0 4px;
}

/* 时间范围选择器：更紧凑、更贴合整体风格 */
:deep(.history-range) {
  width: 320px;
  min-width: min(280px, 100%);
  max-width: min(100%, 40vw);
}

:deep(.history-range.el-range-editor.el-input__wrapper) {
  border-radius: 10px;
  box-shadow: none;
  background: rgba(248, 251, 255, 0.92);
  border: 1px solid #e9eef5;
  padding: 0 10px;
}

:deep(.history-range .el-range-input) {
  font-size: 12px;
  color: #344054;
}

:deep(.history-range .el-range-separator) {
  color: #98a2b3;
  padding: 0 6px;
}

:deep(.history-range .el-range__icon) {
  color: #98a2b3;
}

:deep(.history-range.is-active) {
  border-color: rgba(19, 194, 194, 0.45);
}

.history-wrap.is-dark {
  background: linear-gradient(180deg, rgba(19, 37, 61, 0.96) 0%, rgba(15, 29, 48, 0.96) 100%);
  border-color: #2f4968;
}

.history-wrap.is-dark .history-title,
.history-wrap.is-dark .history-meta,
.history-wrap.is-dark .switch-label {
  color: #dbe9ff;
}

.history-wrap.is-dark :deep(.history-range.el-range-editor.el-input__wrapper) {
  background: rgba(15, 29, 48, 0.95);
  border-color: rgba(71, 95, 124, 0.85);
}

.history-wrap.is-dark :deep(.history-range .el-range-input) {
  color: #dbe9ff;
}

.history-wrap.is-dark :deep(.history-range .el-range-separator),
.history-wrap.is-dark :deep(.history-range .el-range__icon),
.history-wrap.is-dark :deep(.history-range .el-range__close-icon) {
  color: #9fb2ca;
}

.history-wrap.is-dark :deep(.el-pagination) {
  color: #cfe1f8;
}

.history-wrap.is-dark :deep(.el-pagination .btn-prev),
.history-wrap.is-dark :deep(.el-pagination .btn-next),
.history-wrap.is-dark :deep(.el-pagination .el-pager li) {
  background: rgba(15, 29, 48, 0.95) !important;
  border: 1px solid rgba(71, 95, 124, 0.72) !important;
  color: #cfe1f8 !important;
}

.history-wrap.is-dark :deep(.el-pagination .el-pager li.is-active) {
  background: linear-gradient(180deg, #225b8e 0%, #1a4f80 100%) !important;
  border-color: #5fa9e6 !important;
  color: #ffffff !important;
}

.history-wrap.is-dark :deep(.el-pagination .el-select__wrapper),
.history-wrap.is-dark :deep(.el-pagination .el-input__wrapper) {
  background: rgba(15, 29, 48, 0.95) !important;
  box-shadow: 0 0 0 1px rgba(71, 95, 124, 0.85) inset !important;
}

.history-wrap.is-dark :deep(.el-pagination .el-select__selected-item),
.history-wrap.is-dark :deep(.el-pagination .el-input__inner),
.history-wrap.is-dark :deep(.el-pagination .el-select__placeholder) {
  color: #dbe9ff !important;
}

/* 时间选择弹层（teleported 到 body）深色样式 */
:deep(.history-range-popper.el-picker-panel) {
  background: linear-gradient(180deg, rgba(19, 37, 61, 0.98) 0%, rgba(15, 29, 48, 0.98) 100%) !important;
  border-color: #2f4968 !important;
  color: #dbe9ff !important;
}

:deep(.history-range-popper .el-picker-panel__body-wrapper),
:deep(.history-range-popper .el-picker-panel__body),
:deep(.history-range-popper .el-date-range-picker__content),
:deep(.history-range-popper .el-date-table tr),
:deep(.history-range-popper .el-date-table td) {
  background: transparent !important;
}

:deep(.history-range-popper .el-date-range-picker__header),
:deep(.history-range-popper .el-date-range-picker__time-header) {
  color: #dbe9ff !important;
  border-bottom-color: rgba(71, 95, 124, 0.75) !important;
}

:deep(.history-range-popper .el-picker-panel__icon-btn) {
  color: #a7bfdc !important;
}

:deep(.history-range-popper .el-date-table th) {
  color: #9fb2ca !important;
}

:deep(.history-range-popper .el-date-table td .el-date-table-cell__text) {
  color: #dbe9ff !important;
}

:deep(.history-range-popper .el-date-table td.available:hover .el-date-table-cell__text) {
  background: rgba(73, 101, 135, 0.38) !important;
}

:deep(.history-range-popper .el-date-table td.today .el-date-table-cell__text) {
  color: #8fd4ff !important;
}

:deep(.history-range-popper .el-date-table td.current:not(.disabled) .el-date-table-cell__text) {
  background: linear-gradient(180deg, #225b8e 0%, #1a4f80 100%) !important;
  color: #ffffff !important;
}
</style>
