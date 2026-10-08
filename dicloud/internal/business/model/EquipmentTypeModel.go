package model

// EquipmentType 保存前端设备类型维护页面使用的类型字典。
// 设备表仍保留类型名称快照，便于历史数据查询和导出。
type EquipmentType struct {
	ID                    uint   `json:"id" gorm:"primaryKey,column:id"`
	EquipmentTypeName     string `json:"equipmentTypeName" gorm:"column:equipment_type_name;size:128;not null;uniqueIndex:uk_equipment_type_name"`
	EquipmentTypeDescribe string `json:"equipmentTypeDescribe" gorm:"column:equipment_type_describe;size:512"`
}

func (EquipmentType) TableName() string {
	return "equipment_type"
}
