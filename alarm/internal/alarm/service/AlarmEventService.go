package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	alarmdao "alarm/internal/alarm/dao"
	alarmmodel "alarm/internal/alarm/model"
	businessmodel "alarm/internal/business/model"
	commonmodel "alarm/internal/common/model"
	"alarm/models/enum"
	"alarm/util/gormutil"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const defaultAlarmDeliveryMaxAttempts = 5

type alarmLifecycleIdentity struct {
	EquipmentID int    `json:"equipmentId"`
	AlarmCode   string `json:"alarmCode"`
	Begin       int64  `json:"begin"`
	AlarmType   string `json:"alarmType"`
	AlarmStage  int    `json:"alarmStage"`
}

func AcceptAlarmEvent(ctx *gin.Context, message *primitive.MessageExt) (bool, error) {
	if message == nil {
		return false, fmt.Errorf("alarm message is nil")
	}
	eventID := strings.TrimSpace(message.MsgId)
	if eventID == "" {
		return false, fmt.Errorf("alarm message id is empty")
	}

	var data commonmodel.AlarmMQModel
	if err := json.Unmarshal(message.Body, &data); err != nil {
		return false, fmt.Errorf("decode alarm message: %w", err)
	}
	if data.DbName != "" {
		ctx.Set("dbName", data.DbName)
	}
	if data.EquipmentId <= 0 || strings.TrimSpace(data.AlarmCode) == "" {
		return false, fmt.Errorf("alarm equipment or alarm code is invalid")
	}

	db := gormutil.GetGormDB(ctx, "")
	if db == nil {
		return false, fmt.Errorf("alarm database is unavailable")
	}
	payloadHash := sha256Hex(message.Body)
	if handled, err := handleExistingAlarmEvent(db, eventID, payloadHash); handled || err != nil {
		return false, err
	}

	lifecycleKey, err := buildAlarmLifecycleKey(data)
	if err != nil {
		return false, err
	}
	tx := db.Begin()
	if tx.Error != nil {
		return false, tx.Error
	}
	rollback := func(err error) (bool, error) {
		tx.Rollback()
		return false, err
	}

	var equipment businessmodel.Equipment
	if err := tx.Table("equipment").Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("id = ?", data.EquipmentId).Take(&equipment).Error; err != nil {
		return rollback(fmt.Errorf("find alarm equipment %d: %w", data.EquipmentId, err))
	}
	if strings.TrimSpace(equipment.EquipmentName) == "" {
		return rollback(fmt.Errorf("alarm equipment name is empty: %d", data.EquipmentId))
	}

	var existing alarmmodel.AlarmEventInbox
	err = tx.Where("event_id = ?", eventID).Take(&existing).Error
	if err == nil {
		if err := finishExistingAlarmEvent(tx, existing, payloadHash); err != nil {
			return rollback(err)
		}
		if err := tx.Commit().Error; err != nil {
			return false, err
		}
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return rollback(err)
	}

	now := time.Now()
	inbox := alarmmodel.AlarmEventInbox{
		EventID: eventID, LifecycleKey: lifecycleKey, PayloadHash: payloadHash,
		Payload: string(message.Body), Status: alarmmodel.AlarmInboxAccepted,
		ConflictHash: alarmmodel.AlarmAuditUnset,
		CreateTime:   now, UpdateTime: now,
	}
	if err := tx.Create(&inbox).Error; err != nil {
		tx.Rollback()
		if isAlarmUniqueViolation(err) {
			handled, duplicateErr := handleExistingAlarmEvent(db, eventID, payloadHash)
			if handled || duplicateErr != nil {
				return false, duplicateErr
			}
		}
		return false, err
	}

	data.SiteName = equipment.SiteName
	data.EquipmentName = equipment.EquipmentName
	data.CorpId = equipment.CorpId
	if equipment.SiteId == 0 || strings.TrimSpace(data.SiteName) == "" {
		data.SiteName = "-"
	}
	normalizedPayload, err := json.Marshal(data)
	if err != nil {
		return rollback(err)
	}

	recordID, err := applyAlarmLifecycle(tx, data)
	if err != nil {
		return rollback(err)
	}
	deliveries := buildAlarmDeliveries(eventID, lifecycleKey, recordID, data, string(normalizedPayload))
	for index := range deliveries {
		if err := tx.Create(&deliveries[index]).Error; err != nil {
			return rollback(err)
		}
	}
	if err := tx.Model(&alarmmodel.AlarmEventInbox{}).Where("id = ?", inbox.ID).Updates(map[string]interface{}{
		"alarm_record_id": recordID, "processed_at": now.Unix(), "update_time": now,
	}).Error; err != nil {
		return rollback(err)
	}
	if err := tx.Commit().Error; err != nil {
		return false, err
	}
	return true, nil
}

