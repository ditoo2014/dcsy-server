package grpcserver

import (
	"dicloud/internal/common/grpc/auth_grpc"
	"dicloud/internal/common/grpc/tolink_grpc"
)

type TolinkServer struct {
	// UnimplementedTolinkServiceServer这个结构体是必须要内嵌进来的
	// 也就是说我们定义的这个结构体对象必须继承UnimplementedTolinkServiceServer。
	// 嵌入之后，我们就已经实现了GRPC这个服务的接口，但是实现之后我们什么都没做，没有写自己的业务逻辑，
	// 我们要重写实现的这个接口里的函数，这样才能提供一个真正的rpc的能力。
	tolink_grpc.UnimplementedTolinkServiceServer
}

type TolinkSyncLoginServer struct {
	// UnimplementedTolinkServiceServer这个结构体是必须要内嵌进来的
	// 也就是说我们定义的这个结构体对象必须继承UnimplementedTolinkServiceServer。
	// 嵌入之后，我们就已经实现了GRPC这个服务的接口，但是实现之后我们什么都没做，没有写自己的业务逻辑，
	// 我们要重写实现的这个接口里的函数，这样才能提供一个真正的rpc的能力。
	auth_grpc.UnimplementedAuthServiceServer
}
