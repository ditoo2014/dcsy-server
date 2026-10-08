package service

import (
	authdao "alarm/internal/auth/dao"
	commonmodel "alarm/internal/common/model"
	"alarm/models/model"
	normalutil "alarm/util/normal"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

// 获取报警用户信息
func getAlarmUsers(ctx *gin.Context, a *commonmodel.AlarmMQModel) []model.User {

	// 获取用户信息
	userIdStrs := strings.Split(a.AlarmUsers, ",")
	if len(userIdStrs) == 0 {
		log.Println("[ALARM] form alarm user fail, empty users, split str")
		return nil
	}
	userIds := normalutil.StringArrToIntArr(userIdStrs)
	if len(userIds) == 0 {
		log.Println("[ALARM] form alarm user fail, empty users, format")
		return nil
	}
	userDao := authdao.NewUserDAO(ctx)
	userDatas, err := userDao.GetUsersByIdArr(userIds)
	if err != nil || len(userDatas) == 0 {
		return nil
	}

	return userDatas
}
