package dao

import (
	"alarm/internal/alarm/model"
	"alarm/util/gormutil"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SmsDao struct {
	db *gorm.DB
}

func NewSmsDao(ctx *gin.Context) *SmsDao {
	return &SmsDao{db: gormutil.GetGormDB(ctx, "")}
}

func (dao SmsDao) GetAllSecret(smss model.SmsSecret) (smsses []model.SmsSecret) {

	err := dao.db.Model(&smss).Find(&smsses).Error
	if err != nil {
		log.Println("[ALARM] SmsDao find first secret fail ", err)
		return smsses
	}

	return smsses
}
