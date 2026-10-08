package client

import (
	"dicloud/internal/common/grpc/auth_grpc"
	"dicloud/models/enum"
	"dicloud/util/grpcutil"
	"log"

	"google.golang.org/grpc/connectivity"
)

// 获取采集云登录服务客户端
func getTolinkClient() auth_grpc.AuthServiceClient {

	// 初始化客户端 - 通用
	if grpcutil.GetConn(enum.GRPC_SERVER_TOLINK).GetState() != connectivity.Ready {
		log.Println("[GRPC-TolinkConnection] Try Reconnect to TolinkServer")
		grpcutil.GetConn(enum.GRPC_SERVER_TOLINK).Connect()
	}

	// TODO: 可以通过Nacos获取客户端, 不用固定配置IP:PORT

	return auth_grpc.NewAuthServiceClient(grpcutil.GetConn(enum.GRPC_SERVER_TOLINK))
}

// 获取业务云登录服务客户端
func getBusinessClient() auth_grpc.AuthServiceClient {

	// 初始化客户端 - 通用
	if grpcutil.GetConn(enum.GRPC_SERVER_BUSINESS).GetState() != connectivity.Ready {
		log.Println("[GRPC-BusinessAuthConnection] Try Reconnect to BusinessAuthServer")
		grpcutil.GetConn(enum.GRPC_SERVER_BUSINESS).Connect()
	}

	// TODO: 可以通过Nacos获取客户端, 不用固定配置IP:PORT

	return auth_grpc.NewAuthServiceClient(grpcutil.GetConn(enum.GRPC_SERVER_BUSINESS))
}

// 获取云组态登录服务客户端
func getDynaltClient() auth_grpc.AuthServiceClient {

	// 初始化客户端 - 通用
	if grpcutil.GetConn(enum.GRPC_SERVER_DYNALT).GetState() != connectivity.Ready {
		log.Println("[GRPC-DynaltConnection] Try Reconnect to DynaltServer")
		grpcutil.GetConn(enum.GRPC_SERVER_DYNALT).Connect()
	}

	// TODO: 可以通过Nacos获取客户端, 不用固定配置IP:PORT

	return auth_grpc.NewAuthServiceClient(grpcutil.GetConn(enum.GRPC_SERVER_DYNALT))
}
