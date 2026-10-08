package grpcserver

import (
	"context"
	"dicloud/internal/common/grpc/tolink_grpc"
	tolinkdao "dicloud/internal/tolink/dao"
	tolinkmodel "dicloud/internal/tolink/model"
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
)

func (TolinkServer) ExistApiKey(ctx context.Context, request *tolink_grpc.Stream) (*tolink_grpc.TolinkStreamResponse, error) {
	log.Printf("[GRPC-TolinkStreamServer] ExistApiKey = %s\n", request.ApiKey)

	// 响应体
	var responseData = &tolink_grpc.TolinkStreamResponse{
		Code:    200,
		Message: fmt.Sprintf("[GRPC-TolinkStreamServer] ExistApiKey Name = %s, StreamCode = %s\n", request.Name, request.StreamCode),
	}

	ginCtx := &gin.Context{}
	if len(request.DbName) > 0 {
		ginCtx.Set("dbName", request.DbName)
	}

	// 初始化
	streamDao := tolinkdao.NewStreamDAO(ginCtx)

	// 查询数据池
	// 验证数据池是否存在
	dbstream, res := streamDao.ExistApiKey(tolinkmodel.Stream{ApiKey: request.ApiKey})
	if !res {
		log.Println("[GRPC-TolinkStreamServer] Generalstrm get markstrm stream is not exist")
		responseData.Code = 10501
		responseData.Message = "不存在的数据"
		return responseData, nil
	}

	responseData.Stream = baseTolinkStreamGRPCDataFormat(dbstream)

	return responseData, nil
}

// Stream数据转换 - 本地数据 -> grpc数据
func baseTolinkStreamGRPCDataFormat(strm tolinkmodel.Stream) *tolink_grpc.Stream {
	data := &tolink_grpc.Stream{
		Id:           int64(strm.ID),
		Admin:        strm.Admin,
		Name:         strm.Name,
		ApiKey:       strm.ApiKey,
		CreateUserId: int64(strm.CreateUserId),
		CorpId:       int64(strm.CorpId),
		StreamCode:   strm.StreamCode,
	}

	for idx := range strm.List {
		data.List = append(data.List, &tolink_grpc.StreamLabelUnit{
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

	return data
}

// Stream数据转换 - grpc数据 -> 本地数据
func baseStreamDataFormat(request *tolink_grpc.Stream) tolinkmodel.Stream {

	// 通过json转换数据
	var srlRes = false
	strBytes, err := json.Marshal(request)
	var strm = tolinkmodel.Stream{}
	if err == nil {
		err = json.Unmarshal(strBytes, &strm)
		if err == nil {
			srlRes = true
		}
	}

	// 序列化转换失败, 手动操作
	if !srlRes {
		// 反序列化失败, 手动填充数据
		strm = tolinkmodel.Stream{
			ID:           uint(request.Id),
			Admin:        request.Admin,
			Name:         request.Name,
			ApiKey:       request.ApiKey,
			CreateUserId: int(request.CreateUserId),
			CorpId:       int(request.CorpId),
			StreamCode:   request.StreamCode,
		}

		for idx := range request.List {
			strm.List = append(strm.List, tolinkmodel.StreamLabelUnit{
				StreamId:    uint(request.List[idx].StreamId),
				LabelId:     request.List[idx].LabelId,
				Name:        request.List[idx].Name,
				GatewayName: request.List[idx].GatewayName,
				GatewayId:   request.List[idx].GatewayId,
				DataName:    request.List[idx].DataName,
				DataType:    request.List[idx].DataType,
				DataId:      request.List[idx].DataId,
				Formula:     request.List[idx].Formula,
				Unformula:   request.List[idx].Unformula,
			})
		}
	}

	return strm
}
