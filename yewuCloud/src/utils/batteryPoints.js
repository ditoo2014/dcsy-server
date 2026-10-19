import {
  matchesBatteryPointId,
  TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID,
  TOTAL_CHARGE_CAPACITY_POINT_DATA_ID
} from './batteryPointIds.js'
export {
  CYCLE_COUNT_POINT_DATA_ID,
  TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID,
  TOTAL_CHARGE_CAPACITY_POINT_DATA_ID,
  PRESSURE_POINT_DATA_ID,
  TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
  REMAINING_DISCHARGE_TIME_POINT_DATA_ID
} from './batteryPointIds.js'

// 两项累计容量允许沿用设备已绑定的标签，读取其真实 ID，不改写数据库或历史数据。
const cumulativeCapacityLabelNames = new Map([
  [TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID, ['累计放电量', '累计放电容量']],
  [TOTAL_CHARGE_CAPACITY_POINT_DATA_ID, ['累计充电量', '累计充电容量']]
])

export function resolveBatteryHealthStatus(soh) {
  const numericSoh = Number.parseFloat(String(soh).replace('%', '').trim())
  if (!Number.isFinite(numericSoh)) return '-'
  if (numericSoh > 90) return '优'
  if (numericSoh >= 80) return '良'
  if (numericSoh >= 60) return '中'
  return '差'
}

function normalize(value) {
  return value == null ? '' : String(value).trim().toLowerCase()
}

function hasUsableValue(item) {
  if (!item) return false
  if (item.realTimeStatus != null && Number(item.realTimeStatus) !== 1) return false
  const value = item.realTimeVal
  return value !== null && value !== undefined && String(value).trim() !== '' && value !== '异常'
}

function pointTimestamp(item) {
  const timestamp = Number(item && item.realTimeTimestamp)
  if (Number.isFinite(timestamp)) return timestamp
  const parsedDate = Date.parse(item && item.realTimeDate)
  return Number.isFinite(parsedDate) ? parsedDate : 0
}

function selectBestPoint(list, gatewayId) {
  let candidates = list
  const normalizedGatewayId = normalize(gatewayId)
  if (normalizedGatewayId) {
    const gatewayMatches = candidates.filter(item => normalize(item.gatewayId) === normalizedGatewayId)
    if (gatewayMatches.length) candidates = gatewayMatches
  }

  return candidates.slice().sort((left, right) => {
    const usableDiff = Number(hasUsableValue(right)) - Number(hasUsableValue(left))
    if (usableDiff) return usableDiff
    const timeDiff = pointTimestamp(right) - pointTimestamp(left)
    if (timeDiff) return timeDiff
    return normalize(left.labelId).localeCompare(normalize(right.labelId))
  })[0]
}

export function resolveBatteryPoint(list, options = {}) {
  const points = Array.isArray(list) ? list : []
  const { dataIdSuffix, gatewayId } = options
  // 优先使用新英文 ID；其余固定业务数据点仍只按 ID 识别。
  const codeMatches = dataIdSuffix
    ? points.filter(item => matchesBatteryPointId(item && item.dataId, dataIdSuffix))
    : []
  if (codeMatches.length) return selectBestPoint(codeMatches, gatewayId)

  const labelNames = cumulativeCapacityLabelNames.get(normalize(dataIdSuffix))
  if (!labelNames) return null
  // 只从当前设备已有绑定中选择完整名称匹配的标签；保留真实 labelId/dataId 用于查询。
  const boundLabels = points.filter(item => {
    if (!item || !String(item.labelId ?? '').trim()) return false
    const labelName = normalize(item.labelName || item.name).replace(/^[0-9a-f]{2}_/, '')
    return labelNames.includes(labelName)
  })
  return boundLabels.length ? selectBestPoint(boundLabels, gatewayId) : null
}
