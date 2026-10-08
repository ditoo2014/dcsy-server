import { JsonPost } from '@/api/api.js'

function buildLogics(logics) {
  return (logics || []).map(item => {
    const row = {
      logic: item.logic,
      triggerValue: Number(item.triggerValue)
    }
    if (item.id) {
      row.id = item.id
    }
    return row
  })
}

function buildStrMatchs(strMatchs) {
  return (strMatchs || []).map(item => {
    const row = {
      matchStr: item.matchStr,
      replaceCodeFlag: !!item.replaceCodeFlag,
      replaceCode: item.replaceCode || ''
    }
    if (item.id) {
      row.id = item.id
    }
    return row
  })
}

function buildPayload(params) {
  const statisticValue = params.statisticConfig?.statisticValue
  const equipmentIds = (params.equipmentIds || []).map(id => Number(id)).filter(id => !Number.isNaN(id))
  const data = {
    equipmentIds,
    triggerName: params.triggerName || '',
    labelName: params.labelName,
    triggerType: params.triggerType,
    triggerModel: params.triggerModel ?? 0,
    code: params.code,
    cycle: Number(params.cycle) || 0,
    sustainTime: Number(params.sustainTime) || 0,
    formula: params.formula || 'x',
    logics: buildLogics(params.logics),
    strMatchs: buildStrMatchs(params.strMatchs),
    ignoreLogic: !!params.ignoreLogic,
    alarmUser: {
      alarmCode: params.alarmUser?.alarmCode || '',
      alarmStage: params.alarmUser?.alarmStage ?? 1,
      alarmType: params.alarmUser?.alarmType || '0',
      // 当前独立告警程序只保留短信渠道；接收人必须由用户实际选择。
      alarmChannel: params.alarmUser?.alarmChannel || '0',
      alarmUsers: params.alarmUser?.alarmUsers || ''
    },
    statisticConfig: {
      useStatisticValue: !!params.statisticConfig?.useStatisticValue,
      statisticCode: params.code,
      statisticType: params.statisticConfig?.statisticType || '0',
      statisticValue: statisticValue ? parseFloat(statisticValue) : 0,
      statisticValueType: params.statisticConfig?.statisticValueType ?? 0,
      dataSourceType: params.statisticConfig?.dataSourceType ?? 0
    }
  }
  if (params.alarmUser?.id) {
    data.alarmUser.id = params.alarmUser.id
  }
  if (params.statisticConfig?.id) {
    data.statisticConfig.id = params.statisticConfig.id
  }
  if (params.id) {
    data.id = params.id
  }
  return data
}

export function findGeneralTrigger() {
  return JsonPost('business/generaltrigger/find', {})
}

export function addGeneralTrigger(params) {
  return JsonPost('business/generaltrigger/add', buildPayload(params))
}

export function updateGeneralTrigger(params) {
  return JsonPost('business/generaltrigger/update', buildPayload(params))
}

export function deleteGeneralTrigger(params) {
  return JsonPost('business/generaltrigger/delete', {
    id: params.id
  })
}

export function appliesToEquipment(trigger, equipmentId) {
  const id = Number(equipmentId)
  if (!id || !trigger) {
    return false
  }
  const ids = trigger.equipmentIds || []
  if (ids.length === 0) {
    return (trigger.marktriggers || []).some(item => Number(item.equipmentId) === id)
  }
  return ids.some(eid => Number(eid) === id) ||
    (trigger.marktriggers || []).some(item => Number(item.equipmentId) === id)
}
