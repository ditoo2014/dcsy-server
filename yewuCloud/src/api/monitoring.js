// 监控页面接口集中在本文件；文件改名不改变请求路径和参数。
import axios from 'axios'
import { JsonPost } from '@/api/api.js'

import { findDataDetails } from '@/api/equipment.js'
import { getTime } from '@/utils/utils.js'
import { hasPointValue, normalizePointTime, resolveCollectStatus } from '@/utils/collectionStatus.js'

export function findEquipmentBySiteGroupStage02(params) { // 获取监控列表
  const data = {}
  const equipment = {}
  if (params) {
    if (params.equipmentName && params.equipmentName !== '') equipment['equipmentName'] = params.equipmentName // 设备名称(String)
    if ((params.belongToSiteId || params.belongToSiteId === 0) && params.belongToSiteId !== '') equipment['belongToSiteId'] = params.belongToSiteId // 选择的站址ObjectId
    if (params.belongToSiteName && params.belongToSiteName !== '') equipment['belongToSiteName'] = params.belongToSiteName // 选择的站址名称(String)
    if (params.equipmentTypeId && params.equipmentTypeId !== '') equipment['equipmentTypeId'] = params.equipmentTypeId // 设备类型ObjectId
    if (params.equipmentTypeName && params.equipmentTypeName !== '') equipment['equipmentTypeName'] = params.equipmentTypeName // 设备类型名称(String)
    if (params.runningStatus && params.runningStatus !== '' || params.runningStatus === 0) equipment['runningStatus'] = params.runningStatus // 设备运行状态(Integer) 1:正常运行, 0:为异常
    if (params.alarmTriggerNum && params.alarmTriggerNum !== '' || params.alarmTriggerNum === 0) equipment['alarmTriggerNum'] = params.alarmTriggerNum // 设备告警(触发器)个数(Integer)
    if (params.equipmentSerialNumber && params.equipmentSerialNumber !== '') equipment['equipmentSerialNumber'] = params.equipmentSerialNumber // 设备告警(触发器)个数(Integer)
    if (params.codeRule && params.codeRule !== '') equipment['codeRule'] = params.codeRule // 编码规则(用户自定义序列号长度)
    const engStatus = params.engageInternetStatus
    if (engStatus === 0 || (engStatus && engStatus !== '')) equipment['engageInternetStatus'] = engStatus // 并网状态

    if (JSON.stringify(equipment) !== '{}') data['equipment'] = equipment
    if (params.siteGroupIds && params.siteGroupIds.length !== 0) data['siteGroupIds'] = params.siteGroupIds // 站址分组ID组成的数组
  }
  return JsonPost(`business/equipment/findEquipmentBySiteGroupStage02`, data)
}

export function handCheckEquipmentStatus(params) { // 手动校准设备状态
  return JsonPost(`business/equipment/handCheckEquipmentStatus`, {
    id: params.id
  })
}

export function equipmentsStatus(params) { // 获取设备状态
  const data = params.equipIdArr // 设备ID字符串组成的数组
  return JsonPost(`business/equipment/equipmentsStatus`, data)
}

