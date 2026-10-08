package dao

import (
	"alarm/internal/common/model"
	"alarm/util/gormutil"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StatisticDataDao struct {
	db *gorm.DB
}

func NewStatisticDataDao(ctx *gin.Context) *StatisticDataDao {
	return &StatisticDataDao{db: gormutil.GetGormDB(ctx, "")}
}

// 保存
func (dao *StatisticDataDao) Create(stsd model.StatisticData) bool {

	stsd.ID = 0
	err := dao.db.Table(stsd.GetTableName()).Create(&stsd).Error
	if err != nil {
		log.Println("Failed to create statistic data: ", err)
		return false
	}

	return true
}

// 是否存在当前标记数据
func (dao *StatisticDataDao) ExistDateMark(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	// 匹配标记的数据
	if model.StatisticHealthMap[stsd.StatisticCode] > 0 {
		// 设备的SOH匹配, 健康度统计(更新最新的健康度, SOH优, SOH良, SOH中, SOH差)
		// 如果是健康度查询, 可能在一个统计周期变化的情况, 将数据更新到旧状态上
		// 二级统计不用处理, 二级统计只计算累加即可
		err := dao.db.Table(stsd.GetTableName()).Where("equipment_id = ? AND data_source_type = ? AND statistic_code like 'SOH%' AND mark_date = ?", stsd.EquipmentId, stsd.DataSourceType, stsd.MarkDate).First(&dataTem).Error
		if err != nil || dataTem.ID == 0 {
			log.Println("Failed to find equipment SOH dateMark: ", err)
			return dataTem, false
		}
	} else {
		err := dao.db.Table(stsd.GetTableName()).Where("equipment_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.EquipmentId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).First(&dataTem).Error
		if err != nil || dataTem.ID == 0 {
			log.Println("Failed to find equipment dateMark: ", err)
			return dataTem, false
		}
	}

	return dataTem, true
}

// 是否存在当前标记数据
func (dao *StatisticDataDao) ExistAlarmDateMark(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	var sqlStr = " 1=1"
	if len(stsd.StatisticAlarmType) > 0 {
		sqlStr = sqlStr + "  AND statistic_alarm_type = '" + stsd.StatisticAlarmType + "'"
	}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where(sqlStr+" AND equipment_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.EquipmentId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		log.Println("Failed to find equipment alarm dateMark: ", err)
		return dataTem, false
	}

	return dataTem, true
}

// 是否存在当前标记数据
func (dao *StatisticDataDao) ExistSiteDateMark(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where("site_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.SiteId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		log.Println("Failed to find site dateMark: ", err)
		return dataTem, false
	}

	return dataTem, true
}

// 是否存在当前标记数据
func (dao *StatisticDataDao) ExistCorpDateMark(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where("corp_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.CorpId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		log.Println("Failed to find corp dateMark: ", err)
		return dataTem, false
	}

	return dataTem, true
}
func (dao *StatisticDataDao) ExistCorpAlarmDateMark(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	var sqlStr = " 1=1"
	if len(stsd.StatisticAlarmType) > 0 {
		sqlStr = sqlStr + "  AND statistic_alarm_type = '" + stsd.StatisticAlarmType + "'"
	}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where(sqlStr+" AND corp_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.CorpId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		log.Println("Failed to find corp dateMark: ", err)
		return dataTem, false
	}

	return dataTem, true
}

// 更新
func (dao *StatisticDataDao) Update(stsd model.StatisticData) bool {

	// 更新, select *(全部字段更新)
	err := dao.db.Table(stsd.GetTableName()).Select("*").Where("id = ?", stsd.ID).Updates(&stsd).Error
	if err != nil {
		log.Println("Failed to update statistic data: ", err)
		return false
	}

	return true
}

// 是否存在之前的最新累计数据
func (dao *StatisticDataDao) ExistLastCumData(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	err := dao.db.Table(stsd.GetTableName()).Order("mark_date DESC").Where("equipment_id = ? AND data_source_type = ?  AND statistic_code = ? AND mark_date < ?", stsd.EquipmentId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		return dataTem, false
	}

	return dataTem, true
}
func (dao *StatisticDataDao) ExistLastAlarmCumData(stsd model.StatisticData) (model.StatisticData, bool) {
	dataTem := model.StatisticData{}

	err := dao.db.Table(stsd.GetTableName()).Order("mark_date DESC").Where("equipment_id = ? AND data_source_type = ?  AND statistic_code = ? AND statistic_alarm_type = ? AND mark_date < ?", stsd.EquipmentId, stsd.DataSourceType, stsd.StatisticCode, stsd.StatisticAlarmType, stsd.MarkDate).First(&dataTem).Error
	if err != nil || dataTem.ID == 0 {
		return dataTem, false
	}

	return dataTem, true
}

// 是否存在当前标记数据
func (dao *StatisticDataDao) FindBySiteId(stsd model.StatisticData) ([]model.StatisticData, bool) {
	dataTem := []model.StatisticData{}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where("site_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.SiteId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).Find(&dataTem).Error
	if err != nil || len(dataTem) == 0 {
		// log.Println("Failed to find dateMark list by Site: ", err)
		return dataTem, false
	}

	return dataTem, true
}

// 是否存在当前标记数据
func (dao *StatisticDataDao) FindByCorpId(stsd model.StatisticData) ([]model.StatisticData, bool) {
	dataTem := []model.StatisticData{}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where("corp_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.CorpId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).Find(&dataTem).Error
	if err != nil || len(dataTem) == 0 {
		// log.Println("Failed to find dateMark list by Corp: ", err)
		return dataTem, false
	}

	return dataTem, true
}
func (dao *StatisticDataDao) FindAlarmByCorpId(stsd model.StatisticData) ([]model.StatisticData, bool) {
	dataTem := []model.StatisticData{}

	// 匹配标记的数据
	err := dao.db.Table(stsd.GetTableName()).Where("corp_id = ? AND data_source_type = ? AND statistic_code = ? AND statistic_alarm_type = ? AND mark_date = ?", stsd.CorpId, stsd.DataSourceType, stsd.StatisticCode, stsd.StatisticAlarmType, stsd.MarkDate).Find(&dataTem).Error
	if err != nil || len(dataTem) == 0 {
		// log.Println("Failed to find dateMark list by Corp: ", err)
		return dataTem, false
	}

	return dataTem, true
}

// 是否存在, 仍在库中, 未被删除数据
func (dao *StatisticDataDao) FindByCorpIdExistEquipment(stsd model.StatisticData) (float64, bool) {
	var dataTem sql.NullFloat64

	tableName := stsd.GetTableName()
	// 匹配标记的数据, 查询存在的数据
	err := dao.db.Table(tableName).
		Select("sum(statistic_value)").
		Where(tableName+".corp_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", stsd.CorpId, stsd.DataSourceType, stsd.StatisticCode, stsd.MarkDate).
		Joins(`right JOIN "equipment" ON "equipment".id = "` + tableName + `".equipment_id `).
		Find(&dataTem).Error
	if err != nil {
		log.Printf("Failed to sum existing equipment statistic data table=%s corpId=%d code=%s markDate=%s: %v", tableName, stsd.CorpId, stsd.StatisticCode, stsd.MarkDate, err)
		return 0, false
	}
	if dataTem.Valid {
		return dataTem.Float64, true
	}

	// log.Println("Failed to find dateMark list by Corp: ", err)
	return 0, false
}

func (dao *StatisticDataDao) FindEquipmentDailyCumulative(equipmentIDs []int, codes, markDates []string) ([]model.StatisticData, error) {
	data := []model.StatisticData{}
	if len(equipmentIDs) == 0 || len(codes) == 0 || len(markDates) == 0 {
		return data, nil
	}
	tableName := (model.StatisticData{TargetType: 0, StatisticDateType: "daily", StatisticValueType: 0}).GetTableName()
	err := dao.db.Table(tableName).
		Where("equipment_id IN ? AND data_source_type = ? AND statistic_code IN ? AND mark_date IN ?", equipmentIDs, 3, codes, markDates).
		Order("equipment_id, mark_date, statistic_code, end_statistic_time DESC").
		Find(&data).Error
	return data, err
}

// SaveEquipmentDailyCumulative updates first so existing deployments do not
// require an immediate schema migration before on-demand backfill can run.
// Callers serialize this operation with LockStatisticDailyData.
func (dao *StatisticDataDao) SaveEquipmentDailyCumulative(data model.StatisticData) error {
	data.TargetType = 0
	data.StatisticDateType = "daily"
	data.StatisticValueType = 0
	tableName := data.GetTableName()
	updates := map[string]interface{}{
		"site_id":              data.SiteId,
		"corp_id":              data.CorpId,
		"statistic_name":       data.StatisticName,
		"start_statistic_time": data.StartStatisticTime,
		"end_statistic_time":   data.EndStatisticTime,
		"start_value":          data.StartValue,
		"end_value":            data.EndValue,
		"statistic_value":      data.StatisticValue,
		"create_time":          data.CreateTime,
	}
	var existing model.StatisticData
	query := dao.db.Table(tableName).
		Where("equipment_id = ? AND data_source_type = ? AND statistic_code = ? AND mark_date = ?", data.EquipmentId, data.DataSourceType, data.StatisticCode, data.MarkDate).
		First(&existing)
	if query.Error == nil {
		return dao.db.Table(tableName).Where("id = ?", existing.ID).Updates(updates).Error
	}
	if !errors.Is(query.Error, gorm.ErrRecordNotFound) {
		return query.Error
	}
	data.ID = 0
	return dao.db.Table(tableName).Create(&data).Error
}

// 查询统计数据
func (dao *StatisticDataDao) FindStatisticData(stsd model.StatisticData, offset, limit, startTime, endTime int) ([]model.StatisticData, int64, bool) {
	dataTem := []model.StatisticData{}

	sqlStr := baseStatisticDataSql(stsd)

	err := dao.db.Table(stsd.GetTableName()).Offset(offset).Limit(limit).Order("end_statistic_time DESC").
		Where(sqlStr+" AND end_statistic_time >= ? AND end_statistic_time <= ?", startTime, endTime).Find(&dataTem).Error
	if err != nil || len(dataTem) == 0 {
		// log.Println("Failed to find dateMark list for user: ", err)
		return dataTem, 0, false
	}

	fillStatisticDataEquipmentTypeNames(dao.db, dataTem)

	var total int64
	dao.db.Table(stsd.GetTableName()).
		Where(sqlStr+" AND end_statistic_time >= ? AND end_statistic_time <= ?", startTime, endTime).Count(&total)

	return dataTem, total, true
}

// 基本查询语句封装
func baseStatisticDataSql(stsd model.StatisticData) string {
	sqlStr := "1=1"

	if stsd.SiteId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND site_id = %d", stsd.SiteId)
	}
	if stsd.EquipmentId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id = %d", stsd.EquipmentId)
	}
	if stsd.CorpId != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND corp_id = %d", stsd.CorpId)
	}
	if len(stsd.StatisticCode) > 0 {
		sqlStr = sqlStr + " AND statistic_code like '%" + stsd.StatisticCode + "%'"
	}
	if stsd.DataSourceType != 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND data_source_type = %d", stsd.DataSourceType)
	}
	if len(stsd.StatisticAlarmType) > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND statistic_alarm_type = '%s'", stsd.StatisticAlarmType)
	}
	if len(stsd.EquipmentTypeName) > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id IN (SELECT id FROM equipment WHERE equipment_type_name LIKE '%%%s%%')", stsd.EquipmentTypeName)
	}

	return sqlStr
}

