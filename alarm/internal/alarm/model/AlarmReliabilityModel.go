package model

import "time"

const (
	AlarmAuditUnset = "-"

	AlarmInboxAccepted = "accepted"
	AlarmInboxConflict = "conflict"

	AlarmDeliveryPending         = "pending"
	AlarmDeliverySending         = "sending"
	AlarmDeliverySuccess         = "success"
	AlarmDeliveryRetryableFailed = "retryable_failed"
	AlarmDeliveryUnknown         = "unknown"
	AlarmDeliveryDead            = "dead"
	AlarmDeliverySkipped         = "skipped"

	AlarmDeliveryChannelEquipmentCount = "equipment_alarm_count"
)

type AlarmEventInbox struct {
	ID            uint      `json:"id" gorm:"primaryKey,column:id"`
	EventID       string    `json:"eventId" gorm:"column:event_id;size:128;uniqueIndex:uni_alarm_event_inbox_event"`
	LifecycleKey  string    `json:"lifecycleKey" gorm:"column:lifecycle_key;size:64"`
	PayloadHash   string    `json:"payloadHash" gorm:"column:payload_hash;size:64"`
	Payload       string    `json:"-" gorm:"column:payload;type:text"`
	Status        string    `json:"status" gorm:"column:status;size:24"`
	AlarmRecordID uint      `json:"alarmRecordId" gorm:"column:alarm_record_id"`
	ConflictHash  string    `json:"conflictHash" gorm:"column:conflict_hash;size:64"`
	ProcessedAt   int64     `json:"processedAt" gorm:"column:processed_at"`
	CreateTime    time.Time `json:"createTime" gorm:"column:create_time"`
	UpdateTime    time.Time `json:"updateTime" gorm:"column:update_time"`
}

func (AlarmEventInbox) TableName() string {
	return "alarm_event_inbox"
}

type AlarmNotificationDelivery struct {
	ID                uint      `json:"id" gorm:"primaryKey,column:id;index:idx_alarm_delivery_lifecycle,priority:2"`
	EventID           string    `json:"eventId" gorm:"column:event_id;size:128;uniqueIndex:uni_alarm_delivery_target,priority:1"`
	LifecycleKey      string    `json:"lifecycleKey" gorm:"column:lifecycle_key;size:64;index:idx_alarm_delivery_lifecycle,priority:1"`
	AlarmRecordID     uint      `json:"alarmRecordId" gorm:"column:alarm_record_id"`
	Channel           string    `json:"channel" gorm:"column:channel;size:32;uniqueIndex:uni_alarm_delivery_target,priority:2"`
	RecipientKey      string    `json:"recipientKey" gorm:"column:recipient_key;size:80;uniqueIndex:uni_alarm_delivery_target,priority:3"`
	RecipientIDs      string    `json:"-" gorm:"column:recipient_ids;type:text"`
	AlarmPayload      string    `json:"-" gorm:"column:alarm_payload;type:text"`
	Status            string    `json:"status" gorm:"column:status;size:32;index:idx_alarm_delivery_due,priority:1;index:idx_alarm_delivery_lifecycle,priority:3"`
	AttemptCount      int       `json:"attemptCount" gorm:"column:attempt_count"`
	MaxAttempts       int       `json:"maxAttempts" gorm:"column:max_attempts"`
	NextRetryAt       int64     `json:"nextRetryAt" gorm:"column:next_retry_at;index:idx_alarm_delivery_due,priority:2"`
	LeaseOwner        string    `json:"leaseOwner" gorm:"column:lease_owner;size:128"`
	LeaseUntil        int64     `json:"leaseUntil" gorm:"column:lease_until;index:idx_alarm_delivery_due,priority:3"`
	ProviderRequestID string    `json:"providerRequestId" gorm:"column:provider_request_id;size:128"`
	ResponseCode      string    `json:"responseCode" gorm:"column:response_code;size:128"`
	Response          string    `json:"response" gorm:"column:response;type:text"`
	LastError         string    `json:"lastError" gorm:"column:last_error;type:text"`
	SentAt            int64     `json:"sentAt" gorm:"column:sent_at"`
	CreateTime        time.Time `json:"createTime" gorm:"column:create_time"`
	UpdateTime        time.Time `json:"updateTime" gorm:"column:update_time"`
}

func (AlarmNotificationDelivery) TableName() string {
	return "alarm_notification_delivery"
}
