package model

// （1）cascade：级联，当父表更新、删除，子表会同步更新和删除
// （2）set null：置空，当父表更新、删除的时候，字表会把外键字段变为null，所以这个时候设计表的时候该字段要允许为null，否则会出错
// （3）restrict：父表在删除和更新记录的时候，要在子表中检查是否有有关该父表要更新和删除的记录，如果有，则不允许删除个更改
// （4）no action：和restrict一样

// 触发器配置(关联设备信息, 数据点信息)
type TriggerConfig struct {
	ID              uint               `json:"id" gorm:"primaryKey,column:id"`
	SiteId          int                `json:"siteId"`                                                                                                     // 站点ID
	EquipmentId     int                `json:"equipmentId"`                                                                                                // 设备ID
	TriggerType     int                `json:"triggerType"`                                                                                                // 触发器类型 0: 报警触发器, 1: 统计触发器 2: 报警+统计... 3: pushgateway + prometheus + altermanager
	TriggerModel    int                `json:"triggerModel"`                                                                                               // 触发器处理模型/模式(区分浮点型,字符串) 0: 浮点型, 1: 字符串
	Code            string             `json:"code"`                                                                                                       // 触发代码 或统计数据名称
	ApiKey          string             `json:"apiKey"`                                                                                                     // 数据池key
	LabelName       string             `json:"labelName"`                                                                                                  // 数据池数据点名称
	LabelId         string             `json:"labelId"`                                                                                                    // 数据池数据点ID
	Cycle           int64              `json:"cycle"`                                                                                                      // 触发模式 0: 表示单次触发模式; 非0: 表示周期触发模式,且值为触发抑制周期(周期时间内数据不触发不推送-过滤)
	Logics          []RealtimeLogic    `json:"logics" gorm:"->;foreignKey:TriggerId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`          // 报警触发逻辑 ->只读
	StrMatchs       []RealtimeStrMatch `json:"strMatchs" gorm:"->;foreignKey:TriggerId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`       // 报警触发逻辑 ->只读
	Timer           int64              `json:"timer"`                                                                                                      // 报警时间 0: 未报警,正常; 非0: 表示第一次报警状态时间记录
	Record          int64              `json:"record"`                                                                                                     // 触发器上次触发状态 0: 上次未报警 非0: 上一次报警, 上次报警时间戳
	LastStrMatch    string             `json:"lastStrMatch"`                                                                                               // 上一次报警str记录
	Marktrigger     string             `json:"marktrigger"`                                                                                                // 网关数据点记录 可gatewayID+dataId md加密 (添加修改labelId时同步更新)
	Formula         string             `json:"formula"`                                                                                                    // 数据池数据点处理公式
	AlarmUser       AlarmUser          `json:"alarmUser" gorm:"->;foreignKey:TriggerId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`       // 报警提醒用户 ->只读
	StatisticConfig StatisticConfig    `json:"statisticConfig" gorm:"->;foreignKey:TriggerId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 统计配置数据 ->只读
	SustainTime     int64              `json:"sustainTime"`                                                                                                // 持续时间
	SustainRecord   int64              `json:"sustainRecord"`                                                                                              // 触发记录点
	// gorm:"-:all"`            // ignore this field when write, read and migrate with struct
	// AlarmUsers  []TriggerAlarmUser `json:"alarmUsers" gorm:"foreignKey:TriggerId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 报警通知用户, 触发逻辑 CASCADE: 级联操作, 更新即更新子表, 删除即删除子表
}

var (
	// 查询固定字段 (排除网关信息)
	PARAMS = []string{"id", "site_id", "equipment_id", "trigger_type", "code", "api_key", "label_name", "label_id",
		"cycle", "timer", "record", "sustain_time", "sustain_record", "formula"}
)

// TableName 指定表名为 trigger_config
func (TriggerConfig) TableName() string {
	return "trigger_config"
}

// 一个触发器里面的一个详细触发逻辑
type RealtimeLogic struct {
	ID           uint    `json:"id" gorm:"primaryKey,column:id"`
	TriggerId    uint    `json:"triggerId"`    // 触发器ID 表里可重复
	Logic        string  `json:"logic"`        // 具体判断逻辑符号
	TriggerValue float64 `json:"triggerValue"` // 具体判断值
}

// TableName 指定表名为 trigger_realtime_logic
func (RealtimeLogic) TableName() string {
	return "trigger_realtime_logic"
}

// 一个触发器里面的一个详细触发逻辑
type RealtimeStrMatch struct {
	ID              uint   `json:"id" gorm:"primaryKey,column:id"`
	TriggerId       uint   `json:"triggerId"`       // 触发器ID 表里可重复
	MatchStr        string `json:"matchStr"`        // 具体判匹配值
	ReplaceCodeFlag bool   `json:"replaceCodeFlag"` // 是否替换code, 生成报警记录
	ReplaceCode     string `json:"replaceCode"`     // 替换报警值
}

// TableName 指定表名为 trigger_realtime_str_match
func (RealtimeStrMatch) TableName() string {
	return "trigger_realtime_str_match"
}

type AlarmUser struct {
	ID           uint   `json:"id" gorm:"primaryKey,column:id"`
	TriggerId    uint   `json:"triggerId"  gorm:"primaryKey,column:trigger_id"` // 触发器ID 维一值
	AlarmCode    string `json:"alarmCode"`                                      // 报警代码
	AlarmType    string `json:"alarmType"`                                      // 报警类别 0: 普通 1: 设备 2: 平台 3: 预警, 多种方式用','隔开
	AlarmStage   int    `json:"alarmStage"`                                     // 报警等级
	AlarmChannel string `json:"alarmChannel"`                                   // 报警渠道，当前仅支持 0（短信）
	AlarmUsers   string `json:"alarmUsers"`                                     // 报警用户 存储用户ID, 多个用户用','隔开
}

// TableName 指定表名为 alarm_user
func (AlarmUser) TableName() string {
	return "alarm_users"
}

type StatisticConfig struct {
	ID                 uint    `json:"id" gorm:"primaryKey,column:id"`
	TriggerId          uint    `json:"triggerId" gorm:"primaryKey,column:trigger_id"` // 触发器ID 维一值
	StatisticCode      string  `json:"statisticCode"`                                 // 统计代码 - 统一代码(指定唯一统计对象)
	StatisticName      string  `json:"statisticName"`                                 // 统计名称 - 可以命名别名
	DataSourceType     int     `json:"dataSourceType"`                                // 数据来源 0: mqtt数据 1:
	StatisticType      string  `json:"statisticType"`                                 // 统计方式 0: 通用统计 1: 报警条数统计
	UseStatisticValue  bool    `json:"useStatisticValue"`                             // 是否使用设置值统计 true: 使用 - 例: 当StatisticType为1时候 false : 不适用
	StatisticValue     float64 `json:"statisticValue"`                                // 统计值, 覆盖原始值进行统计 1: 一条数据 - 例: 当StatisticType为1时候
	StatisticValueType int     `json:"statisticValueType"`                            // 数据类型 0: 累计值 1: 增量值 - 例: 当StatisticType为1时候
}

// TableName 指定表名为 statistic_config
func (StatisticConfig) TableName() string {
	return "statistic_config"
}
