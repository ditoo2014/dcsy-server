package service

import (
	commonmodel "alarm/internal/common/model"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
)

func TestAliSendSmsWithEmptyUsersDoesNotPanic(t *testing.T) {
	AliSendSms(&gin.Context{}, nil)
	AliSendSms(&gin.Context{}, &commonmodel.AlarmMQModel{})
}

func TestBuildAlarmSmsRequests(t *testing.T) {
	tests := []struct {
		name          string
		phoneCount    int
		expectedBatch []int
	}{
		{name: "empty"},
		{name: "one user", phoneCount: 1, expectedBatch: []int{1}},
		{name: "fifty users", phoneCount: 50, expectedBatch: []int{50}},
		{name: "fifty one users", phoneCount: 51, expectedBatch: []int{50, 1}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			phones := make([]string, 0, test.phoneCount)
			for index := 0; index < test.phoneCount; index++ {
				phones = append(phones, fmt.Sprintf("1380000%04d", index))
			}

			requests := buildAlarmSmsRequests(phones)
			if len(requests) != len(test.expectedBatch) {
				t.Fatalf("expected %d requests, got %d", len(test.expectedBatch), len(requests))
			}
			for index, expectedSize := range test.expectedBatch {
				phoneString := tea.StringValue(requests[index].PhoneNumbers)
				actualSize := 0
				if phoneString != "" {
					actualSize = len(strings.Split(phoneString, ","))
				}
				if actualSize != expectedSize {
					t.Fatalf("batch %d expected %d phones, got %d", index, expectedSize, actualSize)
				}
			}
		})
	}
}

func TestSmsClientCacheConcurrentAccess(t *testing.T) {
	smsClientsMu.Lock()
	previous := smsRenterClients
	smsRenterClients = make(map[string]map[string]*SmsTem)
	smsClientsMu.Unlock()
	t.Cleanup(func() {
		smsClientsMu.Lock()
		smsRenterClients = previous
		smsClientsMu.Unlock()
	})

	const workers = 100
	var waitGroup sync.WaitGroup
	for index := 0; index < workers; index++ {
		waitGroup.Add(1)
		go func(index int) {
			defer waitGroup.Done()
			scope := fmt.Sprintf("tenant-%d", index%4)
			setSmsClients(scope, map[string]*SmsTem{"0": {SignName: scope}})
			_ = getSmsClients(scope)
		}(index)
	}
	waitGroup.Wait()
}