export function getdata(params) { // 获取数据列表数据
  const Apikey = params.Apikey
  const Function = params.Function
  /* 可取值：'countdata'获取采集点数据数量;'formula'根据公式获取数据;'pagetime'根据分页和时间获取历史数据;
     'realtime'获取多个标签的最新数据;为'newdata'获取最新的30个数据; 'status'查询aipKey对应的设备的状态;
     'historydatapasstime'获取历史统计数据；
  */
  const Pagesize = params.Pagesize // 分页大小
  const Pagenum = params.Pagenum // 分页页数
  const Datestart = params.Datestart // 开始时间
  const Dateend = params.Dateend // 结束时间

  let query = `?Apikey=${Apikey}&Function=${Function}`
  let data = {}
  if (Function === 'countdata') {
    query += `&Datestart=${Datestart}&Dateend=${Dateend}`
    data = {
      labelID: params.labelID
    }
  } else if (Function === 'formula') {
    const mapping = {}
    params.mapping.forEach(item => {
      mapping[item.value] = item.key
    })
    const datacome = {}
    params.datacome.forEach(item => {
      datacome[item.key] = item.regFormula
    })
    data = {
      mapping, // 参数 - 标签ID
      datacome // 数据提取
    }
  } else if (Function === 'pagetime') {
    query += `&Pagesize=${Pagesize}&Pagenum=${Pagenum}&Datestart=${Datestart}&Dateend=${Dateend}`
    data = {
      labelID: params.labelID
    }
  } else if (Function === 'realtime') {
    data = params.idArr
  } else if (Function === 'newdata') {
    data = {
      labelID: params.labelID
    }
  } else if (Function === 'historydatapasstime') {
    const timeType = params.timeType // 时间类型（'second','minute','hour'）
    const timeLength = params.timeLength // 时间间隔 number整数
    query += `&Pagesize=${Pagesize}&Pagenum=${Pagenum}&Datestart=${Datestart}&Dateend=${Dateend}&timeLength=${timeLength}&timeType=${timeType}`
    data = {
      labelID: params.labelID
    }
  } else if (Function === 'historydatapasstimelist') {
    data = {
      requests: params.requests
    }
  }
  return JsonPost(`api/getdata${query}`, data)
}

export function getStreamLabelList(apiKey) { // 获取数据池标签及其实际数据点编码
  return JsonPost(`api/getlabel?Apikey=${encodeURIComponent(apiKey || '')}`, {})
}

export async function getRealTimeData(equipData) { // 获取某个设备的实时数据
  const res = await findDataDetails({
    equipmentId: equipData.id // 所属设备Id(String)
  })
  let list = []
  if (res.return && res.return.data) {
    list = res.return.data
  }
  const idArr = []
  list.forEach(labelItem => {
    idArr.push(labelItem.labelId)
  })
  const dataRes = await getdata({
    Apikey: equipData.apiKey,
    Function: 'realtime',
    idArr
  })
  const data = dataRes.return
  if (data) {
    const labels = data.labels
    const times = data.timeList
    const status = data.status
    const values = data.valueList
    if (labels && times && status && values) {
      list.forEach(labelItem => {
        labels.forEach((pointItem, index) => {
          if (labelItem.labelId === pointItem) {
            labelItem.realTimeVal = values[index]
            labelItem.value = values[index]
            labelItem.realTimeDate = getTime(times[index])
            if (status[index]) {
              if (status[index] === -1) {
                labelItem.realTimeVal = '异常'
              }
            } else {
              labelItem.realTimeVal = '异常'
            }
            if (labelItem.realTimeVal && labelItem.realTimeVal !== '异常' && labelItem.realTimeVal !== '' && labelItem.realTimeVal % 1 !== 0) {
              let decimalPlaces = 2
              const contentDataDetails = labelItem.contentDataDetails
              if (contentDataDetails && Array.isArray(contentDataDetails) && contentDataDetails.length !== 0) {
                decimalPlaces = contentDataDetails[0].decimalPlaces
              }
              if (typeof labelItem.realTimeVal === 'number') {
                labelItem.realTimeVal = labelItem.realTimeVal.toFixed(decimalPlaces)
              }
            }
          }
        })
      })
    }
  }
  return list
}

