package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	alarmdao "alarm/internal/alarm/dao"
	alarmmodel "alarm/internal/alarm/model"
	authdao "alarm/internal/auth/dao"
	businessdao "alarm/internal/business/dao"
	businessmodel "alarm/internal/business/model"
	commondao "alarm/internal/common/dao"
	commonmodel "alarm/internal/common/model"
	"alarm/models/enum"
	usermodel "alarm/models/model"
	"alarm/util/gormutil"

	dysmsapi20170525 "github.com/alibabacloud-go/dysmsapi-20170525/v4/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const alarmDeliveryBatchSize = 20

type deliveryAttemptResult struct {
	Status            string
	ResponseCode      string
	Response          string
	LastError         string
	ProviderRequestID string
}

func ProcessAlarmDeliveries(scopeCtx *gin.Context) {
	dao := alarmdao.NewDeliveryDAO(scopeCtx)
	if dao.DB() == nil {
		log.Println("[ALARM] notification delivery database is nil")
		return
	}
	owner := alarmWorkerOwner()
	items, err := dao.FindDue(time.Now(), alarmDeliveryBatchSize)
	if err != nil {
		log.Println("[ALARM] find notification delivery fail:", err)
		return
	}

	var waitGroup sync.WaitGroup
	for index := range items {
		item := items[index]
		claimed, err := dao.TryClaim(item.ID, owner, time.Now())
		if err != nil || !claimed {
			continue
		}
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			processAlarmDelivery(scopeCtx, item, owner)
		}()
	}
	waitGroup.Wait()
}

func processAlarmDelivery(scopeCtx *gin.Context, item alarmmodel.AlarmNotificationDelivery, owner string) {
	result := deliveryAttemptResult{}
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				result = deliveryAttemptResult{
					Status:    alarmmodel.AlarmDeliveryRetryableFailed,
					LastError: fmt.Sprintf("notification delivery panic: %v", recovered),
				}
			}
		}()
		result = executeAlarmDelivery(newDeliveryContext(scopeCtx), item)
	}()

	values, attemptCount := deliveryAttemptUpdates(item, result, time.Now())

	if err := alarmdao.NewDeliveryDAO(newDeliveryContext(scopeCtx)).UpdateAttempt(item.ID, owner, values); err != nil {
		log.Printf("[ALARM] update notification delivery fail, eventId=%s deliveryId=%d err=%v", item.EventID, item.ID, err)
		return
	}
	finalStatus := fmt.Sprint(values["status"])
	if finalStatus != alarmmodel.AlarmDeliverySuccess {
		log.Printf("[ALARM] notification delivery result, eventId=%s deliveryId=%d channel=%s status=%s attempt=%d",
			item.EventID, item.ID, item.Channel, finalStatus, attemptCount)
	}
}

func deliveryAttemptUpdates(item alarmmodel.AlarmNotificationDelivery, result deliveryAttemptResult, now time.Time) (map[string]interface{}, int) {
	attemptCount := item.AttemptCount + 1
	values := map[string]interface{}{
		"status": result.Status, "response_code": deliveryAuditText(result.ResponseCode),
		"response": deliveryAuditText(result.Response), "last_error": deliveryAuditText(result.LastError),
		"provider_request_id": deliveryAuditText(result.ProviderRequestID),
	}
	if result.Status == alarmmodel.AlarmDeliverySuccess {
		values["sent_at"] = now.Unix()
		values["next_retry_at"] = int64(0)
	} else if result.Status == alarmmodel.AlarmDeliveryRetryableFailed || result.Status == alarmmodel.AlarmDeliveryUnknown {
		if attemptCount >= item.MaxAttempts {
			values["status"] = alarmmodel.AlarmDeliveryDead
			values["next_retry_at"] = int64(0)
		} else {
			values["next_retry_at"] = now.Add(alarmDeliveryRetryDelay(attemptCount)).Unix()
		}
	} else {
		values["next_retry_at"] = int64(0)
	}
	return values, attemptCount
}

