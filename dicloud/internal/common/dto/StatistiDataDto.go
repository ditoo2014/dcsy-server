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
		// 数据点编码和中文统计代码均沿用原版，继续使用已有汇总数据。
		pointid.TotalDischargeCapacity: "累计放电量",
		pointid.CycleCount:             "循环次数",
		pointid.TotalChargeCapacity:    "累计充电量",
	}
)
