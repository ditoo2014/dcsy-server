package model

import pq "gitee.com/opengauss/openGauss-connector-go-pq"

type EquipmentHealthStandard struct {
	ID             uint            `json:"id" gorm:"primary key"`
	StandardName   string          `json:"standardName"`                       //
	LevelStandards pq.Float64Array `json:"levelStandards" gorm:"type:float[]"` // 带顺序的级别判断 - from 高 to 低{优, 良, 中, 差} - 如:[0.9, 0.8, 0.6, 0]
}

// TableName 指定表名为 equipment_health_standard
func (EquipmentHealthStandard) TableName() string {
	return "equipment_health_standard"
}
