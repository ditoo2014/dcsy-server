package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	alarmmodel "alarm/internal/alarm/model"
	commonmodel "alarm/internal/common/model"

	"github.com/gin-gonic/gin"
)

func TestDeliveryAttemptUpdatesRetriesThenDies(t *testing.T) {
	now := time.Unix(1_720_000_000, 0)
	delays := []time.Duration{time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute}
	for attemptBefore := 0; attemptBefore < 5; attemptBefore++ {
		item := alarmmodel.AlarmNotificationDelivery{AttemptCount: attemptBefore, MaxAttempts: 5}
		values, attempt := deliveryAttemptUpdates(item, deliveryAttemptResult{
			Status: alarmmodel.AlarmDeliveryRetryableFailed,
		}, now)
		if attempt != attemptBefore+1 {
			t.Fatalf("unexpected attempt count: got=%d want=%d", attempt, attemptBefore+1)
		}
		if attempt == 5 {
			if values["status"] != alarmmodel.AlarmDeliveryDead || values["next_retry_at"] != int64(0) {
				t.Fatalf("fifth failure must be dead: %#v", values)
			}
			continue
		}
		if values["status"] != alarmmodel.AlarmDeliveryRetryableFailed {
			t.Fatalf("attempt %d must remain retryable: %#v", attempt, values)
		}
		wantRetry := now.Add(delays[attempt-1]).Unix()
		if values["next_retry_at"] != wantRetry {
			t.Fatalf("attempt %d retry time: got=%v want=%d", attempt, values["next_retry_at"], wantRetry)
		}
	}
}

func TestClassifyDeliveryTimeoutAsUnknown(t *testing.T) {
	result := classifyDeliveryError(context.DeadlineExceeded)
	if result.Status != alarmmodel.AlarmDeliveryUnknown {
		t.Fatalf("provider timeout must be unknown, got %#v", result)
	}
}

func TestDeliveryWithNoResolvedContactsIsSkipped(t *testing.T) {
	ctx := &gin.Context{}
	data := commonmodel.AlarmMQModel{}
	if result := sendReliableSMS(ctx, data, nil); result.Status != alarmmodel.AlarmDeliverySkipped {
		t.Fatalf("SMS without contacts must be skipped, got %#v", result)
	}
}

func TestEquipmentCountDeliveryUpdatesEquipment(t *testing.T) {
	db := setupAlarmServiceTest(t)
	data := baseAlarmMessage()
	if err := db.Create(&commonmodel.AlarmDataRecord{
		EquipmentId: data.EquipmentId, AlarmCode: data.AlarmCode, BeginTime: data.Begin,
		AlarmType: data.AlarmType, AlarmStage: data.AlarmStage, AlarmStatus: 1,
	}).Error; err != nil {
		t.Fatalf("create active alarm: %v", err)
	}
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal alarm payload: %v", err)
	}
	result := executeAlarmDelivery(&gin.Context{}, alarmmodel.AlarmNotificationDelivery{
		Channel:      alarmmodel.AlarmDeliveryChannelEquipmentCount,
		RecipientIDs: "[1]", AlarmPayload: string(payload),
	})
	if result.Status != alarmmodel.AlarmDeliverySuccess {
		t.Fatalf("equipment count delivery failed: %#v", result)
	}
	var alarmTriggerNum int
	if err := db.Table("equipment").Select("alarm_trigger_num").Where("id = ?", data.EquipmentId).
		Scan(&alarmTriggerNum).Error; err != nil {
		t.Fatalf("read equipment alarm count: %v", err)
	}
	if alarmTriggerNum != 1 {
		t.Fatalf("unexpected equipment alarm count: %d", alarmTriggerNum)
	}
}
