package model

import "time"

// TtdcAlarmRule is a platform-level public alarm definition. The template
// group/template point fields retain the selected point as a management-time
// reference; runtime matching uses that point's data-pool label name across
// every active TTDC battery channel.
type TtdcAlarmRule struct {
	ID              uint                 `json:"id" gorm:"primaryKey;column:id"`
	TemplateGroupID uint                 `json:"templateGroupId" gorm:"column:template_group_id;not null;index"`
	TemplateID      uint                 `json:"templateId" gorm:"column:template_id;not null;index"`
	TemplatePointID uint                 `json:"templatePointId" gorm:"column:template_point_id;not null;index"`
	BaseDataID      string               `json:"baseDataId" gorm:"column:base_data_id;size:48;not null;index"`
	RuleName        string               `json:"ruleName" gorm:"column:rule_name;size:128;not null"`
	RuleCode        string               `json:"ruleCode" gorm:"column:rule_code;size:128;not null"`
	AlarmType       string               `json:"alarmType" gorm:"column:alarm_type;size:8;not null"`
	AlarmStage      int                  `json:"alarmStage" gorm:"column:alarm_stage;not null;default:1"`
	AlarmChannel    string               `json:"alarmChannel" gorm:"column:alarm_channel;size:64"`
	AlarmUsers      string               `json:"alarmUsers" gorm:"column:alarm_users;size:512"`
	TriggerModel    int                  `json:"triggerModel" gorm:"column:trigger_model;not null;default:0"`
	Formula         string               `json:"formula" gorm:"column:formula;size:256;not null;default:'x'"`
	Cycle           int64                `json:"cycle" gorm:"column:cycle;not null;default:0"`
	SustainTime     int64                `json:"sustainTime" gorm:"column:sustain_time;not null;default:0"`
	Enabled         bool                 `json:"enabled" gorm:"column:enabled;not null;default:true"`
	Version         int                  `json:"version" gorm:"column:version;not null;default:1"`
	SortNo          int                  `json:"sortNo" gorm:"column:sort_no;not null;default:0"`
	Description     string               `json:"description" gorm:"column:description;size:512"`
	CreatedBy       int                  `json:"createdBy" gorm:"column:created_by;not null;default:0"`
	UpdatedBy       int                  `json:"updatedBy" gorm:"column:updated_by;not null;default:0"`
	CreatedAt       time.Time            `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt       time.Time            `json:"updatedAt" gorm:"column:updated_at"`
	Logics          []TtdcAlarmRuleLogic `json:"logics" gorm:"foreignKey:RuleID;references:ID;constraint:false"`
	TemplateName    string               `json:"templateName" gorm:"-"`
	PointName       string               `json:"pointName" gorm:"-"`
}

func (TtdcAlarmRule) TableName() string { return "ttdc_alarm_rule" }

// TtdcAlarmRuleLogic stores the numeric comparison used by a public TTDC rule.
// MatchValue remains in the table for backward-compatible reads only.
type TtdcAlarmRuleLogic struct {
	ID           uint    `json:"id" gorm:"primaryKey;column:id"`
	RuleID       uint    `json:"ruleId" gorm:"column:rule_id;not null;index"`
	Logic        string  `json:"logic" gorm:"column:logic;size:8;not null"`
	TriggerValue float64 `json:"triggerValue" gorm:"column:trigger_value;not null;default:0"`
	MatchValue   string  `json:"matchValue" gorm:"column:match_value;size:256"`
	SortNo       int     `json:"sortNo" gorm:"column:sort_no;not null;default:0"`
}

func (TtdcAlarmRuleLogic) TableName() string { return "ttdc_alarm_rule_logic" }