func executeAlarmDelivery(ctx *gin.Context, item alarmmodel.AlarmNotificationDelivery) deliveryAttemptResult {
	var data commonmodel.AlarmMQModel
	if err := json.Unmarshal([]byte(item.AlarmPayload), &data); err != nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryDead, LastError: "invalid alarm payload"}
	}
	var recipientIDs []int64
	if err := json.Unmarshal([]byte(item.RecipientIDs), &recipientIDs); err != nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryDead, LastError: "invalid recipient ids"}
	}

	switch item.Channel {
	case enum.TAG_ALARM_CHANNEL_SHORTMSG:
		return sendReliableSMS(ctx, data, recipientIDs)
	case alarmmodel.AlarmDeliveryChannelEquipmentCount:
		return updateReliableEquipmentAlarmCount(ctx, data)
	default:
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliverySkipped, LastError: "unsupported alarm channel"}
	}
}

func sendReliableSMS(ctx *gin.Context, data commonmodel.AlarmMQModel, recipientIDs []int64) deliveryAttemptResult {
	users, err := findAlarmUsersByIDs(ctx, recipientIDs)
	if err != nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryRetryableFailed, LastError: "load sms recipients failed"}
	}
	phones := make([]string, 0, len(users))
	for _, user := range users {
		if phone := strings.TrimSpace(user.Phone); phone != "" {
			phones = append(phones, phone)
		}
	}
	phones = uniqueSortedStrings(phones)
	if len(phones) == 0 {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliverySkipped, LastError: "sms recipients have no phone"}
	}

	sms, err := findSMSClientForDelivery(ctx, data.CorpId)
	if err != nil || sms == nil || sms.CurClient == nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryRetryableFailed, LastError: "sms client is not configured"}
	}
	request, err := buildReliableSMSRequest(sms, data, phones)
	if err != nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryDead, LastError: err.Error()}
	}
	runtimeOptions := (&util.RuntimeOptions{}).SetConnectTimeout(10000).SetReadTimeout(10000)
	response, err := sms.CurClient.SendSmsWithOptions(request, runtimeOptions)
	if err != nil {
		return classifyDeliveryError(err)
	}
	if response == nil || response.Body == nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryUnknown, LastError: "sms response is empty"}
	}
	code := tea.StringValue(response.Body.Code)
	message := tea.StringValue(response.Body.Message)
	requestID := tea.StringValue(response.Body.RequestId)
	if strings.EqualFold(code, "OK") {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliverySuccess, ResponseCode: code, Response: message, ProviderRequestID: requestID}
	}
	status := alarmmodel.AlarmDeliveryDead
	if isRetryableProviderCode(code + " " + message) {
		status = alarmmodel.AlarmDeliveryRetryableFailed
	}
	return deliveryAttemptResult{Status: status, ResponseCode: code, Response: message, LastError: message, ProviderRequestID: requestID}
}

func buildReliableSMSRequest(sms *SmsTem, data commonmodel.AlarmMQModel, phones []string) (*dysmsapi20170525.SendSmsRequest, error) {
	templateCode := sms.AlarmTemplateCode
	templateValues := map[string]string{
		sms.AlarmSite: data.SiteName, sms.AlarmEquipment: data.EquipmentName,
		sms.AlarmCoding: data.AlarmCode, sms.AlarmValue: fmt.Sprint(data.AlarmValue),
	}
	if data.End > 1 {
		templateCode = sms.RecoverTemplateCode
		templateValues = map[string]string{
			sms.RecoverSite: data.SiteName, sms.RecoverEquipment: data.EquipmentName,
			sms.RecoverCoding: data.AlarmCode, sms.RecoverValue: fmt.Sprint(data.AlarmValue),
		}
	}
	if strings.TrimSpace(sms.SignName) == "" || strings.TrimSpace(templateCode) == "" {
		return nil, fmt.Errorf("sms sign or template is empty")
	}
	templateJSON, err := json.Marshal(templateValues)
	if err != nil {
		return nil, err
	}
	return &dysmsapi20170525.SendSmsRequest{
		PhoneNumbers: tea.String(strings.Join(phones, ",")), SignName: tea.String(sms.SignName),
		TemplateCode: tea.String(templateCode), TemplateParam: tea.String(string(templateJSON)),
	}, nil
}

