package model

import (
	"encoding/json"
	"errors"
)

type AlarmMQModel struct {
	SiteId         int         `json:"siteId"`         // 站点ID
	EquipmentId    int         `json:"equipmentId"`    // 站点ID
	AlarmCode      string      `json:"alarmCode"`      // 报警代码
	AlarmType      string      `json:"alarmType"`      // 报警类别 0: 普通 1: 预警, 多种方式用','隔开
	AlarmStage     int         `json:"alarmStage"`     // 报警等级
	AlarmChannel   string      `json:"alarmChannel"`   // 报警渠道，当前仅支持 0（短信）
	AlarmUsers     string      `json:"alarmUsers"`     // 报警用户 存储用户ID, 多个用户用','隔开
	Begin          int64       `json:"begin"`          // 报警触发时间
	End            int64       `json:"end"`            // 报警恢复时间
	AlarmValue     interface{} `json:"alarmValue"`     // 实时值
	CustomerNumber string      `json:"customerNumber"` // 客户编号，用于区分不同公司的短信账号
	SiteName       string      `json:"siteName"`       // 站点名称
	EquipmentName  string      `json:"equipmentName"`  // 站点名称 + 其他信息
	DbName         string      `json:"dbName"`         // 数据库(多租户)
	CorpId         int         `json:"corpId"`         // 部门ID - 用于发送报警, 保存记录
}

func (p *AlarmMQModel) ToJsonStr() (jsonStr string, err error) {

	packJson, err := json.Marshal(p)
	if nil != err {
		err = errors.New("系统错误")
	}

	jsonStr = string(packJson)

	return
}

type StatisticMQModel struct {
	SiteId             int     `json:"siteId"`             // 站点ID
	EquipmentId        int     `json:"equipmentId"`        // 站点ID
	CorpId             int     `json:"corpId"`             // 部门id
	StatisticCode      string  `json:"statisticCode"`      // 触发代码
	StatisticName      string  `json:"statisticName"`      // 统计名称 - 可以命名别名
	StatisticAlarmType string  `json:"statisticAlarmType"` // 报警类别
	Begin              int64   `json:"begin"`              // 触发时间
	End                int64   `json:"end"`                // 恢复时间
	DataSourceType     int     `json:"dataSourceType"`     // 数据来源 0: mqtt数据 1: 统计池(蒙毅) 2: 业务云设备统计数据
	StatisticType      string  `json:"statisticType"`      // 统计方式 0: 普通方式, 1: 计算报警条数
	StatisticValue     float64 `json:"statisticValue"`     // 设置值
	StatisticValueType int     `json:"statisticValueType"` // 数据类型 0: 累计值 1: 增量值

	TriggerValue interface{} `json:"triggerValue"` // 实时值
	DbName       string      `json:"dbName"`       // 数据库(多租户)
}

func (p *StatisticMQModel) ToJsonStr() (jsonStr string, err error) {

	packJson, err := json.Marshal(p)
	if nil != err {
		err = errors.New("系统错误")
	}

	jsonStr = string(packJson)

	return
}
