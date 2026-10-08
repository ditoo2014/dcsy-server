package model

import "time"

// TtdcAlarmTriggerBinding records the tenant-side runtime instance generated
// from one platform TTDC alarm rule. It does not store an equipment ID: the
// generated trigger's marktriggers carry the current equipment mappings.
type TtdcAlarmTriggerBinding struct {
	ID               uint `json:"id" gorm:"primaryKey;column:id"`
	AlarmRuleID      uint `json:"alarmRuleId" gorm:"column:alarm_rule_id;not null;uniqueIndex:uk_ttdc_alarm_rule_binding,priority:1"`
	TemplateGroupID  uint `json:"templateGroupId" gorm:"column:template_group_id;not null;uniqueIndex:uk_ttdc_alarm_rule_binding,priority:2"`
	GeneralTriggerID uint `json:"generalTriggerId" gorm:"column:general_trigger_id;not null;index"`
	RuleVersion      int  `json:"ruleVersion" gorm:"column:rule_version;not null;default:1"`
	// MarkHash lets the scheduler skip a full trigger/mark rewrite when the
	// tenant's resolved equipment set has not changed.
	MarkHash   string    `json:"-" gorm:"column:mark_hash;size:64"`
	SyncStatus string    `json:"syncStatus" gorm:"column:sync_status;size:16;not null;default:'ready'"`
	SyncError  string    `json:"syncError" gorm:"column:sync_error;size:512"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt  time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (TtdcAlarmTriggerBinding) TableName() string { return "ttdc_alarm_trigger_binding" }