func updateReliableEquipmentAlarmCount(ctx *gin.Context, data commonmodel.AlarmMQModel) deliveryAttemptResult {
	alarmCount, ok := commondao.NewAlarmDataRecordDao(ctx).CountActiveNormalAlarmByEquipment(data.EquipmentId)
	if !ok {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryRetryableFailed, LastError: "count active equipment alarms failed"}
	}
	_, ok = businessdao.NewEquipmentDAO(ctx).UpdateEquipmentAlarmTriggerNum(businessmodel.Equipment{
		ID: uint(data.EquipmentId), AlarmTriggerNum: alarmCount,
	})
	if !ok {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryRetryableFailed, LastError: "update equipment alarm count failed"}
	}
	return deliveryAttemptResult{Status: alarmmodel.AlarmDeliverySuccess, Response: "equipment alarm count updated"}
}

func findAlarmUsersByIDs(ctx *gin.Context, ids []int64) ([]usermodel.User, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	return authdao.NewUserDAO(ctx).GetUsersByIdArr(ids)
}

func findSMSClientForDelivery(ctx *gin.Context, corpID int) (*SmsTem, error) {
	scope := getSmsScopeKey(ctx)
	clients := getSmsClients(scope)
	if clients == nil {
		if err := InitAllSmsClient(ctx); err != nil {
			return nil, err
		}
		clients = getSmsClients(scope)
	}
	for visited := 0; visited < 100; visited++ {
		if client := clients[strconv.Itoa(corpID)]; client != nil {
			return client, nil
		}
		if corpID == 0 {
			break
		}
		parentID, err := findParentCorpID(ctx, corpID)
		if err != nil {
			return nil, err
		}
		corpID = parentID
	}
	return nil, gorm.ErrRecordNotFound
}

func findParentCorpID(ctx *gin.Context, corpID int) (int, error) {
	db := gormutil.GetGormDB(ctx, "")
	if db == nil {
		return 0, fmt.Errorf("corp database is unavailable")
	}
	var corp usermodel.Corp
	if err := db.Table("corp").Select("id, parent_id").Where("id = ?", corpID).Take(&corp).Error; err != nil {
		return 0, err
	}
	return corp.ParentId, nil
}

func classifyDeliveryError(err error) deliveryAttemptResult {
	if err == nil {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliverySuccess}
	}
	var networkError net.Error
	if errors.As(err, &networkError) && networkError.Timeout() {
		return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryUnknown, LastError: "provider request timeout"}
	}
	return deliveryAttemptResult{Status: alarmmodel.AlarmDeliveryRetryableFailed, LastError: "provider request failed"}
}

func isRetryableProviderCode(value string) bool {
	value = strings.ToLower(value)
	return strings.Contains(value, "limit") || strings.Contains(value, "busy") ||
		strings.Contains(value, "system") || strings.Contains(value, "timeout")
}

func alarmDeliveryRetryDelay(attempt int) time.Duration {
	delays := []time.Duration{time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute, 30 * time.Minute}
	if attempt <= 0 {
		return delays[0]
	}
	if attempt > len(delays) {
		return delays[len(delays)-1]
	}
	return delays[attempt-1]
}

func alarmWorkerOwner() string {
	host, _ := os.Hostname()
	return fmt.Sprintf("%s:%d", host, os.Getpid())
}

func newDeliveryContext(parent *gin.Context) *gin.Context {
	ctx := &gin.Context{}
	if parent != nil {
		if dbName, exists := parent.Get("dbName"); exists {
			ctx.Set("dbName", dbName)
		}
	}
	return ctx
}

func uniqueSortedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func truncateDeliveryText(value string) string {
	const maxLength = 1000
	if len(value) <= maxLength {
		return value
	}
	return value[:maxLength]
}

func deliveryAuditText(value string) string {
	value = truncateDeliveryText(value)
	if value == "" {
		return alarmmodel.AlarmAuditUnset
	}
	return value
}
