package service

import (
	"alarm/util/opengauss"
)

// 加载所有平台, 统计所有平台数据
func LoadingAllPlat() (dbNames []string) {
	return opengauss.RenterDBNames()
}