func handleExistingAlarmEvent(db *gorm.DB, eventID, payloadHash string) (bool, error) {
	inbox, err := alarmdao.NewDeliveryDAOWithDB(db).FindInboxByEventID(eventID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if inbox.PayloadHash == payloadHash {
		return true, nil
	}
	if err := db.Model(&alarmmodel.AlarmEventInbox{}).Where("id = ?", inbox.ID).Updates(map[string]interface{}{
		"status": alarmmodel.AlarmInboxConflict, "conflict_hash": payloadHash, "update_time": time.Now(),
	}).Error; err != nil {
		return true, err
	}
	log.Printf("[ALARM] message payload conflict, msgId=%s", eventID)
	return true, nil
}

func finishExistingAlarmEvent(tx *gorm.DB, inbox alarmmodel.AlarmEventInbox, payloadHash string) error {
	if inbox.PayloadHash == payloadHash {
		return nil
	}
	log.Printf("[ALARM] message payload conflict, msgId=%s", inbox.EventID)
	return tx.Model(&alarmmodel.AlarmEventInbox{}).Where("id = ?", inbox.ID).Updates(map[string]interface{}{
		"status": alarmmodel.AlarmInboxConflict, "conflict_hash": payloadHash, "update_time": time.Now(),
	}).Error
}

func applyAlarmLifecycle(tx *gorm.DB, data commonmodel.AlarmMQModel) (uint, error) {
	identity := commonmodel.AlarmDataRecord{
		EquipmentId: data.EquipmentId, AlarmCode: data.AlarmCode, BeginTime: data.Begin,
		AlarmType: data.AlarmType, AlarmStage: data.AlarmStage,
	}
	var existing commonmodel.AlarmDataRecord
	find := tx.Model(&commonmodel.AlarmDataRecord{}).
		Where("equipment_id = ? AND alarm_code = ? AND begin_time = ? AND alarm_type = ? AND alarm_stage = ?",
			identity.EquipmentId, identity.AlarmCode, identity.BeginTime, identity.AlarmType, identity.AlarmStage).
		Order("id ASC").Take(&existing)

	if data.End == 0 {
		if find.Error == nil {
			return existing.ID, nil
		}
		if !errors.Is(find.Error, gorm.ErrRecordNotFound) {
			return 0, find.Error
		}
		record := commonmodel.AlarmDataRecord{
			SiteId: data.SiteId, EquipmentId: data.EquipmentId, AlarmStatus: 1,
			AlarmCode: data.AlarmCode, AlarmType: data.AlarmType, AlarmStage: data.AlarmStage,
			BeginTime: data.Begin, EndTime: 0, AlarmValue: fmt.Sprint(data.AlarmValue), RecoverValue: "0",
			SiteName: data.SiteName, EquipmentName: data.EquipmentName, CorpId: data.CorpId,
		}
		const savepoint = "alarm_lifecycle_create"
		if err := tx.SavePoint(savepoint).Error; err != nil {
			return 0, err
		}
		if err := tx.Create(&record).Error; err != nil {
			if !isAlarmUniqueViolation(err) {
				return 0, err
			}
			if rollbackErr := tx.RollbackTo(savepoint).Error; rollbackErr != nil {
				return 0, rollbackErr
			}
			if findErr := tx.Model(&commonmodel.AlarmDataRecord{}).
				Where("equipment_id = ? AND alarm_code = ? AND begin_time = ? AND alarm_type = ? AND alarm_stage = ?",
					identity.EquipmentId, identity.AlarmCode, identity.BeginTime, identity.AlarmType, identity.AlarmStage).
				Order("id ASC").Take(&existing).Error; findErr != nil {
				return 0, findErr
			}
			return existing.ID, nil
		}
		return record.ID, nil
	}

	if data.End == 1 {
		if find.Error == nil {
			return existing.ID, nil
		}
		if errors.Is(find.Error, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, find.Error
	}

	// Alarm level/type may have changed while an alarm was active. The trigger
	// still carries the original begin time, so use the stable lifecycle fields
	// to recover that record instead of leaving it permanently active.
	if errors.Is(find.Error, gorm.ErrRecordNotFound) {
		find = tx.Model(&commonmodel.AlarmDataRecord{}).
			Where("equipment_id = ? AND alarm_code = ? AND begin_time = ? AND alarm_status = ?",
				identity.EquipmentId, identity.AlarmCode, identity.BeginTime, 1).
			Order("id ASC").Take(&existing)
	}
	if find.Error != nil {
		return 0, find.Error
	}
	if existing.AlarmStatus != 0 {
		if err := tx.Model(&commonmodel.AlarmDataRecord{}).Where("id = ? AND alarm_status <> 0", existing.ID).
			Updates(map[string]interface{}{
				"alarm_status": 0, "end_time": data.End, "recover_value": fmt.Sprint(data.AlarmValue),
			}).Error; err != nil {
			return 0, err
		}
	}
	return existing.ID, nil
}

func buildAlarmDeliveries(eventID, lifecycleKey string, recordID uint, data commonmodel.AlarmMQModel, payload string) []alarmmodel.AlarmNotificationDelivery {
	now := time.Now()
	base := func(channel, recipientKey string, recipientIDs []int64, status, lastError string) alarmmodel.AlarmNotificationDelivery {
		idsJSON, _ := json.Marshal(recipientIDs)
		if lastError == "" {
			lastError = alarmmodel.AlarmAuditUnset
		}
		nextRetryAt := now.Unix()
		if status == alarmmodel.AlarmDeliverySkipped || status == alarmmodel.AlarmDeliveryDead {
			nextRetryAt = 0
		}
		return alarmmodel.AlarmNotificationDelivery{
			EventID: eventID, LifecycleKey: lifecycleKey, AlarmRecordID: recordID,
			Channel: channel, RecipientKey: recipientKey, RecipientIDs: string(idsJSON), AlarmPayload: payload,
			Status: status, MaxAttempts: defaultAlarmDeliveryMaxAttempts, NextRetryAt: nextRetryAt,
			LeaseOwner: alarmmodel.AlarmAuditUnset, ProviderRequestID: alarmmodel.AlarmAuditUnset,
			ResponseCode: alarmmodel.AlarmAuditUnset, Response: alarmmodel.AlarmAuditUnset,
			LastError: lastError, CreateTime: now, UpdateTime: now,
		}
	}

	userIDs := parseAlarmUserIDs(data.AlarmUsers)
	channels := uniqueTrimmedStrings(strings.Split(data.AlarmChannel, ","))
	deliveries := make([]alarmmodel.AlarmNotificationDelivery, 0, len(channels)+1)
	for _, channel := range channels {
		switch channel {
		case "":
			continue
		case enum.TAG_ALARM_CHANNEL_SHORTMSG:
			if len(userIDs) == 0 {
				deliveries = append(deliveries, base(channel, "none", nil, alarmmodel.AlarmDeliverySkipped, "alarm recipients are empty"))
				continue
			}
			for start := 0; start < len(userIDs); start += 50 {
				end := min(start+50, len(userIDs))
				batch := userIDs[start:end]
				deliveries = append(deliveries, base(channel, recipientDigest(channel, batch), batch, alarmmodel.AlarmDeliveryPending, ""))
			}
		default:
			deliveries = append(deliveries, base(channel, "unsupported", nil, alarmmodel.AlarmDeliverySkipped, "alarm channel is not supported"))
		}
	}
	if data.AlarmType == "0" && (data.End == 0 || data.End > 1) {
		recipientKey := fmt.Sprintf("equipment:%d", data.EquipmentId)
		deliveries = append(deliveries, base(alarmmodel.AlarmDeliveryChannelEquipmentCount, recipientKey,
			[]int64{int64(data.EquipmentId)}, alarmmodel.AlarmDeliveryPending, ""))
	}
	return deliveries
}

func parseAlarmUserIDs(value string) []int64 {
	seen := make(map[int64]struct{})
	for _, item := range strings.Split(value, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(item), 10, 64)
		if err == nil && id > 0 {
			seen[id] = struct{}{}
		}
	}
	result := make([]int64, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func uniqueTrimmedStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func buildAlarmLifecycleKey(data commonmodel.AlarmMQModel) (string, error) {
	identityJSON, err := json.Marshal(alarmLifecycleIdentity{
		EquipmentID: data.EquipmentId, AlarmCode: data.AlarmCode, Begin: data.Begin,
		AlarmType: data.AlarmType, AlarmStage: data.AlarmStage,
	})
	if err != nil {
		return "", err
	}
	return sha256Hex(identityJSON), nil
}

func recipientDigest(channel string, userIDs []int64) string {
	parts := make([]string, len(userIDs))
	for index, id := range userIDs {
		parts[index] = strconv.FormatInt(id, 10)
	}
	return sha256Hex([]byte(channel + "|" + strings.Join(parts, ",")))
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func isAlarmUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	pqErr := &pq.Error{}
	return errors.As(err, &pqErr) && pqErr.SQLState() == "23505"
}
