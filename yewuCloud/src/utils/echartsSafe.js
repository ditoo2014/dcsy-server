/** ECharts 5 安全 resize / setOption，避免窗口缩放时 layout 读到 undefined series */

export function getChartContainer(chart, el) {
  if (el) return el
  if (chart && chart.getDom) return chart.getDom()
  return null
}

export function safeEchartsResize(chart, el) {
  if (!chart || chart.isDisposed()) {
    return false
  }
  const dom = getChartContainer(chart, el)
  if (!dom || dom.clientWidth === 0 || dom.clientHeight === 0) {
    return false
  }
  try {
    chart.resize()
    return true
  } catch (e) {
    return false
  }
}

export function normalizeChartSeries(series) {
  if (!Array.isArray(series)) {
    return []
  }
  return series
    .filter(item => item && typeof item === 'object' && item.type)
    .map(item => {
      const next = { ...item }
      if (next.type === 'bar' || next.type === 'line') {
        next.coordinateSystem = 'cartesian2d'
      }
      return next
    })
}

export function safeEchartsSetOption(chart, option, extra = {}) {
  if (!chart || chart.isDisposed() || !option) {
    return false
  }
  const next = { ...option }
  if (next.series) {
    next.series = normalizeChartSeries(next.series)
    if (!next.series.length) {
      return false
    }
  }
  try {
    chart.setOption(next, { notMerge: false, ...extra })
    return true
  } catch (e) {
    try {
      chart.setOption(next, { notMerge: true })
      return true
    } catch (e2) {
      return false
    }
  }
}
