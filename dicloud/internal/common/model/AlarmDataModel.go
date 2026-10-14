package model

// 报警数据
type AlarmDataRecord struct {
	ID            uint   `json:"id" gorm:"primaryKey,column:id"`
	SiteId        int    `json:"siteId"`                                                                                   // 站点ID
	EquipmentId   int    `json:"equipmentId" gorm:"column:equipment_id;uniqueIndex:uni_alarm_record_lifecycle,priority:1"` // 设备ID
	AlarmStatus   int    `json:"alarmStatus"`                                                                              // 报警状态, 报警/正常
	AlarmCode     string `json:"alarmCode" gorm:"column:alarm_code;uniqueIndex:uni_alarm_record_lifecycle,priority:2"`     // 报警代码
	AlarmType     string `json:"alarmType" gorm:"column:alarm_type;uniqueIndex:uni_alarm_record_lifecycle,priority:4"`     // 报警类别 0: 普通 1: 预警, 多种方式用','隔开
	AlarmStage    int    `json:"alarmStage" gorm:"column:alarm_stage;uniqueIndex:uni_alarm_record_lifecycle,priority:5"`   // 报警等级
	BeginTime     int64  `json:"beginTime" gorm:"column:begin_time;uniqueIndex:uni_alarm_record_lifecycle,priority:3"`     // 报警触发时间
	EndTime       int64  `json:"endTime"`                                                                                  // 报警恢复时间
	AlarmValue    string `json:"alarmValue"`                                                                               // 实时值
	RecoverValue  string `json:"recoverValue"`                                                                             // 恢复值
	SiteName      string `json:"siteName"`                                                                                 // 站点名称
	EquipmentName string `json:"equipmentName"`                                                                            // 站点名称 + 其他信息
	EquipmentIds  []int  `json:"equipmentIds" gorm:"-:all"`                                                                // 查询设备ID数组
	CorpId        int    `json:"corpId"`                                                                                   // 部门ID
}

func (AlarmDataRecord) TableName() string {
	return "alarm_data_record"
}

type EquipmentAlarmCount struct {
	AlarmCount  int `json:"alarmCount"`
	EquipmentId int `json:"equipmentId"`
}

type EquipmentAlarmStatus struct {
	AlarmCount  int    `json:"alarmCount"`
	EquipmentId int    `json:"equipmentId"`
	AlarmType   string `json:"alarmType"`
}
