package dao

import (
	migration "alarm/internal/migration"
	"fmt"
	"time"

	alarmmodel "alarm/internal/alarm/model"
	commonmodel "alarm/internal/common/model"
	"alarm/util/gormutil"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DeliveryDAO struct {
	db *gorm.DB
}

const alarmDeliveryLease = 150 * time.Second

func NewDeliveryDAO(ctx *gin.Context) *DeliveryDAO {
	return &DeliveryDAO{db: gormutil.GetGormDB(ctx, "")}
}

func NewDeliveryDAOWithDB(db *gorm.DB) *DeliveryDAO {
	return &DeliveryDAO{db: db}
}

func (dao *DeliveryDAO) DB() *gorm.DB {
	return dao.db
}

func (dao *DeliveryDAO) FindInboxByEventID(eventID string) (alarmmodel.AlarmEventInbox, error) {
	var inbox alarmmodel.AlarmEventInbox
	err := dao.db.Where("event_id = ?", eventID).Take(&inbox).Error
	return inbox, err
}

func (dao *DeliveryDAO) FindDue(now time.Time, limit int) ([]alarmmodel.AlarmNotificationDelivery, error) {
	if err := dao.db.Model(&alarmmodel.AlarmNotificationDelivery{}).
		Where("status = ? AND lease_until > 0 AND lease_until <= ? AND max_attempts > 0 AND attempt_count >= max_attempts",
			alarmmodel.AlarmDeliverySending, now.Unix()).
		Updates(map[string]interface{}{
			"status": alarmmodel.AlarmDeliveryDead, "lease_owner": alarmmodel.AlarmAuditUnset, "lease_until": 0,
			"next_retry_at": 0, "update_time": now,
		}).Error; err != nil {
		return nil, err
	}

	terminal := []string{alarmmodel.AlarmDeliverySuccess, alarmmodel.AlarmDeliveryDead, alarmmodel.AlarmDeliverySkipped}
	var data []alarmmodel.AlarmNotificationDelivery
	err := dao.db.Model(&alarmmodel.AlarmNotificationDelivery{}).
		Where(`((status = ? AND next_retry_at <= ?)
			OR (status IN ? AND next_retry_at > 0 AND next_retry_at <= ?)
			OR (status = ? AND lease_until > 0 AND lease_until <= ?))
			AND (max_attempts = 0 OR attempt_count < max_attempts)
			AND NOT EXISTS (
				SELECT 1 FROM alarm_notification_delivery older
				WHERE older.lifecycle_key = alarm_notification_delivery.lifecycle_key
				AND older.event_id <> alarm_notification_delivery.event_id
				AND older.id < alarm_notification_delivery.id
				AND older.status NOT IN ?
			)`,
			alarmmodel.AlarmDeliveryPending, now.Unix(),
			[]string{alarmmodel.AlarmDeliveryRetryableFailed, alarmmodel.AlarmDeliveryUnknown}, now.Unix(),
			alarmmodel.AlarmDeliverySending, now.Unix(), terminal).
		Order("id ASC").Limit(limit).Find(&data).Error
	return data, err
}

func (dao *DeliveryDAO) TryClaim(id uint, owner string, now time.Time) (bool, error) {
	result := dao.db.Model(&alarmmodel.AlarmNotificationDelivery{}).
		Where(`id = ? AND (max_attempts = 0 OR attempt_count < max_attempts)
			AND ((status IN ?) OR (status = ? AND lease_until > 0 AND lease_until <= ?))`,
			id,
			[]string{alarmmodel.AlarmDeliveryPending, alarmmodel.AlarmDeliveryRetryableFailed, alarmmodel.AlarmDeliveryUnknown},
			alarmmodel.AlarmDeliverySending, now.Unix()).
		Updates(map[string]interface{}{
			"status": alarmmodel.AlarmDeliverySending, "lease_owner": owner,
			"lease_until":   now.Add(alarmDeliveryLease).Unix(),
			"attempt_count": gorm.Expr("attempt_count + 1"), "update_time": now,
		})
	return result.RowsAffected == 1, result.Error
}

func (dao *DeliveryDAO) UpdateAttempt(id uint, owner string, values map[string]interface{}) error {
	values["lease_owner"] = alarmmodel.AlarmAuditUnset
	values["lease_until"] = 0
	values["update_time"] = time.Now()
	result := dao.db.Model(&alarmmodel.AlarmNotificationDelivery{}).
		Where("id = ? AND lease_owner = ?", id, owner).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (dao *DeliveryDAO) DeleteFinishedBefore(before time.Time) error {
	return dao.db.Model(&alarmmodel.AlarmNotificationDelivery{}).
		Where("status IN ? AND update_time < ?", []string{
			alarmmodel.AlarmDeliverySuccess, alarmmodel.AlarmDeliveryDead, alarmmodel.AlarmDeliverySkipped,
		}, before).Delete(&alarmmodel.AlarmNotificationDelivery{}).Error
}

func (dao *DeliveryDAO) DeleteInboxBefore(before time.Time) error {
	return dao.db.Model(&alarmmodel.AlarmEventInbox{}).
		Where(`create_time < ? AND NOT EXISTS (
			SELECT 1 FROM alarm_notification_delivery delivery
			WHERE delivery.event_id = alarm_event_inbox.event_id
		)`, before).Delete(&alarmmodel.AlarmEventInbox{}).Error
}

func ValidateReliabilitySchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("alarm database is nil")
	}
	for _, table := range []string{"alarm_event_inbox", "alarm_notification_delivery", "alarm_data_record"} {
		var marker int
		if err := db.Raw("SELECT 1 FROM " + table + " LIMIT 1").Scan(&marker).Error; err != nil {
			return fmt.Errorf("required alarm table %s is unavailable: %w", table, err)
		}
	}
	requiredIndexes := []struct {
		model interface{}
		name  string
	}{
		{&alarmmodel.AlarmEventInbox{}, "uni_alarm_event_inbox_event"},
		{&alarmmodel.AlarmNotificationDelivery{}, "uni_alarm_delivery_target"},
		{&alarmmodel.AlarmNotificationDelivery{}, "idx_alarm_delivery_due"},
		{&alarmmodel.AlarmNotificationDelivery{}, "idx_alarm_delivery_lifecycle"},
		{&commonmodel.AlarmDataRecord{}, "uni_alarm_record_lifecycle"},
	}
	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			return fmt.Errorf("required alarm index %s is unavailable", index.name)
		}
	}
	return nil
}

