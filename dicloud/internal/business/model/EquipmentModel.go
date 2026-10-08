package model

import (
	"encoding/json"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
)

type Equipment struct {
	ID                   uint                         `json:"id" gorm:"primary key"`
	EquipmentCode        string                       `json:"equipmentCode" gorm:"index:uk_equipment_equipment_code,unique,expression:trim(equipment_code),where:equipment_code IS NOT NULL AND trim(equipment_code) <> '' AND trim(equipment_code) <> '-'"` // 设备编码
	EquipmentName        string                       `json:"equipmentName"`                                                                                                                                                                                 // 设备名称
	EquipmentTypeId      int                          `json:"equipmentTypeId"`                                                                                                                                                                               // 设备类型id
	EquipmentTypeName    string                       `json:"equipmentTypeName"`                                                                                                                                                                             // 设备类型名称
	SiteId               int                          `json:"belongToSiteId"`                                                                                                                                                                                // 站点id
	SiteName             string                       `json:"belongToSiteName"`                                                                                                                                                                              // 站点名称
	WarehouseId          int                          `json:"warehouseId"`                                                                                                                                                                                   // 库房id
	WarehouseName        string                       `json:"warehouseName"`                                                                                                                                                                                 // 库房名称
	Manufacturer         string                       `json:"manufacturer"`                                                                                                                                                                                  // 生产厂家
	ManufacturerTime     string                       `json:"manufacturerTime"`                                                                                                                                                                              // 生产日期
	Batch                string                       `json:"batch"`                                                                                                                                                                                         // 批次
	Specifications       string                       `json:"specifications"`                                                                                                                                                                                // 规格
	BmsManufacturer      string                       `json:"bmsManufacturer"`                                                                                                                                                                               // BMS生产厂家
	BmsVersion           string                       `json:"bmsVersion"`                                                                                                                                                                                    // BMS版本
	ApiKey               string                       `json:"apiKey"`
	ApiKeys              json.RawMessage              `json:"apiKeys" gorm:"type:jsonb"`
	EquipmentDetails     json.RawMessage              `json:"equipmentDetails" gorm:"type:jsonb"`                                                                       // 其他详细信息
	DataDetails          []DataDetails                `json:"dataDetails" gorm:"->;foreignKey:EquipmentId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 数据详情列表
	VideoProtoType       string                       `json:"videoProtoType"`                                                                                           // 视频接入方式
	VideoUrls            pq.StringArray               `json:"videoUrls" gorm:"type:text[]"`                                                                             // 视频监控地址数据, 一般保存无需处理的视频url
	EzvizCloud           json.RawMessage              `json:"ezvizCloud" gorm:"type:jsonb"`
	RunningStatus        int                          `json:"runningStatus"`                                                                                                   // 运行状态
	PositionStatus       int                          `json:"positionStatus"`                                                                                                  // 出入库/上下站/入退网等 状态
	AlarmTriggerNum      int                          `json:"alarmTriggerNum"`                                                                                                 // 报警触发个数
	EngageInternetStatus int                          `json:"engageInternetStatus"`                                                                                            // 并网状态
	CorpId               int                          `json:"corpId"`                                                                                                          // 部门id
	CreateUserId         int                          `json:"createUserId"`                                                                                                    // 创建用户id
	ConcernUserIds       pq.Int64Array                `json:"concernUserIds" gorm:"type:int[]"`                                                                                // 关注用户
	ControlUserIds       pq.Int64Array                `json:"controlUserIds" gorm:"type:int[]"`                                                                                // 控制用户
	CirculationLogs      []EquipmentCirculationLog    `json:"circulationLogs" gorm:"->;foreignKey:EquipmentId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`    // 设备流转日志 ->只读
	LatestCirculationLog *EquipmentCirculationLog     `json:"latestCirculationLog,omitempty" gorm:"-:all"`                                                                     // 最新设备流转日志 ->只读
	GeographyPositions   []EquipmentGeographyPosition `json:"geographyPositions" gorm:"->;foreignKey:EquipmentId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 设备流转日志 ->只读
	CorpList             []int                        `json:"corpList" gorm:"-:all"`                                                                                           // 同级部门(默认) - 只用作查询使用
}

// TableName 指定表名为 equipment
func (Equipment) TableName() string {
	return "equipment"
}

// Redis 序列化
// MarshalBinary 实现 encoding.BinaryMarshaler 接口
func (u Equipment) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}
