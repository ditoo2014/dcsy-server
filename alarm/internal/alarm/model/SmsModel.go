package model

// 短信报警密钥模板信息
type SmsSecret struct {
	ID                  uint   `json:"id" gorm:"primaryKey,column:id"`
	AkId                string `json:"akId"`                // 访问密钥 (阿里云)
	AkSecret            string `json:"akSecret"`            // 访问secret (阿里云)
	SignName            string `json:"signName"`            // 客户签名 (公司注册)
	Endpoint            string `json:"endpoint"`            // 服务地址
	RegionId            string `json:"regionId"`            // 区域ID
	AlarmTemplateCode   string `json:"alarmTemplateCode"`   // 报警模板码/ID - 必须有, 后续可以没有
	AlarmSite           string `json:"alarmSite"`           // 报警站点参数
	AlarmEquipment      string `json:"alarmEquipment"`      // 报警设备参数
	AlarmCoding         string `json:"alarmCoding"`         // 报警代码参数
	AlarmValue          string `json:"alarmValue"`          // 报警值参数
	RecoverTemplateCode string `json:"recoverTemplateCode"` // 恢复模板码/ID - 必须有, 后续可以没有
	RecoverSite         string `json:"recoverSite"`         // 恢复站点参数
	RecoverEquipment    string `json:"recoverEquipment"`    // 恢复设备参数
	RecoverCoding       string `json:"recoverCoding"`       // 恢复代码
	RecoverValue        string `json:"recoverValue"`        // 恢复值参数
	CorpId              int    `json:"corpId"`              // 如果区分不同客户, 通过用户 最小分组id 查询密钥数据
}

// TableName 指定表名为 sms_secret
func (SmsSecret) TableName() string {
	return "sms_secret"
}

// 固定参数
const (
	SMS_REGIONID_DEFAULT = "cn-chengdu"
	SMS_ENDPOINT         = "cn-hangzhou"
	SMS_REGIONID         = "cn-hangzhou"
	SMS_PRODUCT          = "Dysmsapi"
	SMS_DOMAIN           = "dysmsapi.aliyuncs.com"
)

// 申请报警模板时, 必须一致
const (
	SMS_ALARM_SITE_TEMPLATE      = "siteName"
	SMS_ALARM_EQUIPMENT_TEMPLATE = "equipmentName"
	SMS_ALARM_CODING_TEMPLATE    = "coding"
	SMS_ALARM_VALUE_TEMPLATE     = "value"
)

// 申请恢复报警模板时, 必须一致
const (
	SMS_RECOVER_SITE_TEMPLATE      = "siteName"
	SMS_RECOVER_EQUIPMENT_TEMPLATE = "equipmentName"
	SMS_RECOVER_CODING_TEMPLATE    = "coding"
	SMS_RECOVER_VALUE_TEMPLATE     = "value"
)
