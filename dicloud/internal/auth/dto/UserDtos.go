package dto

import (
	"dicloud/models/model"
	"time"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
)

type UserDto struct {
	ID         uint          `json:"id"`
	Username   string        `json:"username"`
	TrueName   string        `json:"trueName"`
	Phone      string        `json:"phone"`
	Email      string        `json:"email"`
	CreateTime time.Time     `json:"createTime" gorm:"type:json;column:create_time"`
	TotalLogin int           `json:"totalLogin"`
	Token      string        `json:"token"`
	LoginIp    string        `json:"loginIp"`
	LastIp     string        `json:"lastIp"`
	Isadmin    int           `json:"isadmin"`
	ParentId   int           `json:"parenId"`
	PlatName   string        `json:"platName"`
	PlatType   string        `json:"platType"`
	Baned      int           `json:"baned"`
	CorpIds    pq.Int64Array `json:"corpIds" gorm:"type:int[]"`
}

type UserListDto struct {
	UserName string `json:"username"`
	TrueName string `json:"trueName"`
}

type UserRole struct {
	UserName string `json:"username"`
	Password string `json:"password"`
	TrueName string `json:"trueName"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`

	RoleId   string `json:"roleId"`
	RoleName string `json:"roleName"`
	RoleDesc string `json:"roleDesc"`
}

type UserNewPasswordDTO struct {
	Password    string `json:"password"`
	NewPassword string `json:"newPassword"`
}

type LoginDTO struct {
	Username  string `json:"username" form:"username"`
	Password  string `json:"password" form:"password"`
	Phone     string `json:"phone" form:"phone"`
	PlatId    int    `json:"platId" form:"platId"`
	SmsCodeId string `json:"smsCodeId" form:"smsCodeId"`
	SmsCode   string `json:"smsCode" form:"smsCode"`
}

type ResetUserPwdDTO struct {
	UsersEntity     model.User `json:"usersEntity"`
	NewPassword     string     `json:"newPassword"`
	ManagerPassword string     `json:"managerPassword"`
}
