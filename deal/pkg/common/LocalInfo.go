package common

import (
	"log"
	"net"
)

const (
	DefaultLocalAddr = "127.0.0.1"
)

// GetLocalAddr 获取本机IP地址
// 返回值: 本机IP地址字符串，如果获取失败则返回默认的127.0.0.1
func GetLocalAddr() string {
	// 获取所有网络接口地址
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		// 获取网络接口失败时，记录错误日志并返回默认地址
		log.Println("[TOLINK] 获取网络接口失败", err)
		return DefaultLocalAddr
	}
	var localIP string
	// 遍历所有网络接口地址
	for _, addr := range addrs {
		// 检查是否为IPNet类型且非回环地址
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			// 检查是否为IPv4地址
			if ipnet.IP.To4() != nil {
				// 找到第一个符合条件的IPv4地址并保存
				localIP = ipnet.IP.String()
				break
			}
		}
	}
	// 如果没有找到合适的IP地址，记录错误日志并返回默认地址
	if localIP == "" {
		log.Println("[TOLINK] 无法获取本机IP地址")
		return DefaultLocalAddr
	}

	return localIP
}

// GenFreePort 获取一个空闲的端口; 端口避免写死, 因为要启动多个实例, 测试负载均衡
func GenFreePort(defaultPort int) int {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		log.Println("[TOLINK] 解析TCP地址失败", err)
		return defaultPort // 或其他合适的默认值
	}
	listen, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Println("[TOLINK] 监听TCP端口失败", err)
		return defaultPort // 或其他合适的默认值
	}
	defer listen.Close()
	return listen.Addr().(*net.TCPAddr).Port
}
