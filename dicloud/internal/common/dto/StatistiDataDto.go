package dto

import "dicloud/util/pointid"

var (
	// 默认统计池 数据点
	DP_STATISTIC_DATA_LIST = []string{pointid.TotalDischargeCapacity, pointid.CycleCount, pointid.TotalChargeCapacity}

	// 默认统计池 数据点
	DP_DefaultStatisticDataMap = map[string]string{
		pointid.TotalDischargeCapacity: "累计放电量",
		pointid.CycleCount:             "循环次数",
		pointid.TotalChargeCapacity:    "累计充电量",
	}

	// 统计数据点 - 对应代码
	DP_STATISTIC_CODE_MAP = map[string]string{
		// 英文数据点 ID 对应既有中文统计代码，汇总记录中的统计代码保持不变。
		pointid.TotalDischargeCapacity: "累计放电量",
		pointid.CycleCount:             "循环次数",
		pointid.TotalChargeCapacity:    "累计充电量",
	}
)
