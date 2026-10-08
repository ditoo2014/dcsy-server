package model

import (
	"encoding/json"
)

const TTDC = "TTDC" // 当前采集平台唯一支持的网关机型

// MODULE_UPDATE 是 TTDC 通道批量保存时使用的更新操作标识。
const MODULE_UPDATE = "update"

// Redis 序列化
// MarshalBinary 实现 encoding.BinaryMarshaler 接口
func (g Gateway) MarshalBinary() ([]byte, error) {
	return json.Marshal(g)
}

// UnmarshalBinary 实现 encoding.BinaryUnmarshaler 接口
func (g *Gateway) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, g)
}

/* 网关 */
type Gateway struct {
	ID             uint          `json:"id" gorm:"primaryKey,column:id"`
	Name           string        `json:"name"`                                                                                                           // 网关名称
	GatewayId      string        `json:"gatewayId" gorm:"column:gateway_id;size:64;not null;uniqueIndex"`                                                // 网关序列号是系统唯一值，也是网关数据点外键引用列
	Model          string        `json:"model"`                                                                                                          // 机型
	Card           string        `json:"card"`                                                                                                           // 卡号
	CreateUserId   int           `json:"createUserId"`                                                                                                   // 创建用户id
	CorpId         int           `json:"corpId"`                                                                                                         // 部门id, 留作备用
	GatewayDatas   []GatewayData `json:"gatewayDatas" gorm:"->;foreignKey:GatewayId;references:GatewayId;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 网关（全部）数据点
	GatewayModules []any         `json:"gatewayModules" gorm:"-:all"`                                                                                    // 网关（全部）设备列表
}

/* TableName 指定表名为 gateway */
func (Gateway) TableName() string {
	return "gateway"
}

/* 网关数据点数据 */
type GatewayData struct {
	ID        uint   `json:"id"`        // 唯一值id, 单个更新时使用
	GatewayId string `json:"gatewayId"` // 网关序列号 不使用数据库id, 用唯一序列号指定
	DataId    string `json:"dataId"`    // 数据点id 在网关内唯一
	DataType  int    `json:"dataType"`  // 数据值类型 1: float 2: 字符串
	Name      string `json:"name"`      // 数据点名称
	Dbconfig  int64  `json:"dbconfig"`  // 数据库配置
}

/* TableName 指定表名为 gateway_data */
func (GatewayData) TableName() string {
	return "gateway_data"
}
