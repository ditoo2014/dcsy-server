package main

import (
	"context"
	"testing"

	"alarm/util/opengauss"

	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

func TestOnMessageAcceptsEmptyBatch(t *testing.T) {
	result, err := onMessage(context.Background())
	if err != nil || result != consumer.ConsumeSuccess {
		t.Fatalf("empty batch expected success, result=%v err=%v", result, err)
	}
}

func TestOnMessageRetriesPersistenceFailureWithoutRedis(t *testing.T) {
	previous := opengauss.OpengaussGormDB
	opengauss.OpengaussGormDB = nil
	t.Cleanup(func() { opengauss.OpengaussGormDB = previous })
	message := &primitive.MessageExt{
		Message: primitive.Message{Body: []byte(`{"equipmentId":1,"alarmCode":"A"}`)},
		MsgId:   "message-id",
	}
	result, err := onMessage(context.Background(), message)
	if err != nil || result != consumer.ConsumeRetryLater {
		t.Fatalf("persistence failure expected retry, result=%v err=%v", result, err)
	}
}
