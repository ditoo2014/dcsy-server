package model

type SystemLog struct {
	Id             int    `json:"id" gorm:"primaryKey;column:id;size:32"`
	UserId         int    `json:"userId" gorm:"column:user_id;type:integer;not null;index:idx_syslog_userid"` // 用户id
	Username       string `json:"username" gorm:"type:varchar;not null"`                                      // 用户登录号
	TrueName       string `json:"trueName" gorm:"type:varchar;not null"`                                      // 用户名
	LogContent     string `json:"logContent" gorm:"type:text;not null"`                                       // 日志内容
	LogTime        int64  `json:"logTime" gorm:"type:bigint;not null"`                                        // 日志记录时间
	IpAddress      string `json:"ipAddress" gorm:"type:varchar;not null;default:'-'"`                         // IP地址
	OperateDetails string `json:"operateDetails" gorm:"type:text"`                                            // 操作详情
	PlatId         int    `json:"platId" gorm:"type:integer;not null;index:idx_syslog_plat"`                  // 平台id 1: 业务云 2: 采集云
}

// TableName 指定表名为 system_log
func (SystemLog) TableName() string {
	return "system_log"
}