export async function getapilistdata(params) { // 批量设备获取数据点的数据
  const data = []
  const equipList = params.equipList
  equipList.forEach(item => {
    data.push({
      apiKey: item.apiKey,
      requestKey: String(item.id),
      labelList: item.labelList // [ "124117416397381872", "124117416397381873", "124117416397381874"......]
    })
  })
  const res = await JsonPost(`api/getapilistdata`, {
    data
  })
  equipList.forEach(item => {
    item.collectStatus = 0
  })
  if (res.return) {
    const list = res.return.data
    if (Array.isArray(list)) {
      const requestMap = new Map()
      const apiKeyMap = new Map()
      list.forEach(dataItem => {
        if (dataItem.requestKey !== undefined && dataItem.requestKey !== '') {
          requestMap.set(String(dataItem.requestKey), dataItem)
        }
        apiKeyMap.set(dataItem.apikey, dataItem)
      })
      equipList.forEach((item, index) => {
        const indexedItem = list[index] && list[index].apikey === item.apiKey ? list[index] : null
        const dataItem = requestMap.get(String(item.id)) || indexedItem || apiKeyMap.get(item.apiKey)
        const pointData = dataItem && dataItem.data
        const labels = pointData && Array.isArray(pointData.label) ? pointData.label : []
        const times = pointData && Array.isArray(pointData.time) ? pointData.time : []
        const status = pointData && Array.isArray(pointData.status) ? pointData.status : []
        const values = pointData && Array.isArray(pointData.value) ? pointData.value : []
        const pointMap = new Map()

        labels.forEach((pointItem, index) => {
          pointMap.set(String(pointItem), {
            status: status[index],
            value: values[index],
            time: times[index]
          })
        })

        item.dataDetails.forEach(labelItem => {
          const point = pointMap.get(String(labelItem.labelId))
          if (!point) return

          labelItem.realTimeStatus = point.status
          labelItem.realTimeTimestamp = normalizePointTime(point.time)
          labelItem.realTimeDate = labelItem.realTimeTimestamp ? getTime(labelItem.realTimeTimestamp) : ''
          labelItem.realTimeVal = point.value
          labelItem.value = point.value
          if (Number(point.status) !== 1 || !hasPointValue(point.value)) {
            labelItem.realTimeVal = '异常'
          }
          if (labelItem.realTimeVal !== '异常' && labelItem.realTimeVal % 1 !== 0) {
            let decimalPlaces = 2
            const contentDataDetails = labelItem.contentDataDetails
            if (contentDataDetails && Array.isArray(contentDataDetails) && contentDataDetails.length !== 0) {
              decimalPlaces = contentDataDetails[0].decimalPlaces
            }
            if (typeof labelItem.realTimeVal === 'number') {
              labelItem.realTimeVal = labelItem.realTimeVal.toFixed(decimalPlaces)
            }
          }
        })
        item.collectStatus = resolveCollectStatus(item, pointMap, dataItem)
      })
    }
  }
  return {
    res,
    equipList
  }
}

export function getGatewayData(params) { // 通过网关id获取数据点
  const Apikey = params.Apikey
  const GatewayID = params.GatewayID // 网关ID
  const Function = params.Function /* 可取值为 countdata 获取采集点数据数量;为 intact 获取详细列表;为 list 获取数据点ID列表;
  为 historydata 获取单个数据点历史数据;为 realtimedata 获取节点下多个数据点的即时数据;为 newdata 获取最新的30个数据;
  datalist 查询某个时刻所有数据点的值 */

  const Formula = params.Formula // 处理公式
  const Pagenum = params.Pagenum // 页码
  const Pagesize = params.Pagesize // 一页的数据条数
  const Datestart = params.Datestart // 开始时间
  const Dateend = params.Dateend // 结束时间
  const dataId = params.dataId
  const Time = params.Time
  let query = `?Apikey=${Apikey}&GatewayID=${GatewayID}&Function=${Function}`
  let data = {}

  if (Function === 'countdata') {
    query += `&Datestart=${Datestart}&Dateend=${Dateend}`
    data = { dataId }
  } else if (Function === 'historydata') {
    query += `&Formula=${Formula}&Pagenum=${Pagenum}&Pagesize=${Pagesize}&Datestart=${Datestart}&Dateend=${Dateend}`
    data = { dataId }
  } else if (Function === 'realtimedata') {
    query += `&Formula=${Formula}`
    data = params.dataIdArray
  } else if (Function === 'newdata') {
    data = { dataId }
  } else if (Function === 'datalist') {
    query += `&Time=${Time}`
  }
  return JsonPost(`/tolink/go/api/getdata2${query}`, data)
}

export function getGatewayModule2(params) { // 获取虚拟网关数据填报数据点配置
  const Apikey = params.Apikey
  const GatewayID = params.GatewayID // 网关ID
  const query = `?Apikey=${Apikey}&GatewayID=${GatewayID}`
  return JsonPost(`/tolink/go/api/getmodule2${query}`, {})
}

export function gatewaycommand(params) { // 数据列表写数据
  const Apikey = params.Apikey
  const Function = params.Function
  const query = `?Apikey=${Apikey}&Function=${Function}`
  return JsonPost(`/tolink/go/api/command${query}`, {
    data: params.data,
    value: parseFloat(params.value)
  })
}

