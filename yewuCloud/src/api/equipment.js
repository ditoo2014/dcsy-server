import { JsonPost } from '@/api/api.js'

export function saveEquipment(params) { // 添加设备
  const data = {
    equipmentName: params.equipmentName, // 设备名称*(String)
    belongToSiteId: params.belongToSiteId, // 选择的站址ObjectId
    belongToSiteName: params.belongToSiteName, // 选择的站址名称(String)
    equipmentTypeId: params.equipmentTypeId, // 设备类型ObjectId
    equipmentTypeName: params.equipmentTypeName, // 设备类型名称(String)
    codeRule: params.codeRule, // 编码规则(用户自定义序列号长度)
    concernUserIds: params.concernUserIds, // 关注用户
    controlUserIds: params.controlUserIds, // 可控制监控的用户
    engageInternetStatus: params.engageInternetStatus, // 并网状态

    apiKey: params.apiKey,
    apiKeys: params.apiKeys, // 数组，例：[{ 'name': '兴福村600方', 'key': '632e66411db99f09317f8b79'}]
    belongToUserId: params.belongToUserId, // int
    runningStatus: params.runningStatus, // int
    alarmTriggerNum: params.alarmTriggerNum, // int
    deleted: params.deleted, // boolean
    videoProtoType: params.videoProtoType,
    videoUrls: params.videoUrls,
    ezvizCloud: params.ezvizCloud,

    equipmentCode: params.equipmentCode, // 设备编码(string)
    warehouseId: params.warehouseId, // 库房id(int)
    warehouseName: params.warehouseName, // 库房名称(string)
    manufacturer: params.manufacturer, // 生产厂家(string)
    manufacturerTime: params.manufacturerTime, // 生产日期(string)
    batch: params.batch, // 批次(string)
    specifications: params.specifications, // 规格(string)
    corpId: params.corpId, // 所属部门(string)
    bmsManufacturer: params.bmsManufacturer, // BMS生产厂家(string)
    bmsVersion: params.bmsVersion // BMS版本(string)
  }
  if (params.equipmentDetails) { // 设备各种详细属性 equipmentDetails的格式为：{'a': 'b', 'c': 'd'} // 'a'为自定义的键，'b'为a的值
    data['equipmentDetails'] = params.equipmentDetails
  }
  if (params.dataDetailsId && params.dataDetailsId !== '') {
    data['dataDetailsId'] = params.dataDetailsId
    data['dataDetailsSerialNumber'] = params.dataDetailSerialNumber
  }
  return JsonPost(`business/equipment/add`, data)
}

export function findEquipmentListExactMatch(params) { // 查询设备
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
  let query = `?pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}`
  if (params.includeLatestCirculation) query += '&includeLatestCirculation=1'
  // 轨迹列表沿用发生时间、日志编号的排序；其他页面保持原有取值规则。
  if (params.latestCirculationOrder === 'time') query += '&latestCirculationOrder=time'
  return JsonPost(`business/equipment/find${query}`, data)
}

export async function getEquipAllPoint(params) { // 获取多个设备以及对应的数据点
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
  const res = await JsonPost(`business/equipment/alldata${query}`, data)
  const equipments = res.return && res.return.data
  if (!Array.isArray(equipments)) return res

  // 业务详情只保存标签 ID，通过标签配置补齐实际数据点 ID，避免按中文名称识别旧编码。
  // 同一批设备共用数据池时只查询一次，不跨请求缓存，确保模板调整后使用最新配置。
  const metadataRequests = new Map()
  await Promise.all(equipments.map(async equipment => {
    const points = equipment.dataDetails
    const apiKey = String(equipment.apiKey || '').trim()
    if (!apiKey || apiKey === '-' || !Array.isArray(points) || !points.length) return
    if (!metadataRequests.has(apiKey)) {
      metadataRequests.set(apiKey, JsonPost(`api/getlabel?Apikey=${encodeURIComponent(apiKey)}`, {})
        .then(response => {
          const labels = response.return && response.return.labelList
          return new Map((Array.isArray(labels) ? labels : []).map(label => [String(label.labelId), label]))
        })
        .catch(error => {
          console.error('读取数据池数据点编码失败', error)
          return new Map()
        }))
    }
    const metadataMap = await metadataRequests.get(apiKey)
    points.forEach(point => {
      const metadata = metadataMap.get(String(point.labelId))
      if (!metadata) return
      point.dataId = metadata.dataId || ''
      point.gatewayId = metadata.gatewayId || ''
    })
  }))
  return res
}

