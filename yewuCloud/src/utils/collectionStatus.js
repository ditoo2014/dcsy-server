const COLLECTION_FRESHNESS_SECONDS = 30 * 60
const COLLECTION_POINT_NAMES = ['组电压', '组电流', 'SOH', 'SOC', '放电时长']
const API_KEY_ERROR_MESSAGES = ['illegal apikey', 'apikey not exist']

export function normalizePointTime(value) {
  const timestamp = Number(value)
  if (!Number.isFinite(timestamp) || timestamp <= 0) return 0
  return timestamp > 1000000000000 ? Math.floor(timestamp / 1000) : timestamp
}

export function hasPointValue(value) {
  return value !== null && value !== undefined && String(value).trim() !== '' && String(value) !== '-999999'
}

function isCollectionPoint(labelItem) {
  const labelName = labelItem.labelName || ''
  return COLLECTION_POINT_NAMES.some(name => {
    if (name === 'SOH' || name === 'SOC') return labelName === name
    return labelName.indexOf(name) !== -1
  })
}

export function resolveCollectStatus(item, pointMap, responseItem) {
  if (!responseItem) return 0
  if (API_KEY_ERROR_MESSAGES.includes(responseItem.msg)) return 999
  if (responseItem.msg === 'empty label list') return 0

  const dataDetails = Array.isArray(item.dataDetails) ? item.dataDetails : []
  const collectionPoints = dataDetails.filter(isCollectionPoint)
  const pointsToCheck = collectionPoints.length > 0 ? collectionPoints : dataDetails
  if (pointsToCheck.length === 0) return 0

  const now = Math.floor(Date.now() / 1000)
  const hasFreshPoint = pointsToCheck.some(labelItem => {
    const point = pointMap.get(String(labelItem.labelId))
    if (!point) return false

    const timestamp = normalizePointTime(point.time)
    const age = now - timestamp
    return Number(point.status) === 1 && hasPointValue(point.value) && timestamp > 0 && age >= 0 && age <= COLLECTION_FRESHNESS_SECONDS
  })

  // Collection availability follows the newest device update; algorithm points can refresh independently.
  return hasFreshPoint ? 1 : 2
}
