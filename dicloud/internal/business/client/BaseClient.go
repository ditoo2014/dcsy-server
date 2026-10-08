package client

import (
	"context"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/models/enum"
	"dicloud/util/grpcutil"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/status"
)

// 获取采集云客户端
func getTolinkClient() tolink_grpc.TolinkServiceClient {
	return tolinkClientForConnection(grpcutil.GetConn(enum.GRPC_SERVER_TOLINK))
}

func tolinkClientForConnection(conn *grpc.ClientConn) tolink_grpc.TolinkServiceClient {
	// 配置缺失时连接为空，仍返回可报告错误的客户端，避免查询接口直接崩溃。
	if conn == nil {
		return tolink_grpc.NewTolinkServiceClient(unavailableTolinkConnection{})
	}

	// 初始化客户端 - 通用
	if conn.GetState() != connectivity.Ready {
		log.Println("[GRPC-TolinkConnection] Try Reconnect to TolinkServer")
		conn.Connect()
	}

	// TODO: 可以通过Nacos获取客户端, 不用固定配置IP:PORT

	return tolink_grpc.NewTolinkServiceClient(conn)
}

// 仅作为缺少连接配置的失败返回，不模拟成功或隐藏服务异常。
type unavailableTolinkConnection struct{}

func (unavailableTolinkConnection) Invoke(context.Context, string, interface{}, interface{}, ...grpc.CallOption) error {
	return status.Error(codes.Unavailable, "采集服务连接未初始化，请检查业务云配置中的采集服务地址和端口")
}

func (unavailableTolinkConnection) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unavailable, "采集服务连接未初始化")
}
