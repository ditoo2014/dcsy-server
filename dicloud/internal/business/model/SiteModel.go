package model

import "encoding/json"

type Site struct {
	ID               uint            `json:"id" gorm:"primary key"`
	ApiKey           string          `json:"apiKey"`                    // apikey
	ApiKeys          json.RawMessage `json:"apiKeys" gorm:"type:jsonb"` // 多组apikey信息
	SiteSerialNumber string          `json:"siteSerialNumber"`          // 站点序列号
	SiteName         string          `json:"siteName"`                  // 站点名称
	SiteGroupId      int             `json:"siteGroupId"`
	SiteGroupLevel   int             `json:"siteGroupLevel"`
	SiteGroupName    string          `json:"siteGroupName"`
	SiteDetails      json.RawMessage `json:"siteDetails" gorm:"type:jsonb"` // 其他详细信息
	CorpId           int             `json:"corpId"`                        // 部门id
	CreateUserId     int             `json:"createUserId"`                  // 创建用户id
}

// TableName 指定表名为 site
func (Site) TableName() string {
	return "site"
}
