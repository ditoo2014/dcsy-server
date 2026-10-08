import { JsonPost } from '@/api/api.js'

export function gettrigger(params) { // 获取触发器列表
  return JsonPost(`business/trigger/find`, {
    equipmentId: params.equipmentId
  })
}

export function createtrigger(params) { // 创建触发器
  const logics = [] // 触发告警的逻辑
  params.logics.forEach(item => {
    logics.push({
      logic: item.logic,
      triggerValue: Number(item.triggerValue)
    })
  })
  const strMatchs = []
  params.strMatchs.forEach(item => {
    strMatchs.push({
      matchStr: item.matchStr, // 匹配字符串
      replaceCodeFlag: item.ReplaceCodeFlag, // 是否替换代码
      replaceCode: item.replaceCode // 替换代码
    })
  })
  const statisticValue = params.statisticConfig.statisticValue
  const data = {
    apiKey: params.apiKey,
    siteId: params.siteId, // 站址名称
    equipmentId: params.equipmentId, // 设备id
    code: params.code, // 告警代码
    labelName: params.labelName, // 数据名称
    labelId: params.labelId, // 数据id
    cycle: Number(params.cycle), // 触发间隔
    triggerType: params.triggerType, // 触发器类型(0:告警, 1:统计, 2: 告警和统计)
    formula: 'x', // 计算公式
    logics, // 触发告警的逻辑
    strMatchs, // 匹配字符串
    alarmUser: {
      alarmCode: params.alarmUser.alarmCode,
      alarmStage: params.alarmUser.alarmStage, // 告警等级
      alarmType: params.alarmUser.alarmType, // 告警类型('0':普通, '1':预警)
      alarmChannel: params.alarmUser.alarmChannel,
      alarmUsers: params.alarmUser.alarmUsers
    },
    statisticConfig: {
      useStatisticValue: params.statisticConfig.useStatisticValue, // 是否使用统计值(true, false)
      statisticCode: params.code, // 告警代码
      statisticType: params.statisticConfig.statisticType, // 统计方式 '0':通用统计 '1':告警条数统计
      statisticValue: statisticValue ? parseFloat(statisticValue) : 0, // 统计值(int)
      statisticValueType: params.statisticConfig.statisticValueType // 数据类型 0:累计值 1:增量值
    }
  }
  return JsonPost(`business/trigger/add`, data)
}

export function deletetrigger(params) { // 删除触发器列表或详细信息
  return JsonPost(`business/trigger/delete`, {
    id: params.id
  })
}

export function debugtrigger(params) { // 触发器调试
  const Apikey = params.Apikey
  const Type = params.Type // 可取: 'result' 结果式获取； 'logic' 结果式调试
  const datacome = {}
  params.datacome.forEach(item => {
    datacome[item.key] = item.debugValue
  })
  const result = {}
  if (params.result) {
    params.result.forEach(item => {
      result[item.ident] = item.formula
    })
  }
  return JsonPost(`/tolink/go/api/debugtrigger?Apikey=${Apikey}&Type=${Type}`, {
    datacome,
    logic: params.logic,
    result
  })
}

export function trigGetdata(params) { // 触发器获取数据
  const Apikey = params.Apikey
  const Function = params.Function // 可取值:formula
  const mapping = {}
  params.mapping.forEach(item => {
    mapping[item.value] = item.key
  })
  const datacome = {}
  params.datacome.forEach(item => {
    datacome[item.key] = item.regFormula
  })
  return JsonPost(`/tolink/go/api/getdata?Apikey=${Apikey}&Function=${Function}`, {
    mapping,
    datacome
  })
}

export function updatetrigger(params) { // 修改触发器
  const logics = [] // 触发告警的逻辑
  params.logics.forEach(item => {
    logics.push({
      id: item.id,
      logic: item.logic,
      triggerValue: Number(item.triggerValue)
    })
  })
  const strMatchs = []
  params.strMatchs.forEach(item => {
    strMatchs.push({
      id: item.id,
      matchStr: item.matchStr, // 匹配字符串
      replaceCodeFlag: item.ReplaceCodeFlag, // 是否替换代码
      replaceCode: item.replaceCode // 替换代码
    })
  })
  const statisticValue = params.statisticConfig.statisticValue
  const data = {
    id: params.id,
    apiKey: params.apiKey,
    siteId: params.siteId, // 站址名称
    equipmentId: params.equipmentId, // 设备id
    code: params.code, // 告警代码
    labelName: params.labelName, // 数据名称
    labelId: params.labelId, // 数据id
    cycle: Number(params.cycle), // 触发间隔
    triggerType: params.triggerType, // 触发器类型(0:告警, 1:统计, 2: 告警和统计)
    formula: 'x', // 计算公式
    logics, // 触发告警的逻辑
    strMatchs, // 匹配字符串
    alarmUser: {
      id: params.alarmUser.id,
      alarmCode: params.alarmUser.alarmCode,
      alarmStage: params.alarmUser.alarmStage, // 告警等级
      alarmType: params.alarmUser.alarmType, // 告警类型('0':普通, '1':预警)
      alarmChannel: params.alarmUser.alarmChannel,
      alarmUsers: params.alarmUser.alarmUsers
    },
    statisticConfig: {
      useStatisticValue: params.statisticConfig.useStatisticValue, // 是否使用统计值
      statisticCode: params.code, // 告警代码
      statisticType: params.statisticConfig.statisticType, // 统计方式 '0':通用统计 '1':告警条数统计
      statisticValue: statisticValue ? parseFloat(statisticValue) : 0, // 统计值(int)
      statisticValueType: params.statisticConfig.statisticValueType // 数据类型 0:累计值 1:增量值
    }
  }
  return JsonPost(`business/trigger/update`, data)
}

