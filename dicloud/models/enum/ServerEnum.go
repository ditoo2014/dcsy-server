package enum

// 服务基本信息
const (
	SERVER_NAME                     = "server.name"                  // 服务名称
	SERVER_PORT                     = "server.port"                  // 服务端口
	SERVER_LOG                      = "log.filePath"                 // 服务日志
	SERVER_TRUSTED_PROXY            = "127.0.0.1"                    // 信任本机代理
	SERVER_USE_NACOS                = "server.useNacos"              // 是否使用Nacos
	SERVER_USE_SPRING_CLOUD_GATEWAY = "server.useSpringCloudGateway" // 是否使用spring-cloud-gateway
)
