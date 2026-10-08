package model

import "encoding/json"

// 统计数据存储
type StatisticData struct {
	ID                 uint     `json:"id" gorm:"primary key"`
	SiteId             int      `json:"siteId"`                         // 站点id
	EquipmentId        int      `json:"equipmentId"`                    // 设备id
	CorpId             int      `json:"corpId"`                         // 部门id
	EquipmentTypeName  string   `json:"equipmentTypeName" gorm:"-:all"` // 电池类型名称（非数据库字段，查询时填充）
	DataSourceType     int      `json:"dataSourceType"`                 // 数据来源 0: mqtt数据 1: 统计池(蒙毅)
	TargetType         int      `json:"targetType" gorm:"-:all"`        // 目标类型 1: 站点 2: 部门 - 只用作表名鉴定
	CodeList           []string `json:"codeList" gorm:"-:all"`          // 统计代码 - 只用作查询使用
	CorpList           []int    `json:"corpList" gorm:"-:all"`          // 同级部门(默认) - 只用作查询使用
	StatisticCode      string   `json:"statisticCode"`                  // 统计编码, 统计名称, 统计记录 - 统一代码(指定唯一统计对象)
	StatisticName      string   `json:"statisticName"`                  // 统计名称 - 可以命名别名
	StatisticAlarmType string   `json:"statisticAlarmType"`             // 报警类型
	StartStatisticTime int64    `json:"startStatisticTime"`             // 开始统计时间 - 增量值类统计重要
	EndStatisticTime   int64    `json:"endStatisticTime"`               // 结束统计时间 - 累计值/增量值 都重要 - (最新统计时间)
	StartValue         float64  `json:"startValue"`                     // 开始值 - 增量值类统计重要
	EndValue           float64  `json:"endValue"`                       // 结束值 - 累计值/增量值 都重要 - (最新统计时间)
	StatisticValue     float64  `json:"statisticValue"`                 // 统计值
	StatisticValueType int      `json:"statisticValueType"`             // 数据类型 - 0:累计值 1:增量值
	StatisticDateType  string   `json:"statisticDateType"`              // 统计粒度 - 统计日期分类 monthly,daily,hourly
	MarkDate           string   `json:"markDate"`                       // 粒度标记 - 统计标记 - 对应粒度具体数据
	CreateTime         int64    `json:"createTime"`                     // 创建时间
}

// TableName 指定动态表名 - 统计不同数据
func (sts StatisticData) GetTableName() string {

	target := "equipment"   // 默认为设备数据统计 - 站点可能未绑定
	granularity := "hourly" // 默认粒度 - 小时(最小粒度)

	switch sts.TargetType {
	case 1:
		target = "site"
	case 2:
		target = "corp"
	default:
		target = "equipment"
	}

	switch sts.StatisticDateType {
	case "monthly":
		granularity = "monthly"
	case "daily":
		granularity = "daily"
	default:
		granularity = "hourly"
	}

	statisticType := "cumulative"
	switch sts.StatisticValueType {
	case 1:
		statisticType = "increment"
	default:
		statisticType = "cumulative"
	}

	return "statistic_data_" + target + "_" + granularity + "_" + statisticType
}

// Redis 序列化
// MarshalBinary 实现 encoding.BinaryMarshaler 接口
func (sd StatisticData) MarshalBinary() ([]byte, error) {
	return json.Marshal(sd)
}

// 健康度统计代码
var (
	StatisticHealthCode_YOU   = "SOH优"
	StatisticHealthCode_LIANG = "SOH良"
	StatisticHealthCode_ZHONG = "SOH中"
	StatisticHealthCode_CHA   = "SOH差"
	// 默认标准代码 - 4个(优良中差)
	StatisticHealthCodeArr = []string{StatisticHealthCode_YOU, StatisticHealthCode_LIANG, StatisticHealthCode_ZHONG, StatisticHealthCode_CHA}
	/* 电池健康度：  soh  >=0.9  优    soh >=0.8   <0.9   良       soh>=0.6   <0.8  中       soh <0.6  差  ，做成可配  */
	StatisticHealthMap = map[string]int{StatisticHealthCode_YOU: 1, StatisticHealthCode_LIANG: 1, StatisticHealthCode_ZHONG: 1, StatisticHealthCode_CHA: 1}
)
