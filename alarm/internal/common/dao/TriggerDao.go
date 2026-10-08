package dao

import (
	"alarm/internal/common/model"
	"alarm/util/gormutil"
	"errors"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TriggerDao struct {
	db *gorm.DB
}

func NewTriggerDao(ctx *gin.Context) *TriggerDao {
	return &TriggerDao{db: gormutil.GetGormDB(ctx, "")}
}

// 是否存在id
func (dao *TriggerDao) ExistId(tc model.TriggerConfig) (model.TriggerConfig, bool) {

	err := dao.db.First(&tc).Error
	if err != nil {
		log.Printf("DO NOT EXIST trigger with id info: %v", err)
		return tc, false
	}

	return tc, true
}

// 是否存在报警代码
func (dao *TriggerDao) ExistEquipmentAlarmCode(tc model.TriggerConfig) (model.TriggerConfig, bool) {
	var tcTem = model.TriggerConfig{}

	err := dao.db.Table("trigger_config").Where("equipment_id = ? and code = ? and trigger_type = ?", tc.EquipmentId, tc.Code, tc.TriggerType).Find(&tcTem).Error
	if err != nil || tcTem.ID == 0 {
		log.Printf("DO NOT EXIST trigger with equipmentId, Code info: %v", err)
		return tcTem, false
	}

	return tcTem, true
}

// 更新时候 - 是否存在报警代码
func (dao *TriggerDao) UpdateExistEquipmentAlarmCode(tc model.TriggerConfig) (model.TriggerConfig, bool) {
	var tcTem = model.TriggerConfig{}

	err := dao.db.Table("trigger_config").Where("equipment_id = ? and code = ? and trigger_type = ? AND id <> ?", tc.EquipmentId, tc.Code, tc.TriggerType, tc.ID).Find(&tcTem).Error
	if err != nil || tcTem.ID == 0 {
		log.Printf("DO NOT EXIST trigger with equipmentId, Code info: %v", err)
		return tcTem, false
	}

	return tcTem, true
}

// 触发标记是否存在 [ALARM]
func (dao *TriggerDao) ExistMark(tc model.TriggerConfig) ([]model.TriggerConfig, bool) {
	var tcsTem = []model.TriggerConfig{}

	err := dao.db.Preload("StrMatchs").Preload("Logics").Preload("AlarmUser").Preload("StatisticConfig").Table("trigger_config").Where("marktrigger = ? ", tc.Marktrigger).Find(&tcsTem).Error
	if err != nil || len(tcsTem) < 1 {
		// log.Printf("DO NOT EXIST trigger with marktrigger info: %v", err)
		return tcsTem, false
	}

	return tcsTem, true
}

// 报警过程数据更新 [ALARM]
func (dao *TriggerDao) SetRecordTimer(tc model.TriggerConfig, record, timer int64) (err error) {

	if timer < 0 && record < 0 {
		return
	}

	/** 封装更新参数 -- 值 >=0 就更新 */
	param := make(map[string]interface{})
	if record > 0 {
		param["record"] = record
	}
	if timer > 0 {
		param["timer"] = timer
	}
	if int64(0) == record {
		param["record"] = 0
	}
	if int64(0) == timer {
		param["timer"] = 0
	}
	if len(tc.LastStrMatch) > 0 {
		param["last_str_match"] = tc.LastStrMatch
	}
	/* 完成格式转换 */

	err = dao.db.Model(&tc).Select("*").Updates(param).Error
	if nil != err {
		err = errors.New("[TriggerDao] 系统错误")
	}

	return
}

// 报警过程数据更新 [ALARM]
func (dao *TriggerDao) SetSustainTimer(tc model.TriggerConfig, record int64) (err error) {

	if record < 0 {
		return
	}

	/** 封装更新参数 -- 值 >=0 就更新 */
	param := make(map[string]interface{})
	if record > 0 {
		param["record"] = record
	}
	if int64(0) == record {
		param["record"] = 0
	}
	/* 完成格式转换 */

	err = dao.db.Model(&tc).Select("*").Updates(param).Error
	if nil != err {
		err = errors.New("[TriggerDao] 系统错误")
	}

	return
}

// 创建
func (dao *TriggerDao) Create(tc model.TriggerConfig) bool {

	tx := dao.db.Begin()

	// 使用 GORM 保存
	if err := tx.Create(&tc).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to create trigger: %v", err)
		return false
	}

	// 保存触发逻辑
	if len(tc.Logics) > 0 {
		for _, v := range tc.Logics {
			v.TriggerId = tc.ID
			if err := tx.Create(&v).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create logic: %v", err)
				return false
			}
		}
	}

	// 保存字符型匹配触发配置
	if len(tc.StrMatchs) > 0 {
		for _, v := range tc.StrMatchs {
			v.TriggerId = tc.ID
			if err := tx.Create(&v).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create str_match: %v", err)
				return false
			}
		}
	}

	// 报警类型触发器
	switch tc.TriggerType {
	case 0:
		// 报警触发器

		// 添加报警配置
		if len(tc.AlarmUser.AlarmType) > 0 {
			// 报警配置存在
			tc.AlarmUser.TriggerId = tc.ID
			if err := tx.Create(&tc.AlarmUser).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create AlarmUser: %v", err)
				return false
			}
		}

	case 1:
		// 统计触发器

		// 添加统计配置
		if len(tc.StatisticConfig.StatisticType) != 0 {

			tc.StatisticConfig.TriggerId = tc.ID
			if err := tx.Create(&tc.StatisticConfig).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create StatisticConfig: %v", err)
				return false
			}
		}

	case 2:
		// 添加报警配置
		if len(tc.AlarmUser.AlarmType) > 0 {
			// 报警配置存在
			tc.AlarmUser.TriggerId = tc.ID
			if err := tx.Create(&tc.AlarmUser).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create AlarmUser: %v", err)
				return false
			}
		}

		// 添加统计配置
		if len(tc.StatisticConfig.StatisticType) != 0 {

			tc.StatisticConfig.TriggerId = tc.ID
			if err := tx.Create(&tc.StatisticConfig).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create StatisticConfig: %v", err)
				return false
			}
		}

	case 3:
		// 推送到: pushgateway + prometheus + altermanager
		// 添加报警配置
		if len(tc.AlarmUser.AlarmType) > 0 {
			// 报警配置存在
			tc.AlarmUser.TriggerId = tc.ID
			if err := tx.Create(&tc.AlarmUser).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create AlarmUser: %v", err)
				return false
			}
		}

		// 添加统计配置
		if len(tc.StatisticConfig.StatisticType) != 0 {

			tc.StatisticConfig.TriggerId = tc.ID
			if err := tx.Create(&tc.StatisticConfig).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create StatisticConfig: %v", err)
				return false
			}
		}

	}

	// 提交事务
	tx.Commit()

	return true
}