// 通过站点查询
func (dao *StatisticDataDao) FindStatisticDataBySite(stsd model.StatisticData, offset, limit, startTime, endTime int, siteIds []int) ([]model.StatisticData, int64, bool) {
	dataTem := []model.StatisticData{}

	sqlStr := baseStatisticDataSql(stsd)

	err := dao.db.Table(stsd.GetTableName()).Offset(offset).Limit(limit).Order("end_statistic_time DESC").
		Where(sqlStr+" AND site_id in ? AND end_statistic_time >= ? AND end_statistic_time <= ?", siteIds, startTime, endTime).Find(&dataTem).Error
	if err != nil || len(dataTem) == 0 {
		// log.Println("Failed to find dateMark list for user: ", err)
		return dataTem, 0, false
	}

	fillStatisticDataEquipmentTypeNames(dao.db, dataTem)

	var total int64
	dao.db.Table(stsd.GetTableName()).
		Where(sqlStr+" AND site_id in ? AND end_statistic_time >= ? AND end_statistic_time <= ?", siteIds, startTime, endTime).Count(&total)

	return dataTem, total, true
}

// 通过站点查询
func (dao *StatisticDataDao) FindStatisticDataByCorp(stsd model.StatisticData, offset, limit, startTime, endTime int, corpIds []int) ([]model.StatisticData, int64, bool) {
	dataTem := []model.StatisticData{}

	sqlStr := baseStatisticDataSql(stsd)

	err := dao.db.Table(stsd.GetTableName()).Offset(offset).Limit(limit).Order("end_statistic_time DESC").
		Where(sqlStr+" AND corp_id in ? AND end_statistic_time >= ? AND end_statistic_time <= ?", corpIds, startTime, endTime).Find(&dataTem).Error
	if err != nil || len(dataTem) == 0 {
		// log.Println("Failed to find dateMark list for user: ", err)
		return dataTem, 0, false
	}

	fillStatisticDataEquipmentTypeNames(dao.db, dataTem)

	var total int64
	dao.db.Table(stsd.GetTableName()).
		Where(sqlStr+" AND corp_id in ? AND end_statistic_time >= ? AND end_statistic_time <= ?", corpIds, startTime, endTime).Count(&total)

	return dataTem, total, true
}

// fillStatisticDataEquipmentTypeNames 批量填充统计数据的电池类型名称
func fillStatisticDataEquipmentTypeNames(db *gorm.DB, data []model.StatisticData) {
	if len(data) == 0 {
		return
	}
	ids := make([]int, 0, len(data))
	seen := make(map[int]struct{}, len(data))
	for _, item := range data {
		if item.EquipmentId > 0 {
			if _, ok := seen[item.EquipmentId]; !ok {
				ids = append(ids, item.EquipmentId)
				seen[item.EquipmentId] = struct{}{}
			}
		}
	}
	if len(ids) == 0 {
		return
	}

	type equipTypeRow struct {
		ID   int    `gorm:"column:id"`
		Name string `gorm:"column:equipment_type_name"`
	}
	var rows []equipTypeRow
	db.Table("equipment").Select("id, equipment_type_name").Where("id IN ?", ids).Scan(&rows)

	typeMap := make(map[int]string, len(rows))
	for _, row := range rows {
		typeMap[row.ID] = row.Name
	}
	for i := range data {
		data[i].EquipmentTypeName = typeMap[data[i].EquipmentId]
	}
}
