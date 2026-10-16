import { JsonPost } from '@/api/api.js'

export function getStatisticdata(params) { // 获取统计池的统计数据
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const filterFunc = params.filterFunc // ConE: 关注设备， BinC: 绑定部门
  const startTime = params.startTime
  const endTime = params.endTime
  const query = `pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}&startTime=${startTime}&endTime=${endTime}`

  const codeList = []
  const corpList = []
  params.codeList.forEach(code => {
    codeList.push(code)
  })
  params.corpList.forEach(corpId => {
    corpList.push(corpId)
  })

  return JsonPost(`business/statisticdata/levelcorp?${query}`, {
    targetType: params.targetType, // 通过指定查询对象 – 0:设备 1:站址 2:部门
    statisticDateType: params.statisticDateType, // 通过指定统计粒度–hourly,daily,monthly
    statisticValueType: params.statisticValueType, // 通过指定统计值类型-0: 累计值 1:增量值
    statisticAlarmType: params.statisticAlarmType, // 通过指定统计告警类型-'0': 告警统计， '1': 预警统计
    codeList, // 统计代码数组 * 指定统计代码 – 只统计对应数据
    corpList // 统计部门数组 * 指定统计部门
  })
}

export function getCountbycorp(params) { // 通过部门统计设备数量
  const filterFunc = params.filterFunc // ConE: 关注设备， BinC: 绑定部门
  const query = `filterFunc=${filterFunc}`
  const corpList = []
  params.corpList.forEach(corpId => {
    corpList.push(corpId)
  })
  return JsonPost(`business/equipment/countbycorp?${query}`, {
    positionStatus: params.positionStatus, // 设备状态 (1:入库；2:出库；3:上站入网；4:退网下站；5:检测待用；6：检测淘汰)
    equipmentName: params.equipmentName, // 设备名称
    corpList // 统计部门数组 * 指定统计部门
  })
}

export function getStatisticdataFind(params) { // 获取统计池的统计数据范围查询接口
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const filterFunc = params.filterFunc // ConE: 关注设备， BinC: 绑定部门
  const startTime = params.startTime
  const endTime = params.endTime
  const query = `pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}&startTime=${startTime}&endTime=${endTime}`

  return JsonPost(`business/statisticdata/find?${query}`, {
    siteId: params.siteId, // 通过站址id查询
    equipmentId: params.equipmentId, // 通过设备id查询
    targetType: params.targetType, // 通过指定查询对象 – 0:设备 1:站址 2:部门
    statisticcode: params.statisticcode, // 统计代码 * 指定统计代码 – 只统计对应数据
    statisticDateType: params.statisticDateType, // 通过指定统计粒度–hourly,daily,monthly
    statisticValueType: params.statisticValueType, // 通过指定统计值类型-0: 累计值 1:增量值
    statisticAlarmType: params.statisticAlarmType, // 通过指定统计告警类型-'0': 告警统计， '1': 预警统计
    corpId: params.corpId // 统计部门 * 指定统计部门
  })
}

export function getEquipmenthltstd() { // 获取电池健康度统计数据
  return JsonPost(`business/equipmenthltstd/find`, {})
}

export function getEquipmentHealthStatistics(params) { // 获取入网电池健康度分布
  return JsonPost(`business/equipment/health/statistics`, {
    corpId: params.corpId
  })
}

export function getDailyAlarmStatistics(params) { // 获取大屏每日告警电池数
  const query = `startTime=${params.startTime}&endTime=${params.endTime}`
  return JsonPost(`business/statisticdata/alarmdaily?${query}`, {
    corpList: params.corpList
  })
}

export function getDailyWarningStatistics(params) { // 获取大屏每日预警电池数
  const query = `startTime=${params.startTime}&endTime=${params.endTime}`
  return JsonPost(`business/statisticdata/warningdaily?${query}`, {
    corpList: params.corpList
  })
}

export function getMarkdatelist(params) { // 获取统计池的统计数据
  const pageNum = params.pageNum
  const pageSize = params.pageSize
  const filterFunc = params.filterFunc // ConE: 关注设备， BinC: 绑定部门
  const startTime = params.startTime
  const endTime = params.endTime
  const query = `pageNum=${pageNum}&pageSize=${pageSize}&filterFunc=${filterFunc}&startTime=${startTime}&endTime=${endTime}`

  const codeList = []
  const corpList = []
  params.codeList.forEach(code => {
    codeList.push(code)
  })
  params.corpList.forEach(corpId => {
    corpList.push(corpId)
  })
  return JsonPost(`business/statisticdata/markdatelist?${query}`, {
    targetType: params.targetType, // 通过指定查询对象 – 0:设备 1:站址 2:部门
    statisticDateType: params.statisticDateType, // 通过指定统计粒度–hourly,daily,monthly
    statisticValueType: params.statisticValueType, // 通过指定统计值类型-0: 累计值 1:增量值
    statisticAlarmType: params.statisticAlarmType, // 通过指定统计告警类型-'0': 告警统计， '1': 预警统计
    codeList, // 统计代码数组 * 指定统计代码 – 只统计对应数据
    corpList // 统计部门数组 * 指定统计部门
  })
}

export function getChargeDischargeDailyStatistics(params) {
  const query = `startTime=${params.startTime}&endTime=${params.endTime}`
  return JsonPost(`business/statisticdata/charge-discharge-daily?${query}`, {
    corpList: params.corpList
  })
}
