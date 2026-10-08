package client

import (
	"dicloud/internal/common/grpc/auth_grpc"
	"dicloud/models/model"
	"encoding/json"

	"github.com/gin-gonic/gin"
)

func SyncToTolink(ctx *gin.Context, userInfo *model.User) (*auth_grpc.SyncLoginResponse, error) {

	x := baseUserDataFormat(userInfo)

	return getTolinkClient().SyncLoginToTolink(ctx, x)
}

func SyncToBusiness(ctx *gin.Context, userInfo *model.User) (*auth_grpc.SyncLoginResponse, error) {

	x := baseUserDataFormat(userInfo)

	return getBusinessClient().SyncLoginToTolink(ctx, x)
}

func SyncToDynalt(ctx *gin.Context, userInfo *model.User) (*auth_grpc.SyncLoginResponse, error) {

	x := baseUserDataFormat(userInfo)

	return getDynaltClient().SyncLoginToTolink(ctx, x)
}

// 用户数据转换 - 本地数据 -> grpc数据
func baseUserDataFormat(ui *model.User) *auth_grpc.UserInfo {

	// 转换数据
	var srlRes = false
	strBytes, err := json.Marshal(ui)
	var e = auth_grpc.UserInfo{}
	if err == nil {
		err := json.Unmarshal(strBytes, &e)
		if err == nil {
			srlRes = true
			e.CreateTimestamp = ui.CreateTime.Unix()
		}
	}

	if !srlRes {
		// 通过json序列化, 反序列化失败
		e = auth_grpc.UserInfo{
			Username:        ui.Username,
			Password:        ui.Password,
			Truename:        ui.TrueName,
			Phone:           ui.Phone,
			Email:           ui.Email,
			CreateTimestamp: ui.CreateTime.Unix(),
			TotalLogin:      int64(ui.TotalLogin),
			Token:           ui.Token,
			LoginIp:         ui.LoginIp,
			LastIp:          ui.LastIp,
			IsAdmin:         int64(ui.Isadmin),
			ClientId:        ui.ClientId,
			ParentId:        int64(ui.ParentId),
			PlatName:        ui.PlatName,
			PlatType:        ui.PlatType,
			PlatId:          int64(ui.PlatId),
			Baned:           int64(ui.Baned),
			CorpIds:         ui.CorpIds,
			Renter:          ui.Renter,
		}
	}

	return &e
}
