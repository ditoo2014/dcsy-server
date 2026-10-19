// 仅补充当前页缺失的关联资料，避免一次加载全部运维记录。
const LOOKUP_CONCURRENCY = 4

export function maintenanceErrorMessage(error, fallback = '运维数据加载失败，请检查网络或稍后重试') {
  const message = String(error?.message || '')
  // 网络库的英文错误不直接展示给用户；后端给出的中文业务原因仍原样保留。
  return /[\u4e00-\u9fff]/.test(message) ? message : fallback
}

export function readMaintenanceFilters(bannerList, now = Date.now()) {
  const find = (placeholder) => bannerList.find(item => item.placeholder === placeholder)
  const range = bannerList.find(item => item.name === '选择时间段')?.timeDateArr
  let startTime = 0
  let endTime = Math.floor(now / 1000)
  if (Array.isArray(range) && range.length === 2) {
    const values = range.map(value => new Date(value).getTime())
    if (!values.every(Number.isFinite)) throw new Error('请选择有效的开始时间和结束时间')
    startTime = Math.floor(Math.min(...values) / 1000)
    endTime = Math.floor(Math.max(...values) / 1000)
  }
  return {
    equipmentCode: String(find('请输入电池ID')?.input || '').trim(),
    equipmentTypeName: String(find('请输入电池类型')?.input || '').trim(),
    // 延续原版 5/10/20 筛选的“大于等于”口径，并与当前时间范围共同生效。
    minimumCount: Math.max(0, Number(find('运维次数大于')?.value) || 0),
    startTime,
    endTime
  }
}

function readRows(response) {
  if (response?.code != null && Number(response.code) !== 200) {
    throw new Error(response.msg || response.message || '运维数据查询失败')
  }
  const rows = response?.return?.data
  if (rows == null) return []
  if (!Array.isArray(rows)) throw new Error('运维接口返回的数据格式不正确')
  return rows
}

function readDetails(value) {
  if (value && typeof value === 'object' && !Array.isArray(value)) return value
  if (typeof value === 'string') {
    try {
      const parsed = JSON.parse(value)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) return parsed
    } catch { /* 旧记录缺失扩展字段时显示空值，不伪造内容。 */ }
  }
  return {}
}

