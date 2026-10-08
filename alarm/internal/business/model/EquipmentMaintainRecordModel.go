package model

import (
	"encoding/json"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
)

type EquipmentMaintainRecord struct {
	ID                uint            `json:"id" gorm:"primary key"`
	EquipmentId       int             `json:"equipmentId"`                                    // 设备id
	EquipmentTypeName string          `json:"equipmentTypeName" gorm:"-:all"`                 // 电池类型名称（非数据库字段，查询时填充）
	MaintainUserId    int             `json:"maintainUserId"`                                 // 维保用户id
	MaintainUserName  string          `json:"maintainUserName"`                               // 维保人员
	MaintainTitle     string          `json:"maintainTitle"`                                  // 维保记录简介
	MaintainContent   string          `json:"maintainContent"`                                // 维保内容
	MaintainPhotos    pq.StringArray  `json:"maintainPhotos" gorm:"type:text[]"`              // 维保照片
	MaintainDetails   json.RawMessage `json:"maintainDetails" gorm:"type:jsonb"`              // 记录操作细节
	CreateTime        int64           `json:"createTime" gorm:"type:json;column:create_time"` // 记录创建时间
}

// TableName 指定表名为 equipment_maintain_record
func (EquipmentMaintainRecord) TableName() string {
	return "public.equipment_maintain_record"
}
