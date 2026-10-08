package dto

var (
	// 默认统计池 数据点
	DP_STATISTIC_DATA_LIST = []string{"tt_fangdiandianliang", "tt_chongdiancishu", "tt_chongdiandianliang"}

	// 默认统计池 数据点
	DP_DefaultStatisticDataMap = map[string]string{
		"tt_fangdiandianliang":  "累计放电量",
		"tt_chongdiancishu":     "循环次数", // 20251124.将充电次数 - 改为循环次数
		"tt_chongdiandianliang": "累计充电量",
	}

	// 统计数据点 - 对应代码
	DP_STATISTIC_CODE_MAP = map[string]string{
		"tt_fangdiandianliang":  "累计放电量",
		"tt_chongdiancishu":     "循环次数",
		"tt_chongdiandianliang": "累计充电量",
	}
)
