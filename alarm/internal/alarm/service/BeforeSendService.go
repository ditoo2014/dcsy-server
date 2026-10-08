package service

import (
	"encoding/json"

	"alarm/internal/common/model"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

type AlarmHandleResult int

const (
	AlarmHandleFailed AlarmHandleResult = iota
	AlarmHandleCreated
	AlarmHandleRecovered
	AlarmHandleContinued
	AlarmHandleDuplicate
)

// MatchAndSaveAlarmData remains for internal compatibility. New consumers should call AcceptAlarmEvent.
func MatchAndSaveAlarmData(ctx *gin.Context, message *primitive.MessageExt) (data model.AlarmMQModel, result AlarmHandleResult) {
	if message == nil || json.Unmarshal(message.Body, &data) != nil {
		return data, AlarmHandleFailed
	}
	accepted, err := AcceptAlarmEvent(ctx, message)
	if err != nil {
		return data, AlarmHandleFailed
	}
	if !accepted {
		return data, AlarmHandleDuplicate
	}
	if data.End == 0 {
		return data, AlarmHandleCreated
	}
	if data.End == 1 {
		return data, AlarmHandleContinued
	}
	return data, AlarmHandleRecovered
}