// AutoMigrateReliabilitySchema 使用 GORM 创建缺失的告警可靠投递表和索引。
// 已有表只做结构检查，避免启动告警程序时修改生产库旧表。
func AutoMigrateReliabilitySchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("alarm database is nil")
	}
	if err := migration.CreateMissingTablesAndValidateExisting(db,
		&alarmmodel.AlarmEventInbox{},
		&alarmmodel.AlarmNotificationDelivery{},
		&commonmodel.AlarmDataRecord{},
	); err != nil {
		return err
	}
	return validateReliabilityIndexes(db)
}

func validateReliabilityIndexes(db *gorm.DB) error {
	requiredIndexes := []struct {
		model interface{}
		name  string
	}{
		{&alarmmodel.AlarmEventInbox{}, "uni_alarm_event_inbox_event"},
		{&alarmmodel.AlarmNotificationDelivery{}, "uni_alarm_delivery_target"},
		{&alarmmodel.AlarmNotificationDelivery{}, "idx_alarm_delivery_due"},
		{&alarmmodel.AlarmNotificationDelivery{}, "idx_alarm_delivery_lifecycle"},
		{&commonmodel.AlarmDataRecord{}, "uni_alarm_record_lifecycle"},
	}
	for _, index := range requiredIndexes {
		if !db.Migrator().HasIndex(index.model, index.name) {
			return fmt.Errorf("required alarm index %s is unavailable; existing tables are not changed automatically", index.name)
		}
	}
	return nil
}