export function deleteEquipment(params) { // 删除设备
  return JsonPost(`business/equipment/delete`, {
    id: params.id // 设备ObjectId
  })
}

export function modifyEquipment(params) { // 修改设备
  const data = {
    id: params.id,
    equipmentName: params.equipmentCode, // 设备名称*(String)
    belongToSiteId: params.belongToSiteId, // 选择的站址ObjectId
    belongToSiteName: params.belongToSiteName, // 选择的站址名称(String)
    equipmentTypeId: params.equipmentTypeId, // 设备类型ObjectId
    equipmentTypeName: params.equipmentTypeName, // 设备类型名称(String)
    codeRule: params.codeRule, // 编码规则(用户自定义序列号长度)
    concernUserIds: params.concernUserIds, // 关注用户
    controlUserIds: params.controlUserIds, // 可控制监控的用户
    engageInternetStatus: params.engageInternetStatus, // 并网状态

    apiKey: params.apiKey,
    apiKeys: params.apiKeys, // 数组，例：[{ 'name': '兴福村600方', 'key': '632e66411db99f09317f8b79'}]
    belongToUserId: params.belongToUserId, // int
    runningStatus: params.runningStatus, // int
    alarmTriggerNum: params.alarmTriggerNum, // int
    deleted: params.deleted, // boolean
    videoProtoType: params.videoProtoType,
    videoUrls: params.videoUrls,
    ezvizCloud: params.ezvizCloud,

    equipmentCode: params.equipmentCode, // 设备编码(string)
    warehouseId: params.warehouseId, // 库房id(int)
    warehouseName: params.warehouseName, // 库房名称(string)
    manufacturer: params.manufacturer, // 生产厂家(string)
    manufacturerTime: params.manufacturerTime, // 生产日期(string)
    batch: params.batch, // 批次(string)
    specifications: params.specifications, // 规格(string)
    corpId: params.corpId, // 所属部门(string)
    bmsManufacturer: params.bmsManufacturer, // BMS生产厂家(string)
    bmsVersion: params.bmsVersion // BMS版本(string)
  }
  if (params.equipmentDetails) { // 设备各种详细属性 equipmentDetails的格式为：{'a': 'b', 'c': 'd'} // 'a'为自定义的键，'b'为a的值
    data['equipmentDetails'] = params.equipmentDetails
  }
  if (params.dataDetailsId && params.dataDetailsId !== '') {
    data['dataDetailsId'] = params.dataDetailsId
    data['dataDetailsSerialNumber'] = params.dataDetailSerialNumber
  }
  return JsonPost(`business/equipment/update`, data)
}

export function saveDataDetails(params) { // 添加数据详情(添加数据表数据)
  const data = {
    sortNum: params.sortNum, // 数据详情序号
    name: params.name, // DataDetail数据详情名称
    equipmentId: params.equipmentId, // 所属设备Id* (String)
    equipmentName: params.equipmentName, // 所属设备名称* (String)
    labelId: params.labelId, // 数据详情labelId(对应采集云labelId) (String)
    labelName: params.labelName // 数据详情labelId(对应采集云labelName) (String)
  }
  data.dataDetailsSerialNumber = params.dataDetailsSerialNumber ? params.dataDetailsSerialNumber : null
  data.contentDataDetails = params.contentDataDetails ? params.contentDataDetails : null
  data.deleted = data.deleted ? data.deleted : false
  return JsonPost(`business/datadetails/add`, data)
}