export function findCollectCycleTriggerData(params) { // 获取定时触发器告警信息
  const equipmentSerialNumber = []
  if (params.equipmentSerialNumber) {
    params.equipmentSerialNumber.forEach(item => {
      equipmentSerialNumber.push({
        id: item.id, symbel: item.symbel
      })
    })
  }
  let query = `?pageNumber=${params.pageNumber}&pageSize=${params.pageSize}`
  if (params.startTime) query += `&startTime=${params.startTime}` // 首个触发的时间
  if (params.endTime) query += `&endTime=${params.endTime}` // 最后一个触发的时间

  const data = {}
  if (params.id) data['id'] = params.id // 数据库id
  if (params.equipmentSerialNumber) data['equipmentSerialNumber'] = equipmentSerialNumber // 设备自定义序列号(hashmap数组)
  if (params.equipmentId) data['equipmentId'] = params.equipmentId // 设备id
  if (params.equipmentName) data['equipmentName'] = params.equipmentName // 设备名称(String)
  if (params.siteSerialNumber) data['siteSerialNumber'] = params.siteSerialNumber // 站址自定义序列号(String)
  if (params.siteName) data['siteName'] = params.siteName // 站址名称(String)
  if (params.coding) data['coding'] = params.coding // 触发器代码(String)
  if (params.begin) data['begin'] = params.begin // 处理开始执行时间(Long)
  if (params.end) data['end'] = params.end // 处理结束执行时间(Long)
  if (params.value) data['value'] = params.value // 逻辑运算结束(Boolean)
  if (params.result) data['result'] = params.result // 处理结果数据(hashmap数组)
  if (params.err) data['err'] = params.err // 不为""表示异常,异常信息(String)

  return JsonPost(`business/collect/findCollectCycleTriggerData${query}`, data)
}

export function findCollectLiveTriggerData(params) { // 获取实时触发器告警信息
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
  if (params.equipmentId) data['equipmentId'] = params.equipmentId // 设备id
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
  return JsonPost(`business/collect/findCollectLiveTriggerData${query}`, data)
}

export function findCollectLiveTriggerDataExport(params) { // 获取实时触发器告警信息
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
  if (params.equipmentId) data['equipmentId'] = params.equipmentId // 设备id
  if (params.equipmentName) data['equipmentName'] = params.equipmentName // 设备名称(String)
  if (params.siteSerialNumber) data['siteSerialNumber'] = params.siteSerialNumber // 站址自定义序列号(String)
  if (params.siteName) data['siteName'] = params.siteName // 站址名称(String)
  if (params.coding) data['coding'] = params.coding // 触发器代码(String)
  if (params.begin) data['begin'] = params.begin // 处理开始执行时间(Long)
  if (params.end) data['end'] = params.end // 处理结束执行时间(Long)
  if (params.value) data['value'] = params.value // 逻辑运算结束(Boolean)
  if (params.result) data['result'] = params.result // 处理结果数据(hashmap数组)
  if (params.err) data['err'] = params.err // 不为""表示异常,异常信息(String)
  return JsonPost(`business/collect/findCollectLiveTriggerDataExport${query}`, data)
}

export function removeCollectCycleTriggerData(params) { // 删除单条定时触发器告警信息
  return JsonPost(`business/collect/removeCollectCycleTriggerData`, {
    id: params.id // 触发器告警数据的id
  })
}

export function removeCollectLiveTriggerData(params) { // 删除单条实时触发器告警信息
  return JsonPost(`business/collect/removeCollectLiveTriggerData`, {
    id: params.id // 触发器告警数据的id
  })
}

export function removeCollectBaseDataByCode(params) { // 根据触发器代码删除触发器告警信息
  return JsonPost(`business/collect/removeCollectBaseDataByCode`, {
    id: params.id,
    coding: params.coding // 触发器代码
  })
}

export function findEquipmentRecord(params) { // 台账记录查询
  return JsonPost(`business/equipment/findEquipmentRecord`, {
    equipmentId: params.equipmentId // 可以用设备id查询
  })
}

