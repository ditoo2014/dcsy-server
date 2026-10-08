package dao

import (
	"alarm/internal/common/model"
	"alarm/util/gormutil"
	"errors"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type GeneralTriggerDao struct {
	db *gorm.DB
}

func NewGeneralTriggerDao(ctx *gin.Context) *GeneralTriggerDao {
	return &GeneralTriggerDao{db: gormutil.GetGormDB(ctx, "")}
}

// 是否存在报警代码
func (dao *GeneralTriggerDao) ExistTriggerName(tc model.GeneralTriggerConfig) (model.GeneralTriggerConfig, bool) {
	var tcTem = model.GeneralTriggerConfig{}

	err := dao.db.Table("general_trigger_config").Where("trigger_name = ? and trigger_type = ?", tc.Code, tc.TriggerType).Find(&tcTem).Error
	if err != nil || tcTem.ID == 0 {
		log.Printf("DO NOT EXIST generaltrigger with Code info: %v", err)
		return tcTem, false
	}

	return tcTem, true
}

// 创建
func (dao *GeneralTriggerDao) Create(gtc model.GeneralTriggerConfig) bool {

	tx := dao.db.Begin()

	// 使用 GORM 保存
	if err := tx.Create(&gtc).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to create trigger: %v", err)
		return false
	}

	// 保存触发匹配项 - 多个设备的触发匹配判断
	if len(gtc.Marktriggers) > 0 {
		for _, v := range gtc.Marktriggers {
			v.TriggerId = gtc.ID
			if err := tx.Create(&v).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create marktrigger data: %v", err)
				return false
			}
		}
	}

	// 保存触发逻辑
	if len(gtc.Logics) > 0 {
		for _, v := range gtc.Logics {
			v.TriggerId = gtc.ID
			if err := tx.Create(&v).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create logic: %v", err)
				return false
			}
		}
	}

	// 保存字符型匹配触发配置
	if len(gtc.StrMatchs) > 0 {
		for _, v := range gtc.StrMatchs {
			v.TriggerId = gtc.ID
			if err := tx.Create(&v).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create str_match: %v", err)
				return false
			}
		}
	}

	// 报警类型触发器
	switch gtc.TriggerType {
	case 0:
		// 报警触发器

		// 添加报警配置
		if len(gtc.AlarmUser.AlarmType) > 0 {
			// 报警配置存在
			gtc.AlarmUser.TriggerId = gtc.ID
			if err := tx.Create(&gtc.AlarmUser).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create AlarmUser: %v", err)
				return false
			}
		}

	case 1:
		// 统计触发器

		// 添加统计配置
		if len(gtc.StatisticConfig.StatisticType) != 0 {

			gtc.StatisticConfig.TriggerId = gtc.ID
			if err := tx.Create(&gtc.StatisticConfig).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create StatisticConfig: %v", err)
				return false
			}
		}

	case 2:
		// 添加报警配置
		if len(gtc.AlarmUser.AlarmType) > 0 {
			// 报警配置存在
			gtc.AlarmUser.TriggerId = gtc.ID
			if err := tx.Create(&gtc.AlarmUser).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create AlarmUser: %v", err)
				return false
			}
		}

		// 添加统计配置
		if len(gtc.StatisticConfig.StatisticType) != 0 {

			gtc.StatisticConfig.TriggerId = gtc.ID
			if err := tx.Create(&gtc.StatisticConfig).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create StatisticConfig: %v", err)
				return false
			}
		}

	case 3:
		// 推送到: pushgateway + prometheus + altermanager
		// 添加报警配置
		if len(gtc.AlarmUser.AlarmType) > 0 {
			// 报警配置存在
			gtc.AlarmUser.TriggerId = gtc.ID
			if err := tx.Create(&gtc.AlarmUser).Error; err != nil {
				// 回滚事务
				tx.Rollback()
				log.Printf("Failed to create AlarmUser: %v", err)
				return false
			}
		}

		// 添加统计配置
		if len(gtc.StatisticConfig.StatisticType) != 0 {

			gtc.StatisticConfig.TriggerId = gtc.ID
			if err := tx.Create(&gtc.StatisticConfig).Error; err != nil {
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

// 是否存在id
func (dao *GeneralTriggerDao) ExistId(tc model.GeneralTriggerConfig) (model.GeneralTriggerConfig, bool) {

	err := dao.db.First(&tc).Error
	if err != nil {
		log.Printf("DO NOT EXIST trigger with id info: %v", err)
		return tc, false
	}

	return tc, true
}

// 更新时候 - 是否存在报警代码
func (dao *GeneralTriggerDao) UpdateExistEquipmentAlarmCode(tc model.GeneralTriggerConfig) (model.GeneralTriggerConfig, bool) {
	var tcTem = model.GeneralTriggerConfig{}

	err := dao.db.Table("general_trigger_config").Where(" code = ? and trigger_type = ? AND id <> ?", tc.Code, tc.TriggerType, tc.ID).Find(&tcTem).Error
	if err != nil || tcTem.ID == 0 {
		log.Printf("DO NOT EXIST trigger with equipmentId, Code info: %v", err)
		return tcTem, false
	}

	return tcTem, true
}

// 更新
func (dao *GeneralTriggerDao) Update(tc model.GeneralTriggerConfig) bool {

	tx := dao.db.Begin()

	// 使用 GORM 更新
	if err := tx.Model(&tc).Where("id = ?", tc.ID).Select("*").Updates(&tc).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to update trigger: %v", err)
		return false
	}

	// 触发匹配项 - 多个设备的触发匹配判断
	// 此处不修改, 只有定时任务定时刷新, 获取最新的设备数据, 进行实时更新

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
func (dao *GeneralTriggerDao) Delete(tc model.GeneralTriggerConfig) bool {

	tx := dao.db.Begin()

	// 使用 GORM 删除
	if err := tx.Where("id = ?", tc.ID).Delete(&tc).Error; err != nil {
		tx.Rollback()
		log.Printf("Failed to delete trigger: %v", err)
		return false
	}

	// 删除触发匹配项 - 多个设备的触发匹配判断
	if err := tx.Model(&model.GeneralMarktrigger{}).Where("trigger_id = ?", tc.ID).Delete(&model.GeneralMarktrigger{}).Error; err != nil {
		// 回滚事务
		tx.Rollback()
		log.Printf("Failed to delete triggermarks: %v", err)
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

// 查询触发器配置全部信息 - 除了绑定的设备外
func (dao *GeneralTriggerDao) FindTriggerLogics(tc model.GeneralTriggerConfig) ([]model.GeneralTriggerConfig, error) {
	var tcs []model.GeneralTriggerConfig

	err := dao.db.Preload("Marktriggers").
		Preload("StrMatchs").
		Preload("Logics").
		Preload("AlarmUser").
		Preload("StatisticConfig").Select(model.GENERAL_TRIGGER_PARAMS).Find(&tcs).Error
	if err != nil {
		log.Printf("Failed to get generalTriggers with logic info: %v", err)
		return nil, err
	}

	return tcs, nil
}

// 查询全部 - bloom过滤器数据
func (dao *GeneralTriggerDao) Find(tc model.GeneralTriggerConfig) ([]model.GeneralTriggerConfig, bool) {
	var tcTems = []model.GeneralTriggerConfig{}

	err := dao.db.Model(&model.GeneralTriggerConfig{}).
		Preload("Marktriggers").
		Preload("Logics").
		Preload("StrMatchs").
		Preload("AlarmUser").
		Preload("StatisticConfig").
		Find(&tcTems).Error
	if err != nil || len(tcTems) == 0 {
		// log.Printf("DO NOT EXIST generalTrigger info: %v", err)
		return tcTems, false
	}

	return tcTems, true
}

// FindByMarktrigger loads only the general triggers bound to one concrete
// gateway/data-point pair. The old Find method remains available for cache
// refresh and management operations, while runtime evaluation avoids reading
// every trigger and every mark for each incoming value.
func (dao *GeneralTriggerDao) FindByMarktrigger(marktrigger string) ([]model.GeneralTriggerConfig, bool) {
	marktrigger = strings.TrimSpace(marktrigger)
	if dao == nil || dao.db == nil || marktrigger == "" {
		return nil, false
	}

	var configs []model.GeneralTriggerConfig
	query := dao.db.Model(&model.GeneralTriggerConfig{}).
		Where("id IN (?)", dao.db.Table("general_marktrigger").Select("trigger_id").Where("marktrigger = ?", marktrigger)).
		Preload("Marktriggers", "marktrigger = ?", marktrigger).
		Preload("Logics").
		Preload("StrMatchs").
		Preload("AlarmUser").
		Preload("StatisticConfig")
	if err := query.Find(&configs).Error; err != nil || len(configs) == 0 {
		return configs, false
	}
	return configs, true
}

func (dao *GeneralTriggerDao) DeleteMarktriggers(tc model.GeneralTriggerConfig) bool {

	// 删除触发匹配项 - 多个设备的触发匹配判断
	if err := dao.db.Model(&model.GeneralMarktrigger{}).Where("trigger_id = ?", tc.ID).Delete(&model.GeneralMarktrigger{}).Error; err != nil {
		// 回滚事务
		log.Printf("Failed to single delete triggermarks: %v", err)
		return false
	}

	return true
}

// 单独删除一个触发器
func (dao *GeneralTriggerDao) DeleteMarkTrigger(gm model.GeneralMarktrigger) bool {

	err := dao.db.Model(&gm).Delete(&gm).Error
	if err != nil {
		log.Printf("Delete one marktrigger info: %v", err)
		return false
	}

	return true
}

// 单独创建一个触发器
func (dao *GeneralTriggerDao) CreateMarkTrigger(gm ...model.GeneralMarktrigger) bool {
	if len(gm) == 0 {
		log.Println("Create one marktrigger info: marktriggers is empty")
		return false
	}

	err := dao.db.Create(&gm).Error
	if err != nil {
		log.Printf("Create one marktrigger info: %v", err)
		return false
	}

	return true
}

// 单独更新一个触发器
func (dao *GeneralTriggerDao) UpdateMarkTrigger(gm model.GeneralMarktrigger) bool {

	err := dao.db.Model(&gm).Updates(&gm).Error
	if err != nil {
		log.Printf("Update one marktrigger info: %v", err)
		return false
	}

	return true
}

// 查询全部 - 触发标记匹配的数据
func (dao *GeneralTriggerDao) FindMarkTriggers(gmts model.GeneralMarktrigger) ([]model.GeneralMarktrigger, bool) {
	var tcTems = []model.GeneralMarktrigger{}

	err := dao.db.Find(&tcTems).Error
	if err != nil || len(tcTems) == 0 {
		log.Printf("DO NOT EXIST generalTrigger info: %v", err)
		return tcTems, false
	}

	return tcTems, true
}

// 报警过程数据更新 [ALARM]
func (dao *GeneralTriggerDao) SetRecordTimer(tc model.GeneralMarktrigger, record, timer int64) (err error) {

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
func (dao *GeneralTriggerDao) SetSustainTimer(tc model.GeneralMarktrigger, record int64) (err error) {

	if record < 0 {
		return
	}

	/** 封装更新参数 -- 值 >=0 就更新 */
	param := make(map[string]interface{})
	if record > 0 {
		param["sustain_record"] = record
	}
	if int64(0) == record {
		param["sustain_record"] = 0
	}
	/* 完成格式转换 */

	err = dao.db.Model(&tc).Select("*").Updates(param).Error
	if nil != err {
		err = errors.New("[TriggerDao] 系统错误")
	}

	return
}
