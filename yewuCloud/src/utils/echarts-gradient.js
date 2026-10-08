/**
 * ECharts 线性渐变配置（纯对象）。
 * 勿在 data() 中使用 echarts.graphic.LinearGradient，Vue 3 响应式会导致 colorStops 无法写入。
 */
export function linearGradientColor(colorStops) {
  return {
    type: 'linear',
    x: 0,
    y: 0,
    x2: 0,
    y2: 1,
    colorStops
  }
}

/** 将 ECharts 4 的 lineStyle/areaStyle/itemStyle.normal 转为 ECharts 5 扁平结构 */
export function normalizeSeriesItem(item) {
  const series = { ...item }
  ;['lineStyle', 'areaStyle', 'itemStyle', 'label'].forEach((key) => {
    if (series[key] && series[key].normal) {
      series[key] = { ...series[key].normal }
    }
  })
  return series
}
