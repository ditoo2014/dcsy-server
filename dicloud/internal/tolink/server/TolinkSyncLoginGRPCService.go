package grpcserver

import (
	"context"
	"dicloud/internal/auth/dao"
	"dicloud/internal/common/grpc/auth_grpc"
	"dicloud/models/model"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func (TolinkSyncLoginServer) SyncToTolink(ctx *context.Context, request *auth_grpc.UserInfo) (*auth_grpc.SyncLoginResponse, error) {
	log.Printf("[GRPC-TolinkSyncLoginServer] username = %s, id = %d\n", request.Username, request.Id)

	// 响应体
	var responseData = &auth_grpc.SyncLoginResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkSyncLoginServer] username = %s, id = %d\n", request.Username, request.Id),
	}

	userInfo := baseUserInfoDataFormat(request)
	{
		ginCtx := &gin.Context{}
		if len(request.DbName) > 0 {
			ginCtx.Set("dbName", request.DbName)
		}

		// 更新或者添加数据
		userDao := dao.NewUserDAO(ginCtx)
		userTem, res := userDao.ExistName(model.User{Username: userInfo.Username})
		if res {
			// 更新登录信息
			userInfo.ID = userTem.ID
			userDao.UpdateLogin(userInfo)
		} else {
			// 创建新用户
			userDao.CreateUser(userInfo)
		}
	}

	return responseData, nil
}

// user数据转换 - grpc数据 -> 本地数据
func baseUserInfoDataFormat(request *auth_grpc.UserInfo) model.User {

	// 通过json转换数据
	var srlRes = false
	strBytes, err := json.Marshal(request)
	var userInfo = model.User{}
	if err == nil {
		err = json.Unmarshal(strBytes, &userInfo)
		if err == nil {
			srlRes = true
		}
	}

	// 序列化转换失败, 手动操作
	if !srlRes {
		// 反序列化失败, 手动填充数据
		userInfo = model.User{
			// ID:           uint(request.Id), // 不根据id处理数据, username是唯一值
			Username:   request.Username,
			Password:   request.Password,
			TrueName:   request.Truename,
			Phone:      request.Phone,
			Email:      request.Email,
			CreateTime: time.Unix(request.CreateTimestamp, 0),
			TotalLogin: int(request.TotalLogin),
			Token:      request.Token,
			LoginIp:    request.LoginIp,
			LastIp:     request.LastIp,
			Isadmin:    int(request.IsAdmin),
			ClientId:   request.ClientId,
			ParentId:   int(request.ParentId),
			PlatName:   request.PlatName,
			PlatType:   request.PlatType,
			PlatId:     int(request.PlatId),
			Baned:      int(request.Baned),
			CorpIds:    request.CorpIds,
			Renter:     request.Renter,
		}
	}

	return userInfo
}
