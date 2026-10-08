import { JsonPost } from '@/api/api.js'

export function saveSite(params) { // 添加站址
  return JsonPost(`business/site/add`, {
    apiKey: params.apiKey, // 对应采集云APIKkey*(String)
    apiKeys: params.apiKeys, // 采集云多组apikey, 格式为:[{name:'', apikey:''}]
    siteName: params.siteName, // 站址名称*(String)
    siteSerialNumber: params.siteSerialNumber, // 站址名称
    siteGroupId: params.siteGroupId, // 站址所属分组ObjectId*
    siteGroupName: params.siteGroupName, // 站址所属分组名称*(String)
    siteGroupLevel: params.siteGroupLevel, // 站址分组级别
    corpId: params.corpId, // 部门id
    siteDetails: {
      longitude: params.longitude, // 经度
      latitude: params.latitude, // 纬度
      provinceName: params.provinceName, // 省份
      cityName: params.cityName, // 城市
      addr: params.addr // 详细地址
    }
  })
}

export function findSiteListExactMatch(params) { // 查询站址
  const data = {}
  if (params) {
    if (params.id && params.id !== 0) data.id = params.id // 站址名称
    if (params.siteSerialNumber && params.siteSerialNumber !== '') data.siteSerialNumber = params.siteSerialNumber // 站址名称
    if (params.siteName && params.siteName !== '') data.siteName = params.siteName // 站址名称
    if ((params.corpId && params.corpId !== '') || params.corpId === 0) data['corpId'] = params.corpId // 部门id
  }
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const filterFunc = params.filterFunc ? params.filterFunc : 'BinC' // ConE:通过设备关注用户查询, BinC:通过站址绑定部门查询
  const query = `?pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}`
  return JsonPost(`business/site/find${query}`, data)
}

export function deleteSite(params) { // 删除站址
  return JsonPost(`business/site/delete`, {
    id: params.id, // 站址ObjectId
    siteName: params.siteName // 站址名称(String)
  })
}

export function modifySite(params) { // 修改站址
  return JsonPost(`business/site/update`, {
    id: params.id,
    apiKey: params.apiKey, // 对应采集云APIKkey*(String)
    apiKeys: params.apiKeys, // 采集云多组apikey, 格式为:[{name:'', apikey:''}]
    siteName: params.siteName, // 站址名称*(String)
    siteSerialNumber: params.siteSerialNumber, // 站址名称
    siteGroupId: params.siteGroupId, // 站址所属分组ObjectId*
    siteGroupName: params.siteGroupName, // 站址所属分组名称*(String)
    siteGroupLevel: params.siteGroupLevel, // 站址分组级别
    corpId: params.corpId, // 部门id
    siteDetails: {
      longitude: params.longitude, // 经度
      latitude: params.latitude, // 纬度
      provinceName: params.provinceName, // 省份
      cityName: params.cityName, // 城市
      addr: params.addr // 详细地址
    }
  })
}
