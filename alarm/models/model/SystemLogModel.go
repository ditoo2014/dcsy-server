package model

type SystemLog struct {
	Id             int    `json:"id"`
	UserId         int    `json:"userId"`         // 用户id
	Username       string `json:"username"`       // 用户登录号
	TrueName       string `json:"trueName"`       // 用户名
	LogContent     string `json:"logContent"`     // 日志内容
	LogTime        int64  `json:"logTime"`        // 日志记录时间
	IpAddress      string `json:"ipAddress"`      // IP地址
	OperateDetails string `json:"operateDetails"` // 操作详情
	PlatId         int    `json:"platId"`         // 平台id 1: 业务云 2: 采集云
}

// TableName 指定表名为 system_log
func (SystemLog) TableName() string {
	return "system_log"
}