export async function findAlarmUsers(params) { // 业务云查询告警设置信息
  const data = {}
  if (params.siteId && params.siteId !== '') data['siteId'] = params.siteId // 站址名称*(String)
  if (params.siteName && params.siteName !== '') data['siteName'] = params.siteName // 站址名称*(String)
  if (params.equipmentId && params.equipmentId !== '') data['equipmentId'] = params.equipmentId // 设备id*(String)
  if (params.equipmentName && params.equipmentName !== '') data['equipmentName'] = params.equipmentName // 设备名称*(String)
  if (params.coding && params.coding !== '') data['coding'] = params.coding // 触发器代码*(String)
  if (params.shortMsgAlarm && params.shortMsgAlarm !== '') data['shortMsgAlarm'] = params.shortMsgAlarm // 接收人员接收告警方式* (Boolean) 是否接收短信告警
  if (params.shortMsgAlarmUserIds && params.shortMsgAlarmUserIds !== '') data['shortMsgAlarmUserIds'] = params.shortMsgAlarmUserIds // 接收短信告警人员ids* (ArrayList)用户id数组

  const res = await JsonPost(`business/alarmUsers/findAlarmUsers`, data)
  return { ...res, ...(res.return || {}) }
}

export function deleteAlarmUsers(params) { // 告警接收人员设置删除
  return JsonPost(`business/alarmUsers/deleteAlarmUsers`, {
    id: params.id // 查询告警接收人员返回的ReceiveCollectAlarmUsers的id* (String)
  })
}

export function batchDeleteAlarmUsers(params) { // 批量删除触发器告警人员信息
  const data = {
    equipmentId: params.equipmentId, // 设备id*(String)
    codings: params.codings // 批量选择的触发器编码* (ArrayList)
  }
  return JsonPost(`business/alarmUsers/batchDeleteAlarmUsers`, data)
}

export function findAlarmRecord(params) { // 查询告警和预警记录
  const pageNum = params.pageNum ? params.pageNum : 1
  const pageSize = params.pageSize ? params.pageSize : 10
  const corpId = params.corpId
  let query = `?pageNum=${pageNum}&pageSize=${pageSize}`
  if (corpId || corpId === 0) {
    query += `&corpId=${corpId}`
  }
  if (params.startTime != null && params.endTime != null) {
    query += `&startTime=${params.startTime}&endTime=${params.endTime}`
  }
  const data = {
    siteId: params.siteId != null ? params.siteId : null,
    equipmentId: params.equipmentId != null ? params.equipmentId : null,
    alarmStatus: params.alarmStatus,
    alarmType: params.alarmType
  }
  return JsonPost(`business/alarmrecord/find${query}`, data)
}

export function countbyequipment(params) { // 查询多个告警或预警记录
  const data = {
    equipmentIds: params.equipmentIds, // 设备id*(String)
    alarmStatus: params.alarmStatus, // 告警状态*(Number) 1:实时,0:历史
    alarmType: params.alarmType // 告警类型*(Number) 0: 告警, 1: 预警
  }
  return JsonPost(`business/alarmrecord/countbyequipment`, data)
}

export function getAlarmStatusByEquipment(params) { // 一次查询多个设备的告警和预警状态
  return JsonPost(`business/alarmrecord/statusbyequipment`, {
    equipmentIds: params.equipmentIds,
    alarmStatus: params.alarmStatus
  })
}

export function deleteAlarmRecord(params) { // 删除告警记录
  const pageNum = params.pageNum ? params.pageNum : 1
  const pageSize = params.pageSize ? params.pageSize : 10
  const query = `?pageNum=${pageNum}&pageSize=${pageSize}`
  const data = {
    siteId: params.siteId, // 站址名称*(String)
    equipmentId: params.equipmentId, // 设备id*(String)
    alarmStatus: params.alarmStatus, // 告警状态*(Number) 1:实时,0:历史
    alarmType: params.alarmType // 告警类型*(Number) 0: 告警, 1: 预警
  }
  return JsonPost(`business/alarmrecord/delete${query}`, data)
}