export function findDataDetails(params) { // 查询数据详情(查询数据表数据)
  const data = {
    equipmentId: params.equipmentId, // 所属设备Id* (String)
    equipmentName: params.equipmentName // 所属设备名称* (String)
  }
  if (params) {
    if (params.name) data['name'] = params.name // DataDetail数据详情名称
    if (params.dataDetailsSerialNumber) data['dataDetailsSerialNumber'] = params.dataDetailsSerialNumber // 数据详情 自定义id(String)
    // dataDetailsSerialNumber参数,暂未决定作用,可传可不传(add 20201114)
    if (params.labelId) data['labelId'] = params.labelId // 数据详情labelId(对应采集云labelId) (String)
    if (params.labelName) data['labelName'] = params.labelName // 数据详情labelId(对应采集云labelName) (String)
    if (params.contentDataDetails && params.contentDataDetails.length !== 0 && Array.isArray(params.contentDataDetails)) { // DataDetail数据详情名称
      const contentDataDetails = []
      params.contentDataDetails.forEach(item => {
        contentDataDetails.push(item)
      })
      data['contentDataDetails'] = contentDataDetails
    }
  }
  return JsonPost(`business/datadetails/find`, data)
}

export function deleteDataDetails(params) { // 删除数据详情(删除数据表数据)
  return JsonPost(`business/datadetails/delete`, {
    id: params.id
  })
}

export function modifyDataDetails(params) { // 修改数据详情(修改数据表数据)
  const data = {
    id: params.id,
    sortNum: params.sortNum, // 数据详情序号
    name: params.name, // DataDetail数据详情名称
    equipmentId: params.equipmentId, // 所属设备Id* (String)
    equipmentName: params.equipmentName, // 所属设备名称* (String)
    labelId: params.labelId, // 数据详情labelId(对应采集云labelId) (String)
    labelName: params.labelName // 数据详情labelId(对应采集云labelName) (String)
  }
  data.dataDetailsSerialNumber = params.dataDetailsSerialNumber ? params.dataDetailsSerialNumber : null
  data.contentDataDetails = params.contentDataDetails ? params.contentDataDetails : null
  data.deleted = data.deleted ? data.deleted : false
  return JsonPost(`business/datadetails/update`, data)
}

export function setEquipmentConcernUsers(params) { // 设备关注用户
  return JsonPost(`business/equipment/setEquipmentConcernUsers`, {
    id: params.id, // 设备id* (String)
    concernUserIds: params.concernUserIds // 设备关注用户id数组* (ArrayList)
  })
}

export function deleteEquipmentConcernUsers(params) { // 删除关注用户
  return JsonPost(`business/equipment/deleteEquipmentConcernUsers`, {
    id: params.id, // 设备id* (String)
    concernUserIds: params.concernUserIds // 设备关注用户id数组* (ArrayList)
  })
}

export function setEquipmentControlUsers(params) { // 设备控制用户
  return JsonPost(`business/equipment/setEquipmentControlUsers`, {
    id: params.id, // 设备id* (String)
    controlUserIds: params.controlUserIds // 设备控制用户id数组* (ArrayList)
  })
}

export function deleteEquipmentControlUsers(params) { // 删除控制用户
  return JsonPost(`business/equipment/deleteEquipmentControlUsers`, {
    id: params.id, // 设备id* (String)
    controlUserIds: params.controlUserIds // 设备关注用户id数组* (ArrayList)
  })
}

export function setEquipmentCameras(params) { // 设备下录像路径添加/修改
  return JsonPost(`business/equipment/setEquipmentCameras`, {
    id: params.id, // 设备id* (String)
    videoProtoType: params.videoProtoType, // 视频接入方式*（string）
    videoUrls: params.videoUrls // 设备摄像头监控地址* (ArrayList)
  })
}

export function getlabel(params) { // 获取标签列表
  const Function = 'intact'
  const ApiKey = params.ApiKey ? params.ApiKey : ''
  const Name = params.Name ? params.Name : ''
  return JsonPost(`/tolink/stream/getlabel?Function=${Function}&Name=${Name}&ApiKey=${ApiKey}`)
}

