package dao

import (
	"dicloud/internal/common/model"
	"dicloud/util/gormutil"
	"errors"
	"fmt"
	"log"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AlarmDataRecordDao struct {
	db *gorm.DB
}

func NewAlarmDataRecordDao(ctx *gin.Context) *AlarmDataRecordDao {
	return &AlarmDataRecordDao{db: gormutil.GetGormDB(ctx, "")}
}

// 创建报警记录
func (dao *AlarmDataRecordDao) Create(alarmRecord model.AlarmDataRecord) bool {

	err := dao.db.Create(&alarmRecord).Error
	if err != nil {
		log.Println("[COMMON DAO] create alarm record fail: ", err)
		return false
	}

	return true
}

// CreateIfNotExists creates one record for an alarm lifecycle.
func (dao *AlarmDataRecordDao) CreateIfNotExists(alarmRecord model.AlarmDataRecord) (bool, error) {
	existing := model.AlarmDataRecord{}
	err := alarmLifecycleQuery(dao.db.Model(&model.AlarmDataRecord{}), alarmRecord).
		Select("id").
		Take(&existing).Error
	if err == nil {
		return false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return false, err
	}

	result := dao.db.Create(&alarmRecord)
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return false, nil
		}
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func isUniqueViolation(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	pqErr := &pq.Error{}
	return errors.As(err, &pqErr) && pqErr.SQLState() == "23505"
}

func alarmLifecycleQuery(db *gorm.DB, alarmRecord model.AlarmDataRecord) *gorm.DB {
	return db.Where(
		"equipment_id = ? and alarm_code = ? and begin_time = ? and alarm_type = ? and alarm_stage = ?",
		alarmRecord.EquipmentId, alarmRecord.AlarmCode, alarmRecord.BeginTime,
		alarmRecord.AlarmType, alarmRecord.AlarmStage,
	)
}

// 最新一条报警记录
func (dao *AlarmDataRecordDao) FindLatestRecord(alarmRecord model.AlarmDataRecord) (model.AlarmDataRecord, bool) {
	dataTem := model.AlarmDataRecord{}

	err := alarmLifecycleQuery(dao.db.Model(&model.AlarmDataRecord{}), alarmRecord).
		Order("id asc").
		First(&dataTem).Error
	if err != nil {
		log.Println("[COMMON DAO] find latest alarm record fail: ", err)
		return dataTem, false
	}

	return dataTem, true
}

// 最新一条报警记录
func (dao *AlarmDataRecordDao) UpdateRecord(alarmRecord model.AlarmDataRecord) (model.AlarmDataRecord, bool) {

	err := dao.db.Model(&alarmRecord).Select("*").Updates(&alarmRecord).Error
	if err != nil {
		log.Println("[COMMON DAO] update alarm record fail: ", err)
		return alarmRecord, false
	}

	return alarmRecord, true
}

func (dao *AlarmDataRecordDao) CountActiveNormalAlarmByEquipment(equipmentId int) (int, bool) {
	var total int64

	activeLifecycleQuery := dao.db.Model(&model.AlarmDataRecord{}).
		Select("equipment_id, alarm_code, begin_time, alarm_type, alarm_stage").
		Where("equipment_id = ? and alarm_type = ? and alarm_status = ?", equipmentId, "0", 1).
		Group("equipment_id, alarm_code, begin_time, alarm_type, alarm_stage")
	err := dao.db.Table("(?) as active_alarm_lifecycle", activeLifecycleQuery).Count(&total).Error
	if err != nil {
		log.Println("[COMMON DAO] count active equipment alarm record fail: ", err)
		return 0, false
	}

	return int(total), true
}

// 报警记录
func (dao *AlarmDataRecordDao) FindRecords(alarmRecord model.AlarmDataRecord, offset, limit int, startTime, endTime int64) ([]model.AlarmDataRecord, int64, bool) {
	dataList := []model.AlarmDataRecord{}

	err := buildAlarmDataRecordQuery(dao.db, alarmRecord, startTime, endTime).
		Offset(offset).Limit(limit).Order("id desc").Find(&dataList).Error
	if err != nil {
		log.Println("[COMMON DAO] Find alarm record fail: ", err)
		return dataList, 0, false
	}
	var total int64
	if err = buildAlarmDataRecordQuery(dao.db, alarmRecord, startTime, endTime).Count(&total).Error; err != nil {
		log.Println("[COMMON DAO] Count alarm record fail: ", err)
		return dataList, 0, false
	}

	return dataList, total, true
}

// 根据部门查询
func (dao *AlarmDataRecordDao) FindRecordsByEquipment(alarmRecord model.AlarmDataRecord, offset, limit int, equipmentIds []int, startTime, endTime int64) ([]model.AlarmDataRecord, int64, bool) {
	dataList := []model.AlarmDataRecord{}

	err := buildAlarmDataRecordQuery(dao.db, alarmRecord, startTime, endTime).
		Where("equipment_id in ?", equipmentIds).
		Offset(offset).Limit(limit).Order("id desc").Find(&dataList).Error
	if err != nil {
		log.Println("[COMMON DAO] Find alarm record fail: ", err)
		return dataList, 0, false
	}
	var total int64
	if err = buildAlarmDataRecordQuery(dao.db, alarmRecord, startTime, endTime).
		Where("equipment_id in ?", equipmentIds).Count(&total).Error; err != nil {
		log.Println("[COMMON DAO] Count alarm record fail: ", err)
		return dataList, 0, false
	}

	return dataList, total, true
}

// FindDailyAlarmStatisticRecords returns every normal-alarm lifecycle started
// in the requested period. Recovered alarms remain eligible for accumulated
// statistics, so this query deliberately does not filter alarm_status.
func (dao *AlarmDataRecordDao) FindDailyAlarmStatisticRecords(equipmentIds []int, startTime, endTime int64) ([]model.AlarmDataRecord, error) {
	dataList := []model.AlarmDataRecord{}
	if len(equipmentIds) == 0 {
		return dataList, nil
	}

	err := dao.db.Model(&model.AlarmDataRecord{}).
		Select("equipment_id, corp_id, alarm_code, begin_time").
		Where("equipment_id IN ? AND begin_time >= ? AND begin_time <= ?", equipmentIds, startTime, endTime).
		Where("(alarm_type = ? OR alarm_type LIKE ? OR alarm_type LIKE ? OR alarm_type LIKE ?)", "0", "0,%", "%,0", "%,0,%").
		Find(&dataList).Error
	if err != nil {
		return nil, err
	}

	return dataList, nil
}

// FindDailyWarningStatisticRecords returns every warning lifecycle started in
// the requested period. Recovered warnings remain eligible for accumulated
// statistics, so this query deliberately does not filter alarm_status.
func (dao *AlarmDataRecordDao) FindDailyWarningStatisticRecords(equipmentIds []int, startTime, endTime int64) ([]model.AlarmDataRecord, error) {
	dataList := []model.AlarmDataRecord{}
	if len(equipmentIds) == 0 {
		return dataList, nil
	}

	err := dao.db.Model(&model.AlarmDataRecord{}).
		Select("equipment_id, corp_id, alarm_code, begin_time").
		Where("equipment_id IN ? AND begin_time >= ? AND begin_time <= ?", equipmentIds, startTime, endTime).
		Where("(alarm_type = ? OR alarm_type LIKE ? OR alarm_type LIKE ? OR alarm_type LIKE ?)", "1", "1,%", "%,1", "%,1,%").
		Find(&dataList).Error
	if err != nil {
		return nil, err
	}

	return dataList, nil
}

func baseAlarmDataRecordSql(alarmRecord model.AlarmDataRecord) string {
	sqlStr := "1=1"

	if len(alarmRecord.AlarmCode) > 0 {
		sqlStr = sqlStr + " AND alarm_code = " + alarmRecord.AlarmCode
	}
	if len(alarmRecord.AlarmType) > 0 {
		sqlStr = sqlStr + " AND alarm_type like '%" + alarmRecord.AlarmType + "%' "
	}
	if alarmRecord.SiteId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND site_id = %d ", alarmRecord.SiteId)
	}
	if alarmRecord.EquipmentId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id = %d ", alarmRecord.EquipmentId)
	}
	return sqlStr
}

