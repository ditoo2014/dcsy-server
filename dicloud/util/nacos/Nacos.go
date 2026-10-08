package nacosutil

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/clients"
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/common/constant"
	"github.com/nacos-group/nacos-sdk-go/vo"
	"github.com/spf13/viper"
)

// 获取nacos客户端

func InitConfigClient() (iClient naming_client.INamingClient, err error) {

	host := viper.GetString("nacos.server.host")
	port := viper.GetInt("nacos.server.port")
	namespaceId := viper.GetString("nacos.server.namespaceId")
	username := viper.GetString("nacos.server.username")
	password := viper.GetString("nacos.server.password")
	if len(host) == 0 {
		host = "127.0.0.1"
	}
	if port == 0 {
		port = 8848
	}
	// 鉴权信息必须来自配置文件或其对应的环境变量，不在源码中提供固定账号密码。
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("未配置Nacos用户名 nacos.server.username")
	}
	if password == "" {
		return nil, errors.New("未配置Nacos密码 nacos.server.password")
	}

	sv := []constant.ServerConfig{
		{
			IpAddr:      host,         // Nacos的服务地址
			Port:        uint64(port), // Nacos的服务端口
			ContextPath: "/nacos",     // Nacos的ContextPath，默认/nacos，在2.0中不需要设置
			Scheme:      "http",       // Nacos的服务地址前缀，默认http，在2.0中不需要设置
		},
	}

	cc := constant.ClientConfig{
		// Nacos服务中配置好命名空间后, 替换
		NamespaceId:         namespaceId,         // 如果需要支持多namespace，我们可以创建多个client,它们有不同的NamespaceId。当namespace是public时，此处填空字符串。
		TimeoutMs:           30000,               // 请求Nacos服务端的超时时间，默认是10000ms
		Username:            username,            // Nacos服务端的API鉴权Username
		Password:            password,            // Nacos服务端的API鉴权Password
		NotLoadCacheAtStart: true,                // 在启动的时候不读取缓存在CacheDir的service信息
		LogLevel:            "debug",             // 日志默认级别，值必须是：debug,info,warn,error，默认值是info
		LogDir:              "./log/nacos/log",   // 日志存储路径
		CacheDir:            "./log/nacos/cache", // 缓存service信息的目录，默认是当前运行目录
	}

	iClient, err = clients.NewNamingClient(
		vo.NacosClientParam{
			ClientConfig:  &cc,
			ServerConfigs: sv,
		},
	)
	if err != nil {
		return nil, err
	}
	return iClient, nil
}

// 初始化nacos并注册服务到nacos

func InitNacosSVC(ServiceName, Ip, ClusterName, GroupName string, Port int, Weight float64, Metadata map[string]string) error {

	iClient, err := InitConfigClient()
	if err != nil {
		log.Println("[NacosUtil] 初始化Nacos客户端失败了")
		return err
	}
	if iClient == nil {
		log.Println("[NacosUtil] Nacos客户端为空")
		return nil
	}

	// 注册服务到nacos
	success, err := iClient.RegisterInstance(vo.RegisterInstanceParam{
		Ip:          Ip,
		Port:        uint64(Port),
		ServiceName: ServiceName,
		Weight:      Weight,
		Enable:      true,
		Healthy:     true,
		Ephemeral:   true,
		Metadata:    Metadata,
		ClusterName: ClusterName, // 默认值DEFAULT
		GroupName:   GroupName,   // 默认值DEFAULT_GROUP
	})

	if err != nil {
		log.Println("[NacosUtil] register fail ", err)
	}
	if success {
		log.Println("[NacosUtil] register nacos success")
		return nil
	}
	return errors.New("注册inventory服务到nacos失败")
}

// 获取服务信息

func GetNacosSVC(serviceName string) (addr string, err error) {
	iClient, err := InitConfigClient()
	if err != nil {
		return "", err
	}
	if iClient == nil {
		log.Println("[NacosUtil] GetNacosService Nacos客户端为空")
		return "", nil
	}

	clusters := []string{"DEFAULT"}
	groupName := viper.GetString("nacos.server.groupName")
	clusterName := viper.GetString("nacos.server.clusterName")
	if len(groupName) == 0 {
		groupName = "DEFAULT_GROUP"
	}
	if len(clusterName) != 0 {
		clusters = append(clusters, clusterName)
	}

	// SelectInstances 只返回满足这些条件的实例列表：healthy=${HealthyOnly},enable=true 和weight>0
	// instances, err := iClient.SelectInstances(vo.SelectInstancesParam{
	// 	ServiceName: svcName,
	// 	GroupName:   "myGroup",             // 默认值DEFAULT_GROUP
	// 	Clusters:    []string{"myCluster"}, // 默认值DEFAULT
	// 	HealthyOnly: true,
	// })
	// if err != nil {
	// 	log.Println("[NacosUtil] GetNacosService 获取实例失败")
	// }
	// var Addr string
	// // 解析实例列表
	// for _, instance := range instances {
	// 	Addr = fmt.Sprintf("%s:%d", instance.Ip, instance.Port)
	// }

	// SelectOneHealthyInstance将会按加权随机轮询的负载均衡策略返回一个健康的实例
	// 实例必须满足的条件：health=true,enable=true and weight>0
	instance, err := iClient.SelectOneHealthyInstance(
		vo.SelectOneHealthInstanceParam{
			ServiceName: serviceName,
			GroupName:   groupName, // 默认值DEFAULT_GROUP
			Clusters:    clusters,  // 默认值DEFAULT
		})

	if err != nil {
		log.Println("[NacosUtil] GetNacosService instance is null")
		return "", err
	}

	// 基本配置信息获取
	addr = fmt.Sprint(instance.Ip) + ":" + fmt.Sprint(instance.Port)
	return addr, nil

}
