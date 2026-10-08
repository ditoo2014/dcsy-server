package model

import "encoding/json"

type Warehouse struct {
	ID               uint            `json:"id" gorm:"primary key"`
	Name             string          `json:"name"`                                                                                                                                                                                          // 库房名称
	Address          string          `json:"address"`                                                                                                                                                                                       // 库房地址
	ManagerName      string          `json:"managerName"`                                                                                                                                                                                   // 库房管理员姓名
	ManagerUserId    int             `json:"managerUserId"`                                                                                                                                                                                 // 库房管理员ID
	ManagerPhone     string          `json:"managerPhone"`                                                                                                                                                                                  // 管理员电话
	WarehouseCode    string          `json:"warehouseCode" gorm:"index:uk_warehouse_warehouse_code,unique,expression:trim(warehouse_code),where:warehouse_code IS NOT NULL AND trim(warehouse_code) <> '' AND trim(warehouse_code) <> '-'"` // 编码
	CorpId           int             `json:"corpId"`                                                                                                                                                                                        // 部门id
	CreateUserId     int             `json:"createUserId"`                                                                                                                                                                                  // 创建用户id
	WarehouseDetails json.RawMessage `json:"warehouseDetails" gorm:"type:jsonb"`                                                                                                                                                            // 库房具体地理位置等其他信息
}

// 实现Gorm接口, 指定表名warehouse
func (Warehouse) TableName() string {
	return "warehouse"
}
