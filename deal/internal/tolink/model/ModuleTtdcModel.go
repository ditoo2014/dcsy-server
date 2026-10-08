package model

import (
	"encoding/json"
	"regexp"
	"strings"
)

const TtdcADRPattern = `^[0-9A-F]{2}$`

var ttdcADRPattern = regexp.MustCompile(TtdcADRPattern)

// NormalizeTtdcADR validates and normalizes one protocol-byte ADR.
func NormalizeTtdcADR(value string) (string, bool) {
	value = strings.ToUpper(strings.TrimSpace(value))
	return value, ttdcADRPattern.MatchString(value)
}

/* Redis 序列化存储 */
func (g GatewayModuleTtdc) MarshalBinary() ([]byte, error) {
	return json.Marshal(g)
}

/* TTDC MODULE drive */
type GatewayModuleTtdc struct {
	ID              uint             `json:"id" gorm:"primaryKey,column:id"`
	GatewayId       string           `json:"gatewayId" gorm:"column:gateway_id;size:64;index"`
	Name            string           `json:"name" gorm:"size:128"`
	Title           string           `json:"title" gorm:"size:128;default:'nil'"`
	Type            int              `json:"type"`                                                                                                 // (对下)通道分类, 采集协议版本
	Transport       string           `json:"transport"`                                                                                            // 采集周期 IP 端口 从机号 16位整数组码顺序 32位整数组码顺序 32位浮点数组码顺序
	TtdcConfig      ModuleTtdcConfig `json:"ttdcConfig" gorm:"->;foreignKey:ModuleId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 配置信息
	ModuleDatas     []ModuleTtdcData `json:"list" gorm:"->;foreignKey:ModuleId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	TemplateID      uint             `json:"templateId,omitempty" gorm:"-"`
	TemplateVersion int              `json:"templateVersion,omitempty" gorm:"-"`
	ProtocolCode    string           `json:"protocolCode,omitempty" gorm:"-"`
	ParserProfile   string           `json:"parserProfile,omitempty" gorm:"-"`
}

/* 如果有其他类型的配置(其他版本协议), 可以新建一种, 不必在原配置上更改 */
type ModuleTtdcConfig struct {
	ID           uint   `json:"id" gorm:"primaryKey,column:id"`
	ModuleId     int    `json:"moduleId"`     // Module 的数据库id
	ModuleSoi    string `json:"moduleSoi"`    // 起始位标志（START OF INFORMATION）
	ModuleVer    string `json:"moduleVer"`    // 通信协议版本号（2.0）
	ModuleAdr    string `json:"moduleAdr"`    // 对同类型设备的不同地址描述（两位十六进制，00－FF）
	ModuleCid1   string `json:"moduleCid1"`   // 设备标识码(设备类型描述)
	ModuleCid2   string `json:"moduleCid2"`   // 命令信息：控制标识码（数据或动作类型描述） // 响应信息：返回码RTN
	ModuleLength string `json:"moduleLength"` // INFO字节长度（包括LENID和LCHKSUM）
	ModuleInfo   string `json:"moduleInfo"`   // 命令信息：控制数据信息COMMAND_INFO // 应答信息：应答数据信息DATA_INFO
	ModuleChksum string `json:"moduleChksum"` // 校验和码
	ModuleEoi    string `json:"moduleEoi"`    // 结束码（END OF INFORMATION）
	// 其他信息 - 高低位, 数据类型
	// OrderFloat32 int `json:"orderFloat32"` // 32位浮点数组码顺序
	// OrderInt8    int `json:"orderInt8"`    // 8位整数组码顺序
	// OrderInt16   int `json:"orderInt16"`   // 16位整数组码顺序
	// OrderInt32   int `json:"orderInt32"`   // 32位整数组码顺序
}

var (
	TTDC_CID1_4BH = "4BH" // 锂硫蓄电池BMS

	TTDC_CID2_42H = "42H" // 获取遥测量信息
	TTDC_CID2_44H = "44H" // 获取遥信量信息
	TTDC_CID2_45H = "45H" // 遥控命令
	TTDC_CID2_47H = "47H" // 获取遥调量信息
	TTDC_CID2_49H = "49H" // 设定遥调量信息
	TTDC_CID2_4FH = "4FH" // 获取通信协议版本号
	TTDC_CID2_51H = "51H" // 获取设备厂商信息
	TTDC_CID2_4BH = "4BH" // 获取历史数据
	TTDC_CID2_4DH = "4DH" // 获取时间
	TTDC_CID2_4EH = "4EH" // 同步时间
)

type ModuleTtdcData struct {
	ID        uint   `json:"id" gorm:"primaryKey,column:id"`
	ModuleId  int    `json:"moduleId" gorm:"column:module_id"`
	Name      string `json:"name" gorm:"size:128"`          // 数据点名称
	DataId    string `json:"dataId" gorm:"size:64;"`        // 数据点ID
	DataType  int    `json:"dataType" gorm:"type:smallint"` // 1: float, 2: 字符串
	Transport string `json:"transport"`                     // 遥测量/遥信量 字节序 字节序号(开始节点) 字节数 是否取位 取位
	Formula   string `json:"formula"`                       // 处理公式
	Unformula string `json:"unformula"`                     // 逆处理公式
	// AlgorithmKey is populated only for actual 42_SOC channels.
	AlgorithmKey *string `json:"algorithmKey,omitempty" gorm:"column:algorithm_key;size:64"`
}

// 表名自定义（可选）
func (GatewayModuleTtdc) TableName() string {
	return "gateway_module_ttdc"
}

// 模块通道配置数据
func (ModuleTtdcConfig) TableName() string {
	return "gateway_module_ttdc_config"
}

// 模块通道数据点数据
func (ModuleTtdcData) TableName() string {
	return "gateway_module_ttdc_data"
}
