package model

import (
	"encoding/json"
	"time"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
)

// 用户表
type User struct {
	ID       uint   `json:"id" gorm:"primaryKey;column:id;size:32"`
	Username string `json:"username" gorm:"type:varchar(60);not null;unique"`
	Password string `json:"password" gorm:"type:varchar(255);not null"`
	// 密码安全状态只由后端维护，不参与客户端 JSON 绑定或 Redis 用户信息序列化。
	// 带时区时间保存绝对时刻，季度边界统一由后端转换为北京时间判断。
	PasswordChangedAt  *time.Time    `json:"-" gorm:"column:password_changed_at;type:timestamptz"`
	MustChangePassword bool          `json:"-" gorm:"column:must_change_password;type:boolean;not null;default:false"`
	TrueName           string        `json:"trueName" gorm:"type:varchar(60);default:''"`
	Phone              string        `json:"phone" gorm:"type:char(11);default:''"`
	Email              string        `json:"email" gorm:"type:varchar;default:''"`
	CreateTime         time.Time     `json:"createTime" gorm:"column:create_time;type:timestamp;default:CURRENT_TIMESTAMP"`
	TotalLogin         int           `json:"totalLogin" gorm:"type:integer;default:0"`
	Token              string        `json:"token" gorm:"type:varchar(512);default:''"`
	LoginIp            string        `json:"loginIp" gorm:"type:varchar(60);default:''"`
	LastIp             string        `json:"lastIp" gorm:"type:varchar(60);default:''"`
	Power              int           `json:"power" gorm:"-:all"` // ignore this field when write, read and migrate with struct
	Isadmin            int           `json:"isadmin" gorm:"type:integer;default:0"`
	ClientId           string        `json:"clientId" gorm:"type:varchar(255);default:''"`
	ParentId           int           `json:"parenId" gorm:"type:integer;default:0"`
	PlatName           string        `json:"platName" gorm:"type:varchar(60);default:''"`
	PlatType           string        `json:"platType" gorm:"type:varchar(60);default:''"`
	PlatId             int           `json:"platId" gorm:"type:integer;not null;default:1"` // 账号使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
	Baned              int           `json:"baned" gorm:"type:integer;not null;default:0"`
	CorpIds            pq.Int64Array `json:"corpIds" gorm:"type:int[];default:'{}'"`
	Renter             string        `json:"renter" gorm:"type:varchar(60);default:''"` // 租户平台信息
	// UserCorps      []UserCorp    `json:"userCorps" gorm:"->;foreignKey:UserId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"` // 报警提醒用户 ->只读
}

// TableName 指定表名为 user
func (User) TableName() string {
	return "public.users"
}

// Redis 序列化
// MarshalBinary 实现 encoding.BinaryMarshaler 接口
func (u User) MarshalBinary() ([]byte, error) {
	return json.Marshal(u)
}

// 权限表
type Permission struct {
	ID             uint   `json:"id" gorm:"primaryKey;column:id;size:32"`
	PermissionName string `json:"permissionName" gorm:"type:varchar(255);not null;default:''"`
	Url            string `json:"url" gorm:"type:varchar(255);default:''"`
	PermissionType int    `json:"type" gorm:"column:type;type:integer;default:1"`
	Level          int    `json:"level" gorm:"type:integer;default:0"`
	ParentId       int    `json:"parentId" gorm:"type:integer;default:0"`
	ApiStage       int    `json:"apiStage" gorm:"type:integer;default:0"`
	PlatId         int    `json:"platId" gorm:"type:integer;default:0;index:idx_permission_plat"`
}

// TableName 指定表名为 permission
func (Permission) TableName() string {
	return "permission"
}

type Permissions struct {
	Data []Permission `json:"data"`
}

// Redis 序列化
func (ps Permissions) MarshalBinary() ([]byte, error) {
	return json.Marshal(ps)
}

// 角色表
type Role struct {
	ID         uint      `json:"id" gorm:"primaryKey;column:id;size:32"`
	RoleName   string    `json:"roleName" gorm:"type:varchar(255);not null;default:''"`
	RoleDesc   string    `json:"roleDesc" gorm:"type:varchar(255);default:''"`
	CreateTime time.Time `json:"createTime" gorm:"column:create_time;type:timestamp;default:CURRENT_TIMESTAMP"`
	PlatId     int       `json:"platId" gorm:"type:integer;default:0;index:idx_role_plat"`
}

// TableName 指定表名为 role
func (Role) TableName() string {
	return "role"
}

// 用户角色表
type UserRole struct {
	ID     uint `json:"id" gorm:"primaryKey;column:id;size:32"`
	UserId uint `json:"userId" gorm:"column:user_id;type:integer;not null;uniqueIndex:uk_user_role,priority:1"`
	RoleId uint `json:"roleId" gorm:"column:role_id;type:integer;not null;uniqueIndex:uk_user_role,priority:2"`
	User   User `json:"-" gorm:"foreignKey:UserId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Role   Role `json:"-" gorm:"foreignKey:RoleId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName 指定表名为 user_role
func (UserRole) TableName() string {
	return "user_role"
}

// 角色权限表
type RolePermission struct {
	ID           uint       `json:"id" gorm:"primaryKey;column:id;size:32"`
	RoleId       uint       `json:"roleId" gorm:"column:role_id;type:integer;not null;uniqueIndex:uk_role_permission,priority:1"`
	PermissionId uint       `json:"permissionId" gorm:"column:permission_id;type:integer;not null;uniqueIndex:uk_role_permission,priority:2"`
	Role         Role       `json:"-" gorm:"foreignKey:RoleId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Permission   Permission `json:"-" gorm:"foreignKey:PermissionId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName 指定表名为 role_permission
func (RolePermission) TableName() string {
	return "role_permission"
}

// 用户组,公司,团队
type Corp struct {
	ID       uint   `json:"id" gorm:"primaryKey;column:id;size:32"`
	Level    int    `json:"level" gorm:"type:integer;not null;default:0"`
	ParentId int    `json:"parentId" gorm:"type:integer;not null;default:0"`
	CorpName string `json:"corpName" gorm:"type:varchar(255);not null;default:'';uniqueIndex:uk_corp_plat_name,priority:2"`
	PlatId   int    `json:"platId" gorm:"type:integer;not null;default:0;uniqueIndex:uk_corp_plat_name,priority:1"`
}

// TableName 指定表名为 corp
func (Corp) TableName() string {
	return "corp"
}

// 用户分组关系表
type UserCorp struct {
	ID     uint `json:"id" gorm:"primaryKey;column:id;size:32"`
	UserId uint `json:"userId" gorm:"column:user_id;type:integer;not null;uniqueIndex:uk_user_corp,priority:1"`
	CorpId uint `json:"corpId" gorm:"column:corp_id;type:integer;not null;uniqueIndex:uk_user_corp,priority:2"`
	User   User `json:"-" gorm:"foreignKey:UserId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Corp   Corp `json:"-" gorm:"foreignKey:CorpId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

// TableName 指定表名为 user_corp
func (UserCorp) TableName() string {
	return "user_corp"
}
