package pointid

import "strings"

// 四个算法数据点使用英文 ID，与模板配置一致；业务统计数据点保留原有编码。
// dicloud 与 deal 独立构建、部署，各自维护相同契约，避免引入跨项目源码依赖。
const (
	RemainingDischargeTime  = "alg_remaining_discharge_time"
	ThermalRunawayRiskLevel = "alg_thermal_runaway_risk_level"
	TemperatureDifference   = "ditoo_temperature_difference"
	VoltageDifference       = "ditoo_voltage_difference"
	TotalDischargeCapacity  = "tt_fangdiandianliang"
	CycleCount              = "tt_chongdiancishu"
	TotalChargeCapacity     = "tt_chongdiandianliang"
)

// Matches 匹配数据点编码及其通道地址前缀，保留实际数据点 ID。
func Matches(actualID, baseID string) bool {
	actualID = strings.ToLower(strings.TrimSpace(actualID))
	baseID = strings.ToLower(strings.TrimSpace(baseID))
	return baseID != "" && (actualID == baseID || strings.HasSuffix(actualID, "_"+baseID))
}