// 更新
func (dao *TriggerDao) Update(tc model.TriggerConfig) bool {

	tx := dao.db.Begin()

	// 使用 GORM 更新
	if err := tx.Model(&tc).Where("id = ?", tc.ID).Select("*").Updates(&tc).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to update trigger: %v", err)
		return false
	}

	// 更新触发逻辑
	if len(tc.Logics) > 0 {
		for _, v := range tc.Logics {
			if v.ID > 0 {
				v.TriggerId = tc.ID
				if err := tx.Model(&v).Where("id = ? AND trigger_id = ?", v.ID, tc.ID).Select("*").Updates(&v).Error; err != nil {
					// 回滚事务
					tx.Rollback()
					log.Printf("Failed to update logic: %v", err)
					return false
				}
			}
		}
	}

	// 更新字符型匹配配置
	if len(tc.StrMatchs) > 0 {
		for _, v := range tc.StrMatchs {
			if v.ID > 0 {
				v.TriggerId = tc.ID
				if err := tx.Model(&v).Where("id = ? AND trigger_id = ?", v.ID, tc.ID).Select("*").Updates(&v).Error; err != nil {
					// 回滚事务
					tx.Rollback()
					log.Printf("Failed to update str_match: %v", err)
					return false
				}
			}
		}
	}

	// 修改报警配置
	if len(tc.AlarmUser.AlarmType) > 0 {
		// 报警配置存在
		if err := tx.Model(&tc.AlarmUser).Where("trigger_id = ?", tc.ID).Updates(&tc.AlarmUser).Error; err != nil {
			// 回滚事务
			tx.Rollback()
			log.Printf("Failed to update AlarmUser: %v", err)
			return false
		}
	}

	// 修改统计配置
	if len(tc.StatisticConfig.StatisticType) != 0 {
		if err := tx.Model(&tc.StatisticConfig).Where("trigger_id = ?", tc.ID).Updates(&tc.StatisticConfig).Error; err != nil {
			// 回滚事务
			tx.Rollback()
			log.Printf("Failed to update StatisticConfig: %v", err)
			return false
		}
	}

	// 提交事务
	tx.Commit()

	return true
}