export function addEqtCmrForAllVdeStm(params) { // 添加设备视频监控
  let videoSystemType = ''
  params.videoSystemTypeArr.forEach(item => {
    if (videoSystemType === '') {
      videoSystemType = item
    } else {
      videoSystemType = '_' + item
    }
  })
  const data = {
    equipmentId: params.equipmentId, // 设备id* (String)
    videoSystemType /*
      设备视频的平台(String),标识指定要进行操作的摄像头平台;
      如果有多个平台的数据操作,用符号 '_' 连接;现有值ISecure、EZVIZ和EZVIZCloud,萤石云传入'EZVIZ'即可;
      传入示例如:ISecure_EZVIZ;此参数
    */
  }
  if (params.iScVideoSerialNumber) data['iScVideoSerialNumber'] = params.iScVideoSerialNumber // 设备摄像头ISecure平台序列号* (ArrayList)
  if (params.iScVideoProtoType) data['iScVideoProtoType'] = params.iScVideoProtoType // 设备摄像头视频协议* (String) 取值范围(rtsp,rtmp)
  if (params.videoProtoType) data['videoProtoType'] = params.videoProtoType // 设备视频接入方式* (String)
  if (params.videoUrls) data['videoUrls'] = params.videoUrls // 设备摄像头EZVIZ监控地址* (ArrayList)
  if (params.ezvizCloud) data['ezvizCloud'] = params.ezvizCloud /*
    设备摄像头萤石云云视频(EZVIZCloud)数据* (ArrayList<HashMap>)
    数据格式为 ezvizCloud: [{ videoName: '', videoType: '1', url: '', AccessToken: ''}]
  */
  return JsonPost(`business/equipment/addEqtCmrForAllVdeStm`, data)
}

export function modifyEqtCmrForAllVdeStm(params) { // 设置设备视频监控
  let videoSystemType = ''
  params.videoSystemTypeArr.forEach(item => {
    if (videoSystemType === '') {
      videoSystemType = item
    } else {
      videoSystemType = '_' + item
    }
  })
  const data = {
    equipmentId: params.equipmentId, // 设备id* (String)
    videoSystemType /*
      设备视频的平台(String),标识指定要进行操作的摄像头平台;
      如果有多个平台的数据操作,用符号 '_' 连接;现有值ISecure、EZVIZ和EZVIZCloud,萤石云传入'EZVIZ'即可;
      传入示例如:ISecure_EZVIZ;此参数
    */
  }
  if (params.iScVideoSerialNumber) data['iScVideoSerialNumber'] = params.iScVideoSerialNumber // 设备摄像头ISecure平台序列号* (ArrayList)
  if (params.iScVideoProtoType) data['iScVideoProtoType'] = params.iScVideoProtoType // 设备摄像头视频协议* (String) 取值范围(rtsp,rtmp)
  if (params.videoProtoType) data['videoProtoType'] = params.videoProtoType // 设备视频接入方式* (String)
  if (params.videoUrls) data['videoUrls'] = params.videoUrls // 设备摄像头EZVIZ监控地址* (ArrayList)
  if (params.ezvizCloud) data['ezvizCloud'] = params.ezvizCloud /*
    设备摄像头萤石云云视频(EZVIZCloud)数据* (ArrayList<HashMap>)
    数据格式为 ezvizCloud: [{ videoName: '', videoType: '1', url: '', AccessToken: ''}]
  */
  return JsonPost(`business/equipment/modifyEqtCmrForAllVdeStm`, data)
}

export function findEqtCmrForAllVdeStm(params) { // 查询设备视频监控
  return JsonPost(`business/equipment/findEqtCmrForAllVdeStm`, {
    equipmentId: params.equipmentId // 设备id* (String)
  })
}

export function requestRedirect(params) { // 查询第三方平台数据的接口
  const header = {}
  for (const i in params.header) {
    header[i] = params.header[i]
  }
  let body = {}
  const dataArr = params.dataArr // 参数为数组
  if (dataArr) {
    body = dataArr
  } else {
    for (const i in params.body) {
      body[i] = params.body[i]
    }
  }

  return JsonPost(`business/cameraDataGet/requestRedirect`, {
    url: params.url, // 请求路径
    method: params.method, // 请求方法 GET、POST(注意大小写)
    header, // 请求头
    body // 请求参数
  })
}