export function addEquipmentRecord(params) { // 台账记录添加
  return JsonPost(`business/equipment/addEquipmentRecord`, {
    equipmentId: params.equipmentId, // 设备id* (String)
    equipmentName: params.equipmentName, // 设备名称* (String)
    recordTime: params.recordTime, // 台账记录时间* (Long)
    recordName: params.recordName, // 台账名称* (String)
    recordType: params.recordType, // 台账类型* (String)
    recordManName: params.recordManName // 记录人* (String)
  })
}

export function addEquipmentRecordFile(params) { // 台账记录上传附件（上传文件）
  const equipmentRecordId = params.equipmentRecordId
  const formData = new FormData()
  formData.append('file', params.file) // file要与接口字段对应
  return JsonPost(`business/equipment/addEquipmentRecordFile?equipmentRecordId=${equipmentRecordId}`, formData)
}

export function recordFileDownload(params) { // 下载文件
  const equipmentRecordId = params.equipmentRecordId
  const filename = params.fileNewName
  return JsonPost(`business/equipment/recordFileDownload?equipmentRecordId=${equipmentRecordId}&filename=${filename}`)
}

export function saveEquipmentCumulativeData(params) { // 数据累计添加
  return JsonPost(`business/equipmentCumulativeData/saveEquipmentCumulativeData`, {
    equipmentId: params.equipmentId, // 设备id* (String)
    equipmentName: params.equipmentName, // 设备名称* (String)
    cumulativeTime: params.cumulativeTime, // 日期时间* (Long)
    cumulativeFlow: params.cumulativeFlow, // 累计流量* (Long)
    cumulativeMedicine: params.cumulativeMedicine, // 累计加药* (Long)
    cumulativeDataDetail: params.cumulativeDataDetail // 其他详细信息(HashMap)
  })
}

export function modifyEquipmentCumulativeData(params) { // 数据累计修改
  return JsonPost(`business/equipmentCumulativeData/modifyEquipmentCumulativeData`, {
    id: params.id,
    equipmentId: params.equipmentId, // 设备id* (String)
    equipmentName: params.equipmentName, // 设备名称* (String)
    cumulativeTime: params.cumulativeTime, // 日期时间* (Long)
    cumulativeFlow: params.cumulativeFlow, // 累计流量* (Long)
    cumulativeMedicine: params.cumulativeMedicine, // 累计加药* (Long)
    cumulativeDataDetail: params.cumulativeDataDetail // 其他详细信息(HashMap)
  })
}

export function deleteEquipmentCumulativeData(params) { // 数据累计删除
  return JsonPost(`business/equipmentCumulativeData/deleteEquipmentCumulativeData`, {
    id: params.id
  })
}

export function findEquipmentCumulativeData(params) { // 数据累计查询
  const data = {}
  if (params.id) data.id = params.id
  if (params.equipmentId) data.equipmentId = params.equipmentId // 设备id* (String)
  if (params.equipmentName) data.equipmentName = params.equipmentName // 设备名称* (String)
  if (params.cumulativeTime) data.cumulativeTime = params.cumulativeTime // 日期时间* (Long)
  if (params.cumulativeFlow) data.cumulativeFlow = params.cumulativeFlow // 累计流量* (Long)
  if (params.id) data.cumulativeMedicine = params.cumulativeMedicine // 累计加药* (Long)
  if (params.id) data.cumulativeDataDetail = params.cumulativeDataDetail // 其他详细信息(HashMap)

  return JsonPost(`business/equipmentCumulativeData/findEquipmentCumulativeData`, data)
}

export function getYSYToken(params) { // 获取萤石云Token
  const appKey = params.appKey
  const appSecret = params.appSecret
  const query = `appKey=${appKey}&appSecret=${appSecret}`
  return new Promise((resolve, reject) => {
    axios({
      method: 'post',
      'Content-Type': 'application/x-www-form-urlencoded',
      url: `https://open.ys7.com/api/lapp/token/get?${query}`
    }).then(res => {
      resolve(res.data)
    })
  })
}

