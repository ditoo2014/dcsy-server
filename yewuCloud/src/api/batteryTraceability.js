import { JsonPost } from '@/api/api.js'

export function siteExport(params) { // 站址导出数据
  const data = {}
  if (params) {
    if (params.id && params.id !== 0) data.id = params.id // 站址数据库id
    if (params.siteSerialNumber && params.siteSerialNumber !== '') data.siteSerialNumber = params.siteSerialNumber // 站址名称
    if (params.siteName && params.siteName !== '') data.siteName = params.siteName // 站址所属分组名称
    if ((params.corpId && params.corpId !== '') || params.corpId === 0) data['corpId'] = params.corpId // 部门id
  }
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const filterFunc = params.filterFunc ? params.filterFunc : 'BinC' // ConE:通过设备关注用户查询, BinC:通过站址绑定部门查询
  const query = `?pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}`
  return JsonPost(`business/site/export${query}`, data, {
    responseType: 'blob'
  })
}

export function siteImport(params) { // 站址导入数据
  const data = new FormData()
  data.append('file', params.fileList[0])
  return JsonPost(`business/site/import`, data, {
    responseType: 'blob'
  })
}

// 模板由当前后端按实际导入表头生成，不依赖旧部署服务器的静态目录。
export function siteImportTemplate() {
  return JsonPost('/business/site/import/template', {}, { responseType: 'blob' })
}

export function warehouseExport(params) { // 库房导出数据
  const data = {}
  if (params) {
    if (params.name && params.name !== '') data.name = params.name
    if ((params.corpId && params.corpId !== '') || params.corpId === 0) data['corpId'] = params.corpId // 部门id
  }
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const query = `?pageNum=${pageNum}&pageSize=${pageSize}`
  return JsonPost(`business/warehouse/export${query}`, data, {
    responseType: 'blob'
  })
}

export function warehouseImport(params) { // 库房导入数据
  const data = new FormData()
  data.append('file', params.fileList[0])
  return JsonPost(`business/warehouse/import`, data, {
    responseType: 'blob'
  })
}

export function warehouseImportTemplate() {
  return JsonPost('/business/warehouse/import/template', {}, { responseType: 'blob' })
}

export function equipmentExport(params) { // 设备导出数据
  const data = {}
  if (params) {
    if (params.id && params.id !== '') data['id'] = params.id // 设备的ObjectId(String)
    if ((params.corpId && params.corpId !== '') || params.corpId === 0) data['corpId'] = params.corpId // 部门id
    if (params.equipmentCode && params.equipmentCode !== '') data['equipmentCode'] = params.equipmentCode // 设备名称(String)
    if (params.equipmentName && params.equipmentName !== '') data['equipmentName'] = params.equipmentName // 设备名称(String)
    if ((params.belongToSiteId || params.belongToSiteId === 0) && params.belongToSiteId !== '') data['belongToSiteId'] = params.belongToSiteId // 选择的站址ObjectId
    if (params.belongToSiteName && params.belongToSiteName !== '') data['belongToSiteName'] = params.belongToSiteName // 选择的站址名称(String)
    if (params.equipmentTypeId && params.equipmentTypeId !== '') data['equipmentTypeId'] = params.equipmentTypeId // 设备类型ObjectId
    if (params.equipmentTypeName && params.equipmentTypeName !== '') data['equipmentTypeName'] = params.equipmentTypeName // 设备类型名称(String)
    if (params.codeRule && params.codeRule !== '') data['codeRule'] = params.codeRule // 编码规则(用户自定义序列号长度)
    if (params.engageInternetStatus && params.engageInternetStatus !== '') data['engageInternetStatus'] = params.engageInternetStatus // 并网状态
    if ((params.positionStatus || params.positionStatus === 0) && params.positionStatus !== '') data['positionStatus'] = params.positionStatus // 当前电池状态（0=未知，不能用 truthy 判断）
    if (params.manufacturer && params.manufacturer !== '') data['manufacturer'] = params.manufacturer // 生产厂家
    if (params.bmsManufacturer && params.bmsManufacturer !== '') data['bmsManufacturer'] = params.bmsManufacturer // BMS生产厂家
  }
  const filterFunc = params.filterFunc ? params.filterFunc : 'BinC' // ConE:通过设备关注用户查询, BinC:通过站址绑定部门查询
  const pageNum = params.pageNum ? params.pageNum : 1
  const pageSize = params.pageSize ? params.pageSize : 10
  const query = `?pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}`
  return JsonPost(`business/equipment/export${query}`, data, {
    responseType: 'blob'
  })
}

export function equipmentImport(params) { // 设备导入数据
  const data = new FormData()
  data.append('file', params.fileList[0])
  return JsonPost(`business/equipment/import`, data, {
    responseType: 'blob'
  })
}

export function equipmentImportTemplate() { // 电池出厂信息导入模板
  return JsonPost(`business/equipment/import/template`, {}, {
    responseType: 'blob'
  })
}

export function statistictaskPlan(params) { // 统计任务
  const startTime = params.startTime
  const endTime = params.endTime
  const query = `?startTime=${startTime}&endTime=${endTime}`
  return JsonPost(`/business/statistictask/start${query}`, {})
}

export function equipmentmitrcdExport(params = {}) { // 运维信息导出数据
  const { equipmentId, equipmentCode, equipmentTypeName, minimumCount, startTime, endTime } = params
  const query = new URLSearchParams({ startTime: startTime || 0, endTime: endTime || 0 })
  return JsonPost(`/business/equipmentmitrcd/export?${query}`, {
    equipmentId, equipmentCode, equipmentTypeName, minimumCount
  }, { responseType: 'blob' })
}

export function equipmentmitrcdImport(params) { // 运维信息导入数据
  const data = new FormData()
  data.append('file', params.fileList[0])
  return JsonPost(`business/equipmentmitrcd/import`, data)
}
