package model

import pq "gitee.com/opengauss/openGauss-connector-go-pq"

// OAuth认证客户端数据, 需要再平台上进行管理
// 客户携带秘钥数据, 访问平台接口, 平台验证客户数据
// 重定向到登录平台用户界面
// 界面根据客户参数, 登录用户进行授权

// OAuth客户端密钥信息数据
type OAuthClientSecret struct {
	ID                 uint                 `json:"id" gorm:"primaryKey,column:id"`
	ClientName         string               `json:"clientName"`                     // 客户名称
	ClientKey          string               `json:"clientKey"`                      // 客户端Key
	ClientSecret       string               `json:"clientSecret"`                   // 客户端Secret
	ClientRedirectPage string               `json:"clientRedirectPage"`             // 客户端获取授权Token之后, 重定向页面
	Baned              int                  `json:"baned"`                          // 是否禁用
	WhiteList          pq.StringArray       `json:"whiteList" gorm:"type:string[]"` // 白名单IP地址
	ManagerIds         pq.Int64Array        `json:"managerIds" gorm:"type:int[]"`   // 管理员账户Id数组 - 根据管理员Id查询, 超管查全部
	CorpId             int64                `json:"corpId"`                         // 客户端所属部门, 部门之间不可见
	TempLoginInfoList  []OAuthTempLoginInfo `json:"tempLoginInfoList" gorm:"->;foreignKey:loginClientId;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (OAuthClientSecret) TableName() string {
	return "oauth_client_secret"
}

// 临时登录信息数据
type OAuthTempLoginInfo struct {
	ID                 uint          `json:"id" gorm:"primaryKey,column:id"`
	LoginClientId      int64         `json:"loginClientId"`                        // 登录客户端Id
	LoginCode          string        `json:"loginCode"`                            // 临时登录码
	LoginTempUserId    string        `json:"loginTempUserId"`                      // (客)临时登录用户Id
	LoginTempToken     string        `json:"loginTempToken"`                       // (客)临时登录用户Token
	GrantUserId        int64         `json:"grantUserId"`                          // (主)授权用户的Id
	GrantUserCorp      pq.Int64Array `json:"grantUserCorp" gorm:"type:int[]"`      // (主)授权用户的部门 - 临时用户的部门 -> 使用(主)授权用户的部门
	GrantPermissionIds pq.Int64Array `json:"grantPermissionIds" gorm:"type:int[]"` // (客)授予的权限 - 临时权限(假定: 临时授权数较少, 暴露接口不多)
}

func (OAuthTempLoginInfo) TableName() string {
	return "oauth_temp_login_info"
}

type JudgeResult struct { // 数据库事务的特性
	Result   bool  `json:"result"`   // 判断结果
	ClientId int64 `json:"clientId"` // 客户端ID - 用户生成绑定授权码
}
