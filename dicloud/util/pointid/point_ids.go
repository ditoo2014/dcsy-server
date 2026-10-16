package pointid

import "strings"

// 电池算法与统计数据点统一使用英文 ID，与公共模板和业务查询保持一致。
// dicloud 与 deal 独立构建、部署，各自维护相同契约，避免引入跨项目源码依赖。
const (
	RemainingDischargeTime  = "alg_remaining_discharge_time"
	ThermalRunawayRiskLevel = "alg_thermal_runaway_risk_level"
	TemperatureDifference   = "ditoo_temperature_difference"
	VoltageDifference       = "ditoo_voltage_difference"
	TotalDischargeCapacity  = "tt_total_discharge_capacity"
	CycleCount              = "cycle_count"
	TotalChargeCapacity     = "tt_total_charge_capacity"
)

// Matches 匹配数据点编码及其通道地址前缀，保留实际数据点 ID。
func Matches(actualID, baseID string) bool {
	actualID = strings.ToLower(strings.TrimSpace(actualID))
	baseID = strings.ToLower(strings.TrimSpace(baseID))
	return baseID != "" && (actualID == baseID || strings.HasSuffix(actualID, "_"+baseID))
}
