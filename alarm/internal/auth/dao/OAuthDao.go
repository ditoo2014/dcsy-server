package dao

import (
	"alarm/models/model"
	"alarm/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type OAuthDao struct {
	db *gorm.DB
}

func NewOAuthDao(ctx *gin.Context) *OAuthDao {
	return &OAuthDao{db: gormutil.GetGormDB(ctx, "")}
}

func (dao OAuthDao) CreateOAuthClientInfo(oac model.OAuthClientSecret) bool {

	err := dao.db.Model(&oac).Create(&oac).Error
	if err != nil {
		log.Println("[OAuthDao] create client info fail, err: ", err)
		return false
	}

	return true
}

func (dao OAuthDao) UpdateOAuthClientInfo(oac model.OAuthClientSecret) bool {

	err := dao.db.Model(&oac).Updates(&oac).Error
	if err != nil {
		log.Println("[OAuthDao] update client info fail, err: ", err)
		return false
	}

	return true
}

func (dao OAuthDao) ExistOauthClientName(oac model.OAuthClientSecret) (model.OAuthClientSecret, bool) {
	var tem = model.OAuthClientSecret{}

	err := dao.db.Model(&oac).Where("client_name = ?", oac.ClientName).Find(&tem).Error
	if err != nil {
		// log.Println("[OAuthDao] find client info fail, err: ", err)
		return tem, false
	}

	return tem, true
}

func (dao OAuthDao) FindAllOACTemps(oac model.OAuthClientSecret, corpIds []int64, offset, limit int) (resList []model.OAuthClientSecret, total int64, err error) {

	sqlStr := "1=1"
	sqlStr = baseOAuthClientSqlFormat(sqlStr, oac)

	err = dao.db.Model(&oac).Preload("TempLoginList", "1=1 order by id desc limit 10").Offset(offset).Limit(limit).Where(sqlStr+" AND corp_id in ? ", corpIds).Find(&resList).Error
	if err != nil {
		log.Println("[OAuthDao] find client info fail, err: ", err)
		return
	}
	dao.db.Model(&oac).Where(sqlStr+" AND corp_id in ? ", corpIds).Count(&total)

	return
}

func (dao OAuthDao) FindOAuthClient(oac model.OAuthClientSecret) (model.OAuthClientSecret, bool) {
	var tem = model.OAuthClientSecret{}

	sqlStr := "1=1"
	sqlStr = baseOAuthClientSqlFormat(sqlStr, oac)

	err := dao.db.Model(&oac).Where(sqlStr).Find(&tem).Error
	if err != nil {
		return tem, false
	}

	return tem, true
}

func baseOAuthClientSqlFormat(sqlStr string, oac model.OAuthClientSecret) string {

	if oac.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d", oac.ID)
	}

	if len(oac.ClientKey) > 0 {
		sqlStr = sqlStr + " AND client_key = '" + oac.ClientKey + "'"
	}
	if len(oac.ClientSecret) > 0 {
		sqlStr = sqlStr + " AND client_secret = '" + oac.ClientSecret + "'"
	}

	if len(oac.ClientName) > 0 {
		sqlStr = sqlStr + " AND client_name like '%" + oac.ClientName + "%'"
	}

	if len(oac.ManagerIds) > 0 {
		// 根据管理员账户查询 - 不是管理员, 查询不到
		idsStr := fmt.Sprintf("%d", oac.ManagerIds[0])
		for i := 1; i < len(oac.ManagerIds); i++ {
			idsStr = idsStr + fmt.Sprintf(",%d", oac.ManagerIds[i])
		}
		sqlStr = sqlStr + " AND (manager_ids && array[" + idsStr + "]::int[] or manager_ids = '{}')"
	}

	return sqlStr
}

func (dao OAuthDao) CreateTempLoginInfo(oactl model.OAuthTempLoginInfo) (model.OAuthTempLoginInfo, bool) {

	err := dao.db.Model(&oactl).Create(&oactl).Error
	if err != nil {
		log.Println("[OAuthDao] create OAuthTempLogin fail, err: ", err)
		return oactl, false
	}

	return oactl, true
}

func (dao OAuthDao) FindTempLoginInfo(oactl model.OAuthTempLoginInfo) (model.OAuthTempLoginInfo, bool) {

	sqlStr := "1=1"

	baseTempLoginSqlFormat(sqlStr, oactl)

	err := dao.db.Model(&oactl).Create(&oactl).Error
	if err != nil {
		log.Println("[OAuthDao] create OAuthTempLogin fail, err: ", err)
		return oactl, false
	}

	return oactl, true
}

func (dao OAuthDao) UpdateTempLoginInfo(oactl model.OAuthTempLoginInfo) (model.OAuthTempLoginInfo, bool) {

	err := dao.db.Model(&oactl).Updates(&oactl).Error
	if err != nil {
		log.Println("[OAuthDao] update OAuthTempLogin fail, err: ", err)
		return oactl, false
	}

	return oactl, true
}

func baseTempLoginSqlFormat(sqlStr string, oactl model.OAuthTempLoginInfo) string {

	if oactl.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d", oactl.ID)
	}

	if len(oactl.LoginCode) > 0 {
		sqlStr = sqlStr + " AND login_code = '" + oactl.LoginCode + "' "
	}
	if len(oactl.LoginTempToken) > 0 {
		sqlStr = sqlStr + " AND login_temp_token = '" + oactl.LoginTempToken + "' "
	}

	return sqlStr
}
