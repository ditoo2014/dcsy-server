package grpcutil

import (
	"log"
	"sync"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
使用场景限制：
多个goroutine频繁读取同一组键值对（读多写少）；
不同goroutine各自操作不同的key，冲突较少；
不需要遍历所有元素，或可以接受非实时一致性遍历；
避免在循环中频繁修改同一个key。
*/
var multiGrpcConnMap = sync.Map{}

// 初始化tolink客户端 - 进行远程调用
func InitConn(serviceName string) *grpc.ClientConn {

	var err error
	// 基本配置信息获取
	port := viper.GetString("server." + serviceName + ".port")
	if len(port) == 0 {
		log.Println("[GRPC-CONN] port is null")
		return nil
	}
	addr := "127.0.0.1:" + port
	host := viper.GetString("server." + serviceName + ".host")
	if len(host) > 0 {
		addr = host + ":" + port
	}

	// 测试连接 - 保证都使用这个初始化链接
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[GRPC-CONN] create gRPC connect, service [%s] addr [%s] fail; 连接失败: %s \n", serviceName, addr, err)
		return conn
	}

	log.Printf("[GRPC-CONN] create gRPC connect, service [%s] addr [%s] success \n", serviceName, addr)
	// 每次调用Store可能触发内部状态转换，高频写同个key反而不如加锁map
	multiGrpcConnMap.Store(serviceName, conn)
	return conn
}

// 通过服务名称, 获取gRPC远程链接
func GetConn(serviceName string) *grpc.ClientConn {

	if len(serviceName) == 0 {
		log.Println("[GRPC-CONN] undefine service name, get connection fail")
		return nil
	}
	if connTem, res := multiGrpcConnMap.Load(serviceName); connTem != nil && res {
		return connTem.(*grpc.ClientConn)
	} else {
		// 通过Nacos服务注册中心, 初始化服务
		curConn := InitConn(serviceName)
		if curConn == nil {
			log.Println("[GRPC-CONN] initial connection fail")
			return nil
		}
		return curConn
	}
}
