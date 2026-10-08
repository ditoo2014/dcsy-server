package model

import (
	"strings"

	"github.com/google/uuid"
)

type Stream struct {
	ID           uint              `json:"id" gorm:"primaryKey,column:id"`
	Admin        string            `json:"admin"`                                                                                                  // 管理员
	Name         string            `json:"name"`                                                                                                   // 数据池名称
	ApiKey       string            `json:"apiKey"`                                                                                                 // apikey
	CreateUserId int               `json:"createUserId"`                                                                                           // 创建用户id
	CorpId       int               `json:"corpId"`                                                                                                 // 部门id, 留作备用
	StreamCode   string            `json:"streamCode"`                                                                                             //刘郑自用
	List         []StreamLabelUnit `json:"streamLables" gorm:"->;foreignKey:StreamId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 数据池（全部）数据点
}

// TableName 指定表名为 stream
func (Stream) TableName() string {
	return "stream"
}

type StreamLabelUnit struct {
	// ID          uint   `json:"id" gorm:"primaryKey,column:id"`
	StreamId    uint   `json:"streamId"`    //数据池的id
	LabelId     string `json:"labelId"`     //标签id
	Name        string `json:"name"`        //标签名称
	GatewayName string `json:"gatewayName"` //网关名称
	GatewayId   string `json:"gatewayId"`   //网关id
	DataName    string `json:"dataName"`    //数据名称
	DataType    string `json:"dataType"`    //数据类型
	DataId      string `json:"dataId"`      //数据点id
	Formula     string `json:"formula"`     //处理公式
	Unformula   string `json:"unformula"`   //逆处理公式
}

type StreamPageSearch struct {
	Page   int    `json:"page"`   //页码
	Size   int    `json:"size"`   //页大小
	Search string `json:"search"` //关键字搜索功能
}

// TableName 指定表名为 stream_label_unit
func (StreamLabelUnit) TableName() string {
	return "stream_label_unit"
}

// GetStreamApiKey 获取ApiKey
func (s Stream) GetStreamApiKey() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
