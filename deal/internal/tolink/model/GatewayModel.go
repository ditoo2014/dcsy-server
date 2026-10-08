package model

import (
	"encoding/json"
)

// TTDC 是采集平台唯一支持的网关机型。
const TTDC = "TTDC"

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
	GatewayId      string        `json:"gatewayId"`                                                                                                      // 作为系统的唯一值, 绑定网关的各种信息
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
