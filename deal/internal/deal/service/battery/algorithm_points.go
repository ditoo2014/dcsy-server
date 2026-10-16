package battery

import "strings"

// algorithmDataID 为非模板解析路径生成英文数据点 ID，并保留通道地址前缀。
func algorithmDataID(adr, key string) string {
	return strings.TrimSpace(adr) + "_" + strings.TrimSpace(key)
}
