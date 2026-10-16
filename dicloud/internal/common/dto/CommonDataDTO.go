package dto

import (
	"dicloud/internal/business/model"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/util/pointid"
)

type CommonDataDTO struct {
	Equipment         model.Equipment
	Gateway           tolink_grpc.Gateway
	GatewaySTR        tolink_grpc.Gateway
	StatisticDataList []StatisticData
}

// 统计池数据 - 统计器
type StatisticData struct {
	GatewayID  string `json:"gatewayID"`  // 网关id
	Point      string `json:"point"`      // 数据点id
	StatsName  string `json:"statsName"`  // 统计器名称
	Type       int    `jons:"type"`       // 统计类型 2: 日统计
	StartTime  string `json:"startTime"`  // 开始统计时间
	DeviceType string `json:"deviceType"` // 设备类型
	Count      string `json:"count"`      // 公式
	Notes      string `json:"notes"`      // 备注
}

// DP - data point
var (
	// 铁塔数据名称转换
	DP_NormalDataTransFormat = map[string]string{
		"电池设备运维ID":  "电池ID",
		"电池充放电状态":   "电池状态",
		"电池组总电压":    "组电压",
		"电流":        "组电流",
		"剩余容量":      "电池组SOC",
		"电池放电次数":    "放电次数",
		"电池单体电压1#":  "电压1",
		"电池单体电压2#":  "电压2",
		"电池单体电压3#":  "电压3",
		"电池单体电压4#":  "电压4",
		"电池单体电压5#":  "电压5",
		"电池单体电压6#":  "电压6",
		"电池单体电压7#":  "电压7",
		"电池单体电压8#":  "电压8",
		"电池单体电压9#":  "电压9",
		"电池单体电压10#": "电压10",
		"电池单体电压11#": "电压11",
		"电池单体电压12#": "电压12",
		"电池单体电压13#": "电压13",
		"电池单体电压14#": "电压14",
		"电池单体电压15#": "电压15",
		"电池单体电压16#": "电压16",
		"电池温度1#":    "温度1",
		"电池温度2#":    "温度2",
		"电池温度3#":    "温度3",
		"电池温度4#":    "温度4",
	}

	// 算法计算数据, +部分内部固定数据, +铁塔未确定数据
	DP_DefaultNormalDataMap = map[string]string{
		"alg_soc":                       "SOC",
		"alg_soh":                       "SOH",
		pointid.RemainingDischargeTime:  "放电时长",
		pointid.ThermalRunawayRiskLevel: "热失控风险等级",
		pointid.TotalDischargeCapacity:  "累计放电量",
		pointid.CycleCount:              "循环次数",
		pointid.TotalChargeCapacity:     "累计充电量",
		"tt_envtemp":                    "环境温度",
		"tt_mostemp":                    "MOS温度",
		"ditoo_ecores":                  "单体电池支数",
		pointid.VoltageDifference:       "压差",
		"ditoo_tempnum":                 "温度采集数",
		pointid.TemperatureDifference:   "温差",
		"alg_cell01_soc":                "1号单体SOC",
		"alg_cell02_soc":                "2号单体SOC",
		"alg_cell03_soc":                "3号单体SOC",
		"alg_cell04_soc":                "4号单体SOC",
		"alg_cell05_soc":                "5号单体SOC",
		"alg_cell06_soc":                "6号单体SOC",
		"alg_cell07_soc":                "7号单体SOC",
		"alg_cell08_soc":                "8号单体SOC",
		"alg_cell09_soc":                "9号单体SOC",
		"alg_cell10_soc":                "10号单体SOC",
		"alg_cell11_soc":                "11号单体SOC",
		"alg_cell12_soc":                "12号单体SOC",
		"alg_cell13_soc":                "13号单体SOC",
		"alg_cell14_soc":                "14号单体SOC",
		"alg_cell15_soc":                "15号单体SOC",
		"alg_cell16_soc":                "16号单体SOC",
	}

	// 字符串数据点
	DP_STRDataMap = map[string]string{
		pointid.ThermalRunawayRiskLevel: "热失控风险等级",
	}

	// 默认健康度数据点ID - 算法平台返回数据
	DP_DEFAULT_SOH_DATAID = "alg_soh"
)