export function controlYSYCamera(params) { // 控制萤石云摄像头,文档地址: https://open.ys7.com/doc/zh/book/index/device_ptz.html
  const accessToken = params.accessToken // 授权过程获取的access_token
  const deviceSerial = params.deviceSerial // 设备序列号,存在英文字母的设备序列号，字母需为大写
  const channelNo = params.channelNo // 通道号
  const direction = params.direction // 操作命令：0-上，1-下，2-左，3-右，4-左上，5-左下，6-右上，7-右下，8-放大，9-缩小，10-近焦距，11-远焦距
  const speed = params.speed ? params.speed : 1 // 云台速度：0-慢，1-适中，2-快，海康设备参数不可为0
  const query = `accessToken=${accessToken}&deviceSerial=${deviceSerial}&channelNo=${channelNo}&direction=${direction}&speed=${speed}`
  return new Promise((resolve, reject) => {
    axios({
      method: 'post',
      'Content-Type': 'application/x-www-form-urlencoded',
      url: `https://open.ys7.com/api/lapp/device/ptz/start?${query}`
    }).then(res => {
      resolve(res.data)
    })
  })
}

export function stopControlYSYCamera(params) { // 停止控制萤石云摄像头,文档地址: https://open.ys7.com/doc/zh/book/index/device_ptz.html
  const accessToken = params.accessToken // 授权过程获取的access_token
  const deviceSerial = params.deviceSerial // 设备序列号,存在英文字母的设备序列号，字母需为大写
  const channelNo = params.channelNo // 通道号
  const direction = params.direction // 操作命令：0-上，1-下，2-左，3-右，4-左上，5-左下，6-右上，7-右下，8-放大，9-缩小，10-近焦距，11-远焦距
  const query = `accessToken=${accessToken}&deviceSerial=${deviceSerial}&channelNo=${channelNo}&direction=${direction}`
  return new Promise((resolve, reject) => {
    axios({
      method: 'post',
      'Content-Type': 'application/x-www-form-urlencoded',
      url: `https://open.ys7.com/api/lapp/device/ptz/stop?${query}`
    }).then(res => {
      resolve(res.data)
    })
  })
}

export function findCltLiveEquipmentDataStsTlyIncData(params) { // 用气统计
  const startTime = params.startTime // 开始时间戳 (s)
  const endTime = params.endTime // 结束时间戳 (s)
  const query = `?startTime=${startTime}&endTime=${endTime}`
  const data = {
    timeType: params.timeType, // 时间类型(值: daily 、 monthly )*(String)
    coding: params.coding ? params.coding : '', // 设备统计编码
    dataType: params.dataType // 数据类型，累计数据: incremental,增量数据: cumulative
  }
  if (params.siteId) data['siteId'] = params.siteId // 站址名称*(String)
  if (params.equipmentId) data['equipmentId'] = params.equipmentId // 设备id*(String)
  if (params.statisticObject) data['statisticObject'] = params.statisticObject // 数据统计对象id*(String)
  return JsonPost(`business/collect/statistic/findCltLiveEquipmentDataStsTlyIncData${query}`, data)
}

export function findCltLiveEquipmentDataStsTlyOrgData(params) { // 用气统计 (在工艺设施的统计设置中进行配置)
  const startTime = params.startTime // 开始时间戳 (s)
  const endTime = params.endTime // 结束时间戳 (s)
  const query = `?startTime=${startTime}&endTime=${endTime}`
  const data = {
    timeType: params.timeType, // 时间类型(值: daily 、 monthly )*(String)
    coding: params.coding ? params.coding : '', // 设备统计编码
    dataType: params.dataType // 数据类型，累计数据:cumulative,增量数据:incremental
  }
  if (params.siteId) data['siteId'] = params.siteId // 客户id*(String)
  if (params.equipmentId) data['equipmentId'] = params.equipmentId // 设备id*(String)
  if (params.statisticObject) data['statisticObject'] = params.statisticObject // 查询统计数据对象*(String)(值:site、equipment;*默认值site,如果查询设备必传equipment)
  return JsonPost(`business/collect/statistic/findCltLiveEquipmentDataStsTlyOrgData${query}`, data)
}

export function equipmentcirlog(params) { // 电池流转记录
  return JsonPost(`business/equipmentcirlog/find`, {
    equipmentId: params.equipmentId
  })
}

export { streamCreate, streamDelete } from '@/api/stream.js'
