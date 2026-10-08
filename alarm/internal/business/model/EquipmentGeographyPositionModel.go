package model

import (
	"encoding/json"
)

type EquipmentGeographyPosition struct {
	ID              uint            `json:"id" gorm:"primary key"`
	EquipmentId     int             `json:"equipmentId"`                                    // 设备id
	Longitude       string          `json:"longitude"`                                      // 经度
	Latitude        string          `json:"latitude"`                                       // 纬度
	PositionDetails json.RawMessage `json:"positionDetails" gorm:"type:jsonb"`              // 位置详情
	CreateUserId    int             `json:"createUserId"`                                   // 创建用户id
	CreateTime      int64           `json:"createTime" gorm:"type:json;column:create_time"` // 记录创建时间
}

// TableName 指定表名为 equipment_geography_position
func (EquipmentGeographyPosition) TableName() string {
	return "public.equipment_geography_position"
}
