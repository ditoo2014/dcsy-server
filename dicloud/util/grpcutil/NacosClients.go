package grpcutil

import (
	nacosutil "dicloud/util/nacos"
	"log"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var multiNacosGrpcConnMap = sync.Map{}

func GetGrpcConnectionByServiceName(serviceName string) *grpc.ClientConn {
	if len(serviceName) == 0 {
		log.Println("[GRPC-NACOS-CONN] undefine service name, get connection fail")
		return nil
	}
	if connTem, res := multiNacosGrpcConnMap.Load(serviceName); connTem != nil && res {
		return connTem.(*grpc.ClientConn)
	} else {
		// 通过Nacos服务注册中心, 初始化服务
		curConn := initGrpcConnectionByNacosServiceName(serviceName)
		if curConn == nil {
			log.Println("[GRPC-CONN] initial connection fail")
			return nil
		}
		return curConn
	}
}

// 根据Nacos服务名, 初始化GRPC客户端 - 进行远程调用
func initGrpcConnectionByNacosServiceName(serviceName string) *grpc.ClientConn {

	// 基本配置信息获取
	addr, err := nacosutil.GetNacosSVC(serviceName)
	if len(addr) == 0 || err != nil {
		log.Printf("[GRPC-NACOS-CONN] GRPC addr is nil; 连接失败: %s \n", err)
		return nil
	}

	// 测试连接 - 保证都使用这个初始化链接
	curGrpcConn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Printf("[GRPC-NACOS-CONN] create GRPC connect, service [%s] addr [%s] fail; 连接失败: %s \n", serviceName, addr, err)
		return nil
	}

	log.Printf("[GRPC-NACOS-CONN] create GRPC connect, service [%s] addr [%s] success \n", serviceName, addr)
	// 每次调用Store可能触发内部状态转换，高频写同个key反而不如加锁map
	multiNacosGrpcConnMap.Store(serviceName, curGrpcConn)
	return curGrpcConn
}
