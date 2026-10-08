package model

type GroupCorpCount struct {
	CorpId int `json:"corpId"`
	Count  int `json:"count"`
}

var (
	EquipmentExportHeader         = []string{"电池ID", "电池类型", "生产厂家", "生产日期", "批次", "规格", "所属公司", "BMS厂家", "BMS版本"}
	SiteExportHeader              = []string{"站点名称", "站址编码", "部门", "详细地址", "经度", "纬度"}
	WarehouseExportHeader         = []string{"库房名称", "库房编号", "库房地址", "管理员", "管理员电话", "部门", "经度", "纬度"}
	EquipmentMaintainRecordHeader = []string{"电池ID", "站址名称", "故障描述", "图片", "运维时间", "标识"}
)
