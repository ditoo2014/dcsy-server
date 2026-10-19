import { JsonPost } from '@/api/api.js'

export async function findCollectLiveTriggerTempData(params) { // 查询正在告警的信息
  const equipmentSerialNumber = []
  if (params.equipmentSerialNumber) {
    params.equipmentSerialNumber.forEach(item => {
      equipmentSerialNumber.push({
        id: item.id, symbel: item.symbel
      })
    })
  }
  let likeSearch = params.likeSearch
  if (likeSearch === undefined) {
    likeSearch = true
  }
  let query = `?pageNumber=${params.pageNumber}&pageSize=${params.pageSize}&likeSearch=${likeSearch}` // likeSearch 用于开启模糊查询功能
  if (params.startTime) query += `&startTime=${params.startTime}` // 首个触发的时间
  if (params.endTime) query += `&endTime=${params.endTime}` // 最后一个触发的时间

  const data = {}
  if (params.id) data['id'] = params.id // 数据库id
  if (params.equipmentSerialNumber) data['equipmentSerialNumber'] = equipmentSerialNumber // 设备自定义序列号(hashmap数组)
  if (params.equipmentId) data['equipmentId'] = params.equipmentId // 设备id(String)
  if (params.equipmentName) data['equipmentName'] = params.equipmentName // 设备名称(String)
  if (params.siteSerialNumber) data['siteSerialNumber'] = params.siteSerialNumber // 站址自定义序列号(String)
  if (params.siteId) data['siteId'] = params.siteId // 站址名称(String)
  if (params.siteName) data['siteName'] = params.siteName // 站址名称(String)
  if (params.coding) data['coding'] = params.coding // 触发器代码(String)
  if (params.begin) data['begin'] = params.begin // 处理开始执行时间(Long)
  if (params.end) data['end'] = params.end // 处理结束执行时间(Long)
  if (params.value) data['value'] = params.value // 逻辑运算结束(Boolean)
  if (params.result) data['result'] = params.result // 处理结果数据(hashmap数组)
  if (params.err) data['err'] = params.err // 不为""表示异常,异常信息(String)
  const res = await JsonPost(`business/collect/findCollectLiveTriggerTempData${query}`, data)
  return { ...res, ...(res.return || {}) }
}

export function removeCollectLiveTriggerTempData(params) { // 删除正在告警的信息
  return JsonPost(`business/collect/removeCollectLiveTriggerTempData`, {
    id: params.id // 数据库id
  })
}
