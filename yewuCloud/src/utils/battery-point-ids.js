// 电池算法与统计数据点统一使用英文 ID，与模板、后端解析保持一致。
export const REMAINING_DISCHARGE_TIME_POINT_DATA_ID = 'alg_remaining_discharge_time'
export const THERMAL_RUNAWAY_RISK_POINT_DATA_ID = 'alg_thermal_runaway_risk_level'
export const TEMPERATURE_DIFFERENCE_POINT_DATA_ID = 'ditoo_temperature_difference'
export const PRESSURE_POINT_DATA_ID = 'ditoo_voltage_difference'
export const TOTAL_DISCHARGE_CAPACITY_POINT_DATA_ID = 'tt_total_discharge_capacity'
export const CYCLE_COUNT_POINT_DATA_ID = 'cycle_count'
export const TOTAL_CHARGE_CAPACITY_POINT_DATA_ID = 'tt_total_charge_capacity'

export function normalizeBatteryPointId(value) {
  return String(value ?? '').trim().toLowerCase()
}

// 识别数据点编码及其通道地址前缀，保留真实标签 ID。
export function matchesBatteryPointId(value, baseId) {
  const id = normalizeBatteryPointId(value)
  const base = normalizeBatteryPointId(baseId)
  return Boolean(base) && (id === base || id.endsWith(`_${base}`))
}
