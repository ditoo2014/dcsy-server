package dao

import (
	"alarm/internal/business/model"
	"alarm/util/gormutil"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SiteGroupDao struct {
	db *gorm.DB
}

func NewSiteGroupDao(ctx *gin.Context) *SiteGroupDao {
	return &SiteGroupDao{db: gormutil.GetGormDB(ctx, "")}
}

// 添加全部
func (sgd SiteGroupDao) AddSiteGroupArr(sgarr []model.SiteGroup) bool {
	// 开启事务
	tx := sgd.db.Begin()

	// 添加, 或者修改
	for _, v := range sgarr {
		err := tx.Create(&v).Error
		if err != nil {
			log.Println("[SiteGroupDao] add single siteGroup fail.", v)
			tx.Rollback()
			return false
		}
	}

	tx.Commit()
	return true
}

// 根据id删除
func (sgd SiteGroupDao) DeleteSiteGroup(sg model.SiteGroup) bool {

	err := sgd.db.Delete(&sg).Error
	if err != nil {
		log.Println("[SiteGroupDao] delete single siteGroup Fail.")
		return false
	}

	return true
}

func (sgd SiteGroupDao) UpdateSiteGroupArr(sgarr []model.SiteGroup) bool {
	// 开启事务
	tx := sgd.db.Begin()

	// 更新
	for _, v := range sgarr {
		if v.ID == 0 {
			log.Println("[SiteGroupDao] update single siteGroup fail. id is ZERO: ", v)
			tx.Rollback()
			return false
		}
		err := tx.Where("id = ?", v.ID).Updates(&v).Error
		if err != nil {
			log.Println("[SiteGroupDao] update single siteGroup fail.", v)
			tx.Rollback()
			return false
		}
	}

	tx.Commit()
	return true
}

func (sgd SiteGroupDao) FindSiteGroupArr() (datal []model.SiteGroup, res bool) {

	err := sgd.db.Table("site_group").Find(&datal).Error
	if err != nil {
		log.Println("[SiteGroupDao] find fail.")
		return nil, false
	}

	return datal, true
}
