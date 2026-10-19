package dao

import (
	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type EquipmentMaintainRecordDao struct {
	db *gorm.DB
}

func NewEquipmentMaintainRecordDao(ctx *gin.Context) *EquipmentMaintainRecordDao {
	return &EquipmentMaintainRecordDao{db: gormutil.GetGormDB(ctx, "")}
}

// 设备维保记录
func (dao *EquipmentMaintainRecordDao) Create(emr model.EquipmentMaintainRecord) (model.EquipmentMaintainRecord, bool) {

	// 创建记录
	err := dao.db.Create(&emr).Error
	if err != nil {
		log.Println("[EquipmentMaintainRecordDao] create record failed!")
		return emr, false
	}

	return emr, true
}

func (dao *EquipmentMaintainRecordDao) Find(emr model.EquipmentMaintainRecord, offset, limit, startTime, endTime int) ([]model.EquipmentMaintainRecord, bool, int64) {
	rows, total, err := dao.FindScoped(MaintainRecordFilter{
		EquipmentID: emr.EquipmentId, EquipmentTypeName: emr.EquipmentTypeName,
		StartTime: int64(startTime), EndTime: int64(endTime), IsAdmin: true,
	}, offset, limit)
	result := make([]model.EquipmentMaintainRecord, 0, len(rows))
	for _, row := range rows {
		record := row.EquipmentMaintainRecord
		record.EquipmentTypeName = row.EquipmentTypeName
		result = append(result, record)
	}
	return result, err == nil, total
}

// MaintainRecordFilter 将筛选条件与服务端计算的部门权限范围统一传给查询。
// IsAdmin、CorpIDs 不从请求体绑定，避免客户端扩大自己的数据权限。
type MaintainRecordFilter struct {
	EquipmentID       int
	EquipmentCode     string
	EquipmentTypeName string
	MinimumCount      int
	StartTime         int64
	EndTime           int64
	IsAdmin           bool
	CorpIDs           []int
}

// MaintainRecordRow 保留原返回字段，并附带导出和列表所需的设备、站点信息。
type MaintainRecordRow struct {
	model.EquipmentMaintainRecord
	EquipmentCode     string `json:"equipmentCode"`
	EquipmentName     string `json:"equipmentName"`
	EquipmentTypeName string `json:"equipmentTypeName"`
	SiteSerialNumber  string `json:"siteSerialNumber"`
	// 直接返回当前状态，避免前端逐条补查设备；NULL 保留为未知状态。
	PositionStatus *int `json:"positionStatus"`
}

// maintainRecordQuery 同时供列表和导出使用；所有外部筛选值均通过占位符传递。
func (dao *EquipmentMaintainRecordDao) maintainRecordQuery(filter MaintainRecordFilter) *gorm.DB {
	query := dao.db.Table("public.equipment_maintain_record AS r").
		Joins("JOIN public.equipment AS e ON e.id = r.equipment_id").
		Joins("LEFT JOIN public.site AS s ON s.id = e.site_id").
		Where("r.create_time >= ? AND r.create_time <= ?", filter.StartTime, filter.EndTime)
	if !filter.IsAdmin {
		if len(filter.CorpIDs) == 0 {
			query = query.Where("1 = 0")
		} else {
			// 与设备列表保持一致：已分配部门的普通用户同时可以看到未分配部门的设备。
			query = query.Where("e.corp_id IN ?", filter.CorpIDs)
		}
	}
	if filter.EquipmentID > 0 {
		query = query.Where("r.equipment_id = ?", filter.EquipmentID)
	}
	if code := strings.TrimSpace(filter.EquipmentCode); code != "" {
		// 电池 ID 优先使用编码，旧设备没有有效编码时才使用名称。
		query = query.Where("COALESCE(NULLIF(NULLIF(TRIM(e.equipment_code), ''), '-'), e.equipment_name) LIKE ?", "%"+code+"%")
	}
	if name := strings.TrimSpace(filter.EquipmentTypeName); name != "" {
		query = query.Where("e.equipment_type_name LIKE ?", "%"+name+"%")
	}
	if filter.MinimumCount > 0 {
		// 按当前时间范围及同一组设备筛选条件计算次数，再筛选记录，
		// 不能按分页后的结果计数，也不能让列表和导出使用不同口径。
		counts := dao.db.Table("public.equipment_maintain_record AS counted").
			Joins("JOIN public.equipment AS counted_e ON counted_e.id = counted.equipment_id").
			Select("counted.equipment_id").
			Where("counted.create_time >= ? AND counted.create_time <= ?", filter.StartTime, filter.EndTime).
			Group("counted.equipment_id").Having("COUNT(*) >= ?", filter.MinimumCount)
		if !filter.IsAdmin {
			// 权限范围必须参与次数统计，避免把无权设备的记录算入次数。
			if len(filter.CorpIDs) == 0 {
				query = query.Where("1 = 0")
			} else {
				counts = counts.Where("counted_e.corp_id IN ?", filter.CorpIDs)
			}
		}
		if code := strings.TrimSpace(filter.EquipmentCode); code != "" {
			counts = counts.Where("COALESCE(NULLIF(NULLIF(TRIM(counted_e.equipment_code), ''), '-'), counted_e.equipment_name) LIKE ?", "%"+code+"%")
		}
		if name := strings.TrimSpace(filter.EquipmentTypeName); name != "" {
			counts = counts.Where("counted_e.equipment_type_name LIKE ?", "%"+name+"%")
		}
		query = query.Where("r.equipment_id IN (?)", counts)
	}
	return query
}

// CountEquipmentScoped 按相同权限和筛选条件统计电池数，不受当前页大小影响。
func (dao *EquipmentMaintainRecordDao) CountEquipmentScoped(filter MaintainRecordFilter) (int64, error) {
	var count int64
	err := dao.maintainRecordQuery(filter).Distinct("r.equipment_id").Count(&count).Error
	return count, err
}

// FindScoped 查询指定权限内的记录。limit 为 0 时用于完整导出，不截断为固定条数。
func (dao *EquipmentMaintainRecordDao) FindScoped(filter MaintainRecordFilter, offset, limit int) ([]MaintainRecordRow, int64, error) {
	rows := make([]MaintainRecordRow, 0)
	var total int64
	if err := dao.maintainRecordQuery(filter).Count(&total).Error; err != nil {
		return rows, 0, err
	}
	if total == 0 {
		return rows, 0, nil
	}
	query := dao.maintainRecordQuery(filter).
		Select("r.*, e.equipment_code, e.equipment_name, e.equipment_type_name, e.position_status, s.site_serial_number").
		Order("r.create_time DESC, r.id DESC")
	if limit > 0 {
		query = query.Offset(offset).Limit(limit)
	}
	if err := query.Scan(&rows).Error; err != nil {
		return rows, total, err
	}
	return rows, total, nil
}

// fillMaintainRecordEquipmentTypeNames 批量填充维保记录的电池类型名称
func fillMaintainRecordEquipmentTypeNames(db *gorm.DB, data []model.EquipmentMaintainRecord) {
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

// 创建设备
func (dao *EquipmentMaintainRecordDao) ImportEquipmentMaintainRecords(eqs []model.EquipmentMaintainRecord) ([]model.EquipmentMaintainRecord, bool) {

	err := dao.db.Create(&eqs).Error
	if err != nil {
		log.Println("Failed to create EquipmentMaintainRecord list: ", err)
		return eqs, false
	}

	return eqs, true
}

// CountExactRecord 只按稳定业务字段识别历史重复记录；不更新也不覆盖原记录。
// 运维时间、设备和故障描述同时相同才视为可靠重复，避免误删不同批次的记录。
func (dao *EquipmentMaintainRecordDao) CountExactRecord(record model.EquipmentMaintainRecord) (int64, error) {
	var count int64
	err := dao.db.Model(&model.EquipmentMaintainRecord{}).
		Where("equipment_id = ? AND create_time = ?", record.EquipmentId, record.CreateTime).
		// 故障描述与导出一致：新记录取标题，旧记录空标题或占位符取内容。
		Where("COALESCE(NULLIF(NULLIF(TRIM(maintain_title), ''), '-'), maintain_content) = ?", record.MaintainTitle).
		Count(&count).Error
	return count, err
}