func buildAlarmDataRecordQuery(db *gorm.DB, alarmRecord model.AlarmDataRecord, startTime, endTime int64) *gorm.DB {
	query := db.Model(&alarmRecord).
		Where(baseAlarmDataRecordSql(alarmRecord)).
		Where("alarm_status = ?", alarmRecord.AlarmStatus)
	if startTime > 0 && endTime > 0 {
		query = query.Where("begin_time >= ? AND begin_time <= ?", startTime, endTime)
	}
	return query
}

func (dao *AlarmDataRecordDao) DeleteRecord(alarmRecord model.AlarmDataRecord) bool {

	err := dao.db.Model(&alarmRecord).Delete(&alarmRecord).Error
	if err != nil {
		log.Println("[COMMON DAO] delete alarm record fail: ", err)
		return false
	}

	return true
}

// 单个设备报警记录统计
func (dao *AlarmDataRecordDao) CountRecords(alarmRecord model.AlarmDataRecord) ([]model.EquipmentAlarmCount, bool) {
	var resultData = []model.EquipmentAlarmCount{}

	err := buildAlarmDataRecordQuery(dao.db, alarmRecord, 0, 0).
		Where("equipment_id in ?", alarmRecord.EquipmentIds).
		Group("equipment_id").
		Select("count(1) as alarm_count,equipment_id").
		Find(&resultData).Error
	if err != nil {
		log.Println("[COMMON DAO] Count alarm record fail: ", err)
		return resultData, false
	}

	return resultData, true
}

func (dao *AlarmDataRecordDao) CountRecordStatus(alarmRecord model.AlarmDataRecord) ([]model.EquipmentAlarmStatus, bool) {
	resultData := []model.EquipmentAlarmStatus{}
	err := buildAlarmDataRecordQuery(dao.db, alarmRecord, 0, 0).
		Where("equipment_id in ?", alarmRecord.EquipmentIds).
		Group("equipment_id,alarm_type").
		Select("count(1) as alarm_count,equipment_id,alarm_type").
		Find(&resultData).Error
	if err != nil {
		log.Println("[COMMON DAO] Count alarm status fail: ", err)
		return resultData, false
	}

	return resultData, true
}