function displayTime(value) {
  if (value == null || value === '') return '-'
  if (!/^\d{10}(?:\d{3})?$/.test(String(value))) return String(value)
  const date = new Date(Number(value) * (String(value).length === 10 ? 1000 : 1))
  if (!Number.isFinite(date.getTime())) return '-'
  const pad = number => String(number).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

function hasValue(value) {
  const text = value === undefined || value === null ? '' : String(value).trim()
  // 旧接口用短横线表示缺失值，不能把它当作真实编码、站址或图片地址。
  return text !== '' && text !== '-'
}

function firstValue(...values) {
  return values.find(hasValue)
}

async function lookupAll(ids, lookup, isCurrent) {
  const result = new Map()
  let index = 0
  await Promise.all(Array.from({ length: Math.min(LOOKUP_CONCURRENCY, ids.length) }, async () => {
    while (index < ids.length && isCurrent()) {
      const id = ids[index++]
      // 设备或站点资料是运维主记录的补充信息。单条关联请求失败时保留主记录，
      // 不能因为一台设备的资料异常而清空整张运维列表。
      try {
        const response = await lookup(id)
        if (!isCurrent()) return
        result.set(String(id), readRows(response).find(item => String(item.id) === String(id)) || {})
      } catch {
        if (!isCurrent()) return
        result.set(String(id), {})
      }
    }
  }))
  return result
}

/** 先取得真实运维记录，再补齐电池/站点信息；同一次查询内每个实体只请求一次。 */
export async function loadMaintenanceTable(api, filters, isCurrent = () => true) {
  const pageNum = filters.pageNum || 1
  const pageSize = filters.pageSize || 10
  const response = await api.findRecords({
    pageNum, pageSize,
    startTime: filters.startTime, endTime: filters.endTime,
    equipmentCode: filters.equipmentCode,
    equipmentTypeName: filters.equipmentTypeName,
    minimumCount: filters.minimumCount
  })
  if (!isCurrent()) return null
  const records = readRows(response)
  const total = Number(response?.return?.total)
  const equipmentCount = Number(response?.return?.equipmentCount)
  if (!Number.isFinite(total) || total < 0 || !Number.isFinite(equipmentCount) || equipmentCount < 0) {
    throw new Error('运维统计数据暂不可用，请稍后重试')
  }

  const equipmentIds = [...new Set(records.map(row => row.equipmentId).filter(id => Number(id) > 0))]
  // 新接口会在运维记录中直接返回设备资料。先使用这些字段，只有确实缺字段时才补查设备。
  // 这样既减少请求，也能在补充接口失败时保留后端已经返回的资料。
  const equipmentMap = new Map()
  records.forEach(record => {
    const key = String(record.equipmentId)
    const current = equipmentMap.get(key) || {}
    equipmentMap.set(key, {
      ...current,
      equipmentCode: firstValue(current.equipmentCode, record.equipmentCode),
      equipmentName: firstValue(current.equipmentName, record.equipmentName),
      equipmentTypeName: firstValue(current.equipmentTypeName, record.equipmentTypeName),
      siteSerialNumber: firstValue(current.siteSerialNumber, record.siteSerialNumber),
      positionStatus: current.positionStatus !== undefined && current.positionStatus !== null
        ? current.positionStatus
        : record.positionStatus,
      belongToSiteId: firstValue(current.belongToSiteId, record.belongToSiteId),
      belongToSiteName: firstValue(current.belongToSiteName, record.belongToSiteName)
    })
  })
  const equipmentLookupIds = equipmentIds.filter(id => {
    const equipment = equipmentMap.get(String(id)) || {}
    return !hasValue(equipment.equipmentCode) || !hasValue(equipment.equipmentName) ||
      !hasValue(equipment.equipmentTypeName) || equipment.positionStatus == null ||
      (!hasValue(equipment.siteSerialNumber) && !hasValue(equipment.belongToSiteId))
  })
  const equipmentSupplement = await lookupAll(equipmentLookupIds,
    id => api.findEquipment({ id, pageNum: 1, pageSize: 1 }), isCurrent)
  if (!isCurrent()) return null
  equipmentSupplement.forEach((supplement, id) => {
    const current = equipmentMap.get(id) || {}
    const merged = { ...current }
    Object.entries(supplement).forEach(([key, value]) => {
      if (key === 'positionStatus' ? value !== undefined && value !== null : hasValue(value)) {
        merged[key] = value
      }
    })
    equipmentMap.set(id, merged)
  })
  const siteIds = [...new Set([...equipmentMap.values()]
    .filter(row => !hasValue(row.siteSerialNumber))
    .map(row => row.belongToSiteId).filter(id => Number(id) > 0))]
  const siteMap = await lookupAll(siteIds, id => api.findSite({ id, pageNum: 1, pageSize: 1 }), isCurrent)
  if (!isCurrent()) return null

  const rows = records.map(record => {
    const equipment = equipmentMap.get(String(record.equipmentId)) || {}
    const site = siteMap.get(String(equipment.belongToSiteId)) || {}
    const details = readDetails(record.maintainDetails)
    const equipmentCode = String(firstValue(record.equipmentCode, equipment.equipmentCode) || '').trim()
    const equipmentName = String(firstValue(record.equipmentName, equipment.equipmentName) || '').trim()
    const equipmentTypeName = String(firstValue(record.equipmentTypeName, equipment.equipmentTypeName) || '')
    const siteSerialNumber = firstValue(record.siteSerialNumber, equipment.siteSerialNumber,
      site.siteSerialNumber, equipment.belongToSiteName)
    const positionStatus = record.positionStatus !== undefined && record.positionStatus !== null
      ? record.positionStatus : (equipment.positionStatus ?? null)
    const imageUrls = Array.from(new Set([
      ...(Array.isArray(record.maintainPhotos) ? record.maintainPhotos : []),
      details.imageUrl
    ].filter(hasValue)))
    return {
      id: record.id,
      equipmentId: record.equipmentId,
      // 页面展示和筛选统一优先使用设备编码；兼容旧设备没有编码时使用设备名称。
      batteryCode: equipmentCode || equipmentName || '-',
      equipmentCode,
      equipmentName,
      equipmentTypeName,
      siteSerialNumber: siteSerialNumber || '-',
      maintenanceDescription: firstValue(record.maintainTitle, record.maintainContent) || '',
      faultDescription: firstValue(record.maintainTitle, record.maintainContent) || '',
      faultDetail: details.fault_detail || record.maintainContent || '',
      imgUrl: imageUrls[0] || '',
      imgUrls: imageUrls,
      // 新记录时间以数据库 createTime 为准；旧记录没有时才读取扩展 JSON 时间。
      maintenanceTime: displayTime(firstValue(
        Number(record.createTime) > 0 ? record.createTime : undefined,
        details.create_time
      )),
      bs: details.bs || '',
      province_company: details.province_company || '',
      // 当前状态只取设备真实状态；没有数据时详情明确显示未知。
      positionStatus
    }
  })

  // 次数、编码和类型筛选已由后端统一处理；这里只负责分页整理和统计展示。
  return {
    rows: rows.map((row, index) => ({ ...row, rowNumber: (pageNum - 1) * pageSize + index + 1 })),
    total,
    equipmentCount
  }
}
