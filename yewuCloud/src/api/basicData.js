import { get, JsonPost, formDataPost } from '@/api/api.js'

export function saveSiteGroupArr(params) { // 添加站址分组
  const siteGroupArrayList = []
  params.list.forEach((item, index) => {
    siteGroupArrayList.push({
      treeId: item.treeId, // 自身ID
      parentId: item.parentId, // 父节点ID
      level: item.level, // 级别
      siteGroupName: item.siteGroupName // 设备分组名称
    })
    if (item.id) siteGroupArrayList[index].id = item.id // 旧数据的id
  })
  const data = { siteGroupArrayList }
  if (params.id) data['id'] = params.id // 全部站址分组的总ID（若添加前已有站址分组则必填）
  return JsonPost(`business/sitegrouparr/add`, data)
}

export async function findSiteGroupArr(params) { // 查询全部站址分组
  let query = `?`
  if (params) {
    if (params.id && params.id !== '') query += `id=${params.id}&` // 站址分组ObjectId
    if (params.siteGroupName && params.siteGroupName !== '') query += `siteGroupName=${params.siteGroupName}` // 站址分组名称(String)
  }
  const res = await get(`business/sitegrouparr/find${query}`)
  const payload = res.return || {}
  return { ...res, ...payload, siteGroupArr: payload.data || payload.siteGroupArr }
}

export function deleteSiteGroup(params) { // 删除站址分组
  return JsonPost(`business/site/deleteSiteGroup`, {
    treeId: params.treeId,
    id: params.id, // 站址分组ObjectId
    siteGroupName: params.siteGroupName // 站址分组名称(String)
  })
}

export function modifySiteGroupArr(params) { // 修改站址分组
  const siteGroupArrayList = []
  params.list.forEach(item => {
    siteGroupArrayList.push({
      id: item.id, // 站址分组标志位
      treeId: item.treeId, // 自身ID
      parentId: item.parentId, // 父节点ID
      level: item.level, // 级别
      siteGroupName: item.siteGroupName // 设备分组名称
    })
  })
  return JsonPost(`business/site/modifySiteGroupArr`, {
    id: params.id, // 全部站址分组的总ID
    siteGroupArrayList
  })
}

export function saveEquipmentType(params) { // 添加设备类型
  return JsonPost(`business/equipment/saveEquipmentType`, {
    equipmentTypeName: params.equipmentTypeName, // 设备类型名称(String)
    equipmentTypeDescribe: params.equipmentTypeDescribe // 设备类型描述
  })
}

export async function findEquipmentTypeList(params) { // 查询设备类型
  let query = ``
  if (params && params.equipmentTypeName) {
    query = `?equipmentTypeName=${params.equipmentTypeName}` // 设备类型名称
  }
  const res = await get(`business/equipment/findEquipmentTypeList${query}`)
  return { ...res, ...(res.return || {}) }
}

export function deleteEquipmentType(params) { // 删除设备类型
  return JsonPost(`business/equipment/deleteEquipmentType`, {
    id: params.id, // 设备类型ObjectId
    equipmentTypeName: params.equipmentTypeName // 设备类型名称(String)
  })
}

export function modifyEquipmentType(params) { // 修改设备类型
  return JsonPost(`business/equipment/modifyEquipmentType`, {
    id: params.id, // 设备类型ObjectId
    equipmentTypeName: params.equipmentTypeName, // 设备类型名称(String)
    equipmentTypeDescribe: params.equipmentTypeDescribe // 设备类型描述
  })
}

export function findEquipmentmitrcd(param) {
  return JsonPost(`business/equipmentmitrcd/find?pageNum=${param.pageNum}&pageSize=${param.pageSize}&startTime=${param.startTime}&endTime=${param.endTime}`, {
    equipmentId: param.equipmentId,
    equipmentCode: param.equipmentCode,
    equipmentTypeName: param.equipmentTypeName,
    minimumCount: param.minimumCount,
    pageNum: param.pageNum,
    pageSize: param.pageSize,
    startTime: param.startTime,
    endTime: param.endTime
  })
}

export function uploadEquipmentmitrcdImage(param) {
  console.log('param:', param)
  const formData = new FormData() // 创建FormData对象
  formData.append('image', param.file) // 添加文件到FormData
  return formDataPost(`go/images/upload`, formData)
}

export function getCurrentDateTimeString() {
  const now = new Date()

  const year = now.getFullYear() // 年，如 2025
  const month = String(now.getMonth() + 1).padStart(2, '0') // 月，从0开始，要+1，补零到2位
  const day = String(now.getDate()).padStart(2, '0') // 日，补零到2位

  const hours = String(now.getHours()).padStart(2, '0') // 时，补零
  const minutes = String(now.getMinutes()).padStart(2, '0') // 分，补零
  const seconds = String(now.getSeconds()).padStart(2, '0') // 秒，补零

  // 拼接成：2025-09-10 09:14:31
  return `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`
}

export function addEquipmentmitrcd(param) {
  return JsonPost(`business/equipmentmitrcd/add`, {
    equipmentId: param.equipmentId,
    maintainUserId: param.maintainUserId,
    maintainUserName: param.maintainUserName,
    maintainTitle: param.maintainTitle,
    maintainContent: param.maintainContent,
    maintainPhotos: param.maintainPhotos,
    maintainDetails: {
      create_time: param.maintainDetails.create_time,
      imageUrl: param.maintainDetails.imageUrl,
      bs: param.maintainDetails.bs,
      province_company: param.maintainDetails.province_company,
      // 故障详情属于运维记录扩展信息，保留在 JSON 字段中避免改动旧表结构。
      fault_detail: param.maintainDetails.fault_detail
    }
  })
}

export function findEquipmentcirlog(param) {
  return JsonPost(`business/equipmentcirlog/find`, {
    equipmentId: param.equipmentId
  })
}
