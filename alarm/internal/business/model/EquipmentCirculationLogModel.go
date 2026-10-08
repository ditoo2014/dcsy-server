package model

import (
	"encoding/json"
)

type EquipmentCirculationLog struct {
	ID                uint            `json:"id" gorm:"primary key"`
	EquipmentId       int             `json:"equipmentId"`                                    // 设备id
	EquipmentTypeName string          `json:"equipmentTypeName" gorm:"-:all"`                 // 电池类型名称（非数据库字段，查询时填充）
	OperatorUserId    int             `json:"operatorUserId"`                                 // 操作用户id
	OperateType       int             `json:"operateType"`                                    // 操作类型 1: 入库, 2: 出库, 3: 上站入网, 其他...
	OperateContent    string          `json:"operateContent"`                                 // 操作内容
	OperateDetails    json.RawMessage `json:"operateDetails" gorm:"type:jsonb"`               // 记录操作细节
	CreateTime        int64           `json:"createTime" gorm:"type:json;column:create_time"` // 记录创建时间
}

// TableName 指定表名为 equipment_circulation_log
func (EquipmentCirculationLog) TableName() string {
	return "public.equipment_circulation_log"
}