// 删除
func (dao *TriggerDao) Delete(tc model.TriggerConfig) bool {

	tx := dao.db.Begin()

	// 使用 GORM 删除
	if err := tx.Where("id = ?", tc.ID).Delete(&tc).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete trigger: %v", err)
		return false
	}

	// 删除触发配置
	if err := tx.Model(&model.RealtimeLogic{}).Where("trigger_id = ?", tc.ID).Delete(&model.RealtimeLogic{}).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to delete logic: %v", err)
		return false
	}

	// 删除字符串匹配配置
	if err := tx.Model(&model.RealtimeStrMatch{}).Where("trigger_id = ?", tc.ID).Delete(&model.RealtimeStrMatch{}).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to delete str_match: %v", err)
		return false
	}

	// 删除报警配置
	if err := tx.Model(&tc.AlarmUser).Where("trigger_id = ?", tc.ID).Delete(&tc.AlarmUser).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to delete AlarmUser: %v", err)
		return false
	}

	// 删除统计配置
	if err := tx.Model(&tc.StatisticConfig).Where("trigger_id = ?", tc.ID).Delete(&tc.StatisticConfig).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to delete StatisticConfig: %v", err)
		return false
	}

	tx.Commit()

	return true
}

// 查询触发器配置全部信息
func (dao *TriggerDao) FindTriggerLogics(tc model.TriggerConfig) ([]model.TriggerConfig, error) {
	var tcs []model.TriggerConfig

	err := dao.db.Preload("StrMatchs").Preload("Logics").Preload("AlarmUser").Preload("StatisticConfig").Select(model.PARAMS).Where("equipment_id = ?", tc.EquipmentId).Find(&tcs).Error
	if err != nil {
		log.Printf("Failed to get Triggers with logic info: %v", err)
		return nil, err
	}

	return tcs, nil
}

// 查询全部
func (dao *TriggerDao) Find(tc model.TriggerConfig) ([]model.TriggerConfig, bool) {
	var tcTems = []model.TriggerConfig{}

	sqlStr := baseTriggerConfigSql(tc)

	err := dao.db.Table("trigger_config").Where(sqlStr).Find(&tcTems).Error
	if err != nil || len(tcTems) == 0 {
		log.Printf("DO NOT EXIST trigger info: %v", err)
		return tcTems, false
	}

	return tcTems, true
}

// 基本查询sql封装
func baseTriggerConfigSql(tc model.TriggerConfig) string {

	sqlStr := "1=1 "
	if tc.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d ", tc.ID)
	}
	if tc.EquipmentId > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND equipment_id = %d ", tc.EquipmentId)
	}
	if len(tc.Marktrigger) > 0 {
		sqlStr = sqlStr + " AND mark_trigger = " + tc.Marktrigger
	}

	return sqlStr
}
