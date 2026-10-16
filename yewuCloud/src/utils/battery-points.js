import { matchesBatteryPointId } from './battery-point-ids.js'
export {
  CYCLE_COUNT_POINT_DATA_ID,
  TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID,
  TOTAL_CHARGE_CAPACITY_POINT_DATA_ID,
  PRESSURE_POINT_DATA_ID,
  TEMPERATURE_DIFFERENCE_POINT_DATA_ID,
  REMAINING_DISCHARGE_TIME_POINT_DATA_ID
} from './battery-point-ids.js'

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

function matchesDataIdSuffix(dataId, suffix) {
  return matchesBatteryPointId(dataId, suffix)
}

function pointName(item) {
  return normalize(item && (item.labelName || item.name))
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
  const { dataIdSuffix, exactNames = [], aliases = [], gatewayId } = options
  const codeMatches = dataIdSuffix
    ? points.filter(item => matchesDataIdSuffix(item && item.dataId, dataIdSuffix))
    : []
  if (codeMatches.length) return selectBestPoint(codeMatches, gatewayId)

  // 原版 ID 未匹配时，沿用名称回退并保留实际标签中的数据点 ID。
  const normalizedExactNames = exactNames.map(normalize).filter(Boolean)
  const exactNameMatches = normalizedExactNames.length
    ? points.filter(item => normalizedExactNames.includes(pointName(item)))
    : []
  if (exactNameMatches.length) return selectBestPoint(exactNameMatches, gatewayId)

  const normalizedAliases = aliases.map(normalize).filter(Boolean)
  const nameMatches = points.filter(item => {
    const name = pointName(item)
    return normalizedAliases.some(alias => name.indexOf(alias) !== -1)
  })
  return nameMatches.length ? selectBestPoint(nameMatches, gatewayId) : null
}
