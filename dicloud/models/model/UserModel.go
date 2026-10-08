package model

import (
	"encoding/json"
	"time"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
)

// 用户表
type User struct {
	ID         uint          `json:"id" gorm:"primaryKey,column:id"`
	Username   string        `json:"username"`
	Password   string        `json:"password"`
	TrueName   string        `json:"trueName"`
	Phone      string        `json:"phone"`
	Email      string        `json:"email"`
	CreateTime time.Time     `json:"createTime" gorm:"type:json;column:create_time"`
	TotalLogin int           `json:"totalLogin"`
	Token      string        `json:"token"`
	LoginIp    string        `json:"loginIp"`
	LastIp     string        `json:"lastIp"`
	Power      int           `json:"power" gorm:"-:all"` // ignore this field when write, read and migrate with struct
	Isadmin    int           `json:"isadmin"`
	ClientId   string        `json:"clientId"`
	ParentId   int           `json:"parenId"`
	PlatName   string        `json:"platName"`
	PlatType   string        `json:"platType"`
	PlatId     int           `json:"platId"` // 账号使用范围 0: 全平台, 1: 业务云, 2: 采集云, 3: 组态云
	Baned      int           `json:"baned"`
	CorpIds    pq.Int64Array `json:"corpIds" gorm:"type:int[]"`
	Renter     string        `json:"renter"` // 租户平台信息
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
	ID       uint   `json:"id" gorm:"column:id"`
	Level    int    `json:"level"`
	ParentId int    `json:"parentId"`
	CorpName string `json:"corpName"`
	PlatId   int    `json:"platId"`
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
