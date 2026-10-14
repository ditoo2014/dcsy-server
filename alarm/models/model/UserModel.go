package model

import (
	"encoding/json"
	"time"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
)

// 用户表
type User struct {
	ID         uint          `json:"id" gorm:"primaryKey;column:id;size:32"`
	Username   string        `json:"username" gorm:"type:varchar(60);not null;unique"`
	Password   string        `json:"password" gorm:"type:varchar(255);not null"`
	TrueName   string        `json:"trueName" gorm:"type:varchar(60);default:''"`
	Phone      string        `json:"phone" gorm:"type:char(11);default:''"`
	Email      string        `json:"email" gorm:"type:varchar;default:''"`
	CreateTime time.Time     `json:"createTime" gorm:"column:create_time;type:timestamp;default:CURRENT_TIMESTAMP"`
	TotalLogin int           `json:"totalLogin" gorm:"type:integer;default:0"`
	Token      string        `json:"token" gorm:"type:varchar(255);default:''"`
	LoginIp    string        `json:"loginIp" gorm:"type:varchar(60);default:''"`
	LastIp     string        `json:"lastIp" gorm:"type:varchar(60);default:''"`
	Power      int           `json:"power" gorm:"-:all"` // ignore this field when write, read and migrate with struct
	Isadmin    int           `json:"isadmin" gorm:"type:integer;default:0"`
	ClientId   string        `json:"clientId" gorm:"type:varchar(255);default:''"`
	ParentId   int           `json:"parenId" gorm:"type:integer;default:0"`
	PlatName   string        `json:"platName" gorm:"type:varchar(60);default:''"`
	PlatType   string        `json:"platType" gorm:"type:varchar(60);default:''"`
	PlatId     int           `json:"platId" gorm:"type:integer;not null;default:1"` // 账号使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
	Baned      int           `json:"baned" gorm:"type:integer;not null;default:0"`
	CorpIds    pq.Int64Array `json:"corpIds" gorm:"type:int[];default:'{}'"`
	Renter     string        `json:"renter" gorm:"type:varchar(60);default:''"` // 租户平台信息
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
	ID             uint   `json:"id" gorm:"primaryKey,column:id"`
	PermissionName string `json:"permissionName"`
	Url            string `json:"url"`
	PermissionType int    `json:"type" gorm:"column:type"`
	Level          int    `json:"level"`
	ParentId       int    `json:"parentId"`
	ApiStage       int    `json:"apiStage"`
	PlatId         int    `json:"platId"`
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
	ID         uint      `json:"id" gorm:"primaryKey,column:id"`
	RoleName   string    `json:"roleName"`
	RoleDesc   string    `json:"roleDesc"`
	CreateTime time.Time `json:"createTime"`
	PlatId     int       `json:"platId"`
}

// TableName 指定表名为 role
func (Role) TableName() string {
	return "role"
}

// 用户角色表
type UserRole struct {
	ID     uint `json:"id" gorm:"column:id"`
	UserId uint `json:"userId"`
	RoleId uint `json:"roleId"`
}

// TableName 指定表名为 user_role
func (UserRole) TableName() string {
	return "user_role"
}

// 角色权限表
type RolePermission struct {
	ID           uint `json:"id" gorm:"column:id"`
	RoleId       uint `json:"roleId"`
	PermissionId uint `json:"permissionId"`
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
	ID     uint `json:"id" gorm:"column:id"`
	UserId uint `json:"userId"`
	CorpId uint `json:"corpId"`
}

// TableName 指定表名为 user_corp
func (UserCorp) TableName() string {
	return "user_corp"
}
