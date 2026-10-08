package model

import "encoding/json"

type DataDetails struct {
	ID                 uint            `json:"id" gorm:"primaryKey,column:id"`
	EquipmentId        int             `json:"equipmentId"`                          // 作为系统的唯一值, 绑定网关的各种信息
	Name               string          `json:"name"`                                 // 业务云数据点名称
	LabelId            string          `json:"labelId"`                              // 数据池数据点id
	LabelName          string          `json:"labelName"`                            // 数据池数据点名称
	ContentDataDetails json.RawMessage `json:"contentDataDetails" gorm:"type:jsonb"` // 数据详情 列表; 如: 符号-$, 单位-美元
	SortNum            int             `json:"sortNum"`                              // 排序序号
}

// TableName 指定表名为 data_detail
func (DataDetails) TableName() string {
	return "data_details"
}
