package model

type SiteGroup struct {
	ID            uint   `json:"id" gorm:"primary key"`
	SiteGroupName string `json:"siteGroupName"`
	TreeId        string `json:"treeId"`
	ParentId      string `json:"parentId"`
	Level         string `json:"level"`
}

// TableName 指定表名为 site_group
func (SiteGroup) TableName() string {
	return "site_group"
}

type SiteGroupArr struct {
	SiteGroupArrayList []SiteGroup `json:"siteGroupArrayList"`
}
