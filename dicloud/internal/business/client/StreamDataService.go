package client

import (
	"dicloud/internal/common/grpc/tolink_grpc"
	tolinkmodel "dicloud/internal/tolink/model"
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
)

func ExistApikey(ctx *gin.Context, strm *tolinkmodel.Stream) (*tolink_grpc.TolinkStreamResponse, error) {

	e := baseStreamDataFormat(strm)

	// 切换数据源
	nameTem, exist := ctx.Get("dbName")
	if exist {
		e.DbName = fmt.Sprint(nameTem)
	}

	return getTolinkClient().ExistApiKey(ctx, e)
}

// Stream数据转换 - 本地数据 -> grpc数据
func baseStreamDataFormat(strm *tolinkmodel.Stream) *tolink_grpc.Stream {

	// 转换数据
	var srlRes = false
	strBytes, err := json.Marshal(strm)
	var e = tolink_grpc.Stream{}
	if err == nil {
		err := json.Unmarshal(strBytes, &e)
		if err == nil {
			srlRes = true
		}
	}

	if !srlRes {
		// 通过json序列化, 反序列化失败
		e = tolink_grpc.Stream{
			Id:           int64(strm.ID),
			Admin:        strm.Admin,
			Name:         strm.Name,
			ApiKey:       strm.ApiKey,
			CreateUserId: int64(strm.CreateUserId),
			CorpId:       int64(strm.CorpId),
			StreamCode:   strm.StreamCode,
		}

		// 切片数据封装
		for idx := range strm.List {
			e.List = append(e.List, &tolink_grpc.StreamLabelUnit{
				StreamId:    int64(strm.List[idx].StreamId),
				LabelId:     strm.List[idx].LabelId,
				Name:        strm.List[idx].Name,
				GatewayName: strm.List[idx].GatewayName,
				GatewayId:   strm.List[idx].GatewayId,
				DataName:    strm.List[idx].DataName,
				DataType:    strm.List[idx].DataType,
				DataId:      strm.List[idx].DataId,
				Formula:     strm.List[idx].Formula,
				Unformula:   strm.List[idx].Unformula,
			})
		}
	}

	return &e
}
