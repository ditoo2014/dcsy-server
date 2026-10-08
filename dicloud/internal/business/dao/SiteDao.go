package dao

import (
	"dicloud/internal/business/model"
	"dicloud/util/gormutil"
	normalutil "dicloud/util/normal"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SiteDAO struct {
	db *gorm.DB
}

// 创建新的 SiteDAO 实例
func NewSiteDAO(ctx *gin.Context) *SiteDAO {
	return &SiteDAO{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *SiteDAO) CreateSite(s model.Site) (model.Site, bool) {

	err := dao.db.Create(&s).Error
	if err != nil || s.ID == 0 {
		log.Println("[SiteDao] Failed to create SITE: ", err)
		return s, false
	}

	return s, true
}

// 创建站点列表
func (dao *SiteDAO) CreateSites(sts []model.Site) ([]model.Site, bool) {

	err := dao.db.Create(&sts).Error
	if err != nil {
		log.Println("Failed to create SITE list: ", err)
		return sts, false
	}

	return sts, true
}

// 查询站点列表
func (dao *SiteDAO) FindSiteAll(s model.Site, offset int, limit int) ([]model.Site, bool) {
	sites := []model.Site{}

	sqlStr := baseSiteSql(s)

	err := dao.db.Table("site").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr).Scan(&sites).Error
	if err != nil {
		log.Println("[SiteDao] to find all SITE: ", err)
		return sites, false
	}

	return sites, true
}

func (dao *SiteDAO) CountSiteAll(s model.Site) (total int64, res bool) {

	sqlStr := baseSiteSql(s)

	err := dao.db.Table("site").Where(sqlStr).Count(&total).Error
	if err != nil {
		log.Println("[SiteDao] Failed to count all SITE: ", err)
		return total, false
	}

	return total, true
}

// 鉴权查询 -id信息
func (dao *SiteDAO) FindSite(s model.Site, siteIds []int, offset int, limit int) ([]model.Site, bool) {
	sites := []model.Site{}

	sqlStr := baseSiteSql(s)

	// 带条件查询
	err := dao.db.Table("site").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND id in ?", siteIds).Scan(&sites).Error
	if err != nil {
		log.Println("[SiteDao] Failed to find SITE: ", err)
		return sites, false
	}

	return sites, true
}

func (dao *SiteDAO) CountSite(s model.Site, userIds []int) (total int64, res bool) {

	sqlStr := baseSiteSql(s)

	// 带条件查询
	err := dao.db.Table("site").Where(sqlStr+" AND id in ?", userIds).Count(&total).Error
	if err != nil {
		log.Println("[SiteDao] Failed to count SITE: ", err)
		return total, false
	}

	return total, true
}

// 根据部门查询
func (dao *SiteDAO) FindSiteByCorp(s model.Site, corpIds []int, offset int, limit int) ([]model.Site, bool) {
	sites := []model.Site{}

	sqlStr := baseSiteSql(s)
	// 未分配部门的站点也查出
	corpIds = append(corpIds, 0)
	corpIds = normalutil.Unique(corpIds)

	// 带条件查询
	err := dao.db.Table("site").Offset(offset).Limit(limit).Order("id desc").Where(sqlStr+" AND corp_id in ?", corpIds).Scan(&sites).Error
	if err != nil {
		log.Println("[SiteDao] Failed to find bind SITE by corp: ", err)
		return sites, false
	}

	return sites, true
}

func (dao *SiteDAO) CountSiteByCorp(s model.Site, corpIds []int) (total int64, res bool) {

	sqlStr := baseSiteSql(s)
	// 未分配部门的站点也查出
	corpIds = append(corpIds, 0)
	corpIds = normalutil.Unique(corpIds)

	// 带条件查询
	err := dao.db.Table("site").Order("id desc").Where(sqlStr+" AND corp_id in ?", corpIds).Count(&total).Error
	if err != nil {
		log.Println("[SiteDao] Failed to find bind SITE by corp: ", err)
		return total, false
	}

	return total, true
}

// 封装基本查询SQL
func baseSiteSql(s model.Site) (sqlStr string) {

	sqlStr = "1=1"
	if s.ID > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND id = %d ", s.ID)
	}
	if len(s.SiteName) > 0 {
		sqlStr = sqlStr + " AND site_name LIKE '%" + s.SiteName + "%' "
	}
	if s.SiteGroupId > 0 {
		sqlStr = sqlStr + fmt.Sprintf(" AND site_group_id = %d ", s.SiteGroupId)
	}
	if len(s.SiteSerialNumber) > 0 {
		sqlStr = sqlStr + " AND site_serial_number LIKE '%" + s.SiteSerialNumber + "%' "
	}

	return
}

// 查询一站点
func (dao *SiteDAO) FindOneSiteById(s model.Site) (model.Site, bool) {

	// 根据id查询
	err := dao.db.Table("site").Order("id desc").Find(&s).Scan(&s).Error
	if err != nil {
		log.Println("[SiteDao] Failed to find one SITE by id: ", err)
		return s, false
	}

	return s, true
}

// 查询站点名是否重名
func (dao *SiteDAO) SiteNameExist(s model.Site) (model.Site, bool) {
	st := model.Site{}
	name := strings.TrimSpace(s.SiteName)
	if name == "" {
		return st, false
	}

	query := dao.db.Table("site").Where("TRIM(site_name) = ?", name)
	if s.ID != 0 {
		query = query.Where("id <> ?", s.ID)
	}
	err := query.Scan(&st).Error
	if err != nil || st.ID == 0 {
		log.Println("[SiteDao] Failed to find by siteName: ", err)
		return st, false
	}

	return st, true
}

func (dao *SiteDAO) ExistSiteSearial(c model.Site) (model.Site, bool) {
	cp := model.Site{}
	serialNumber := strings.TrimSpace(c.SiteSerialNumber)
	if serialNumber == "" {
		return cp, false
	}

	query := dao.db.Model(&model.Site{}).Where("TRIM(site_serial_number) = ?", serialNumber)
	if c.ID != 0 {
		query = query.Where("id <> ?", c.ID)
	}
	err := query.Scan(&cp).Error
	if err != nil || cp.ID == 0 {
		log.Printf("Failed SiteSerialNumber not exist: %v", err)
		return cp, false
	}

	return cp, true
}

// 导入站点
func (dao *SiteDAO) ImportSites(eqs, ueqs []model.Site) ([]model.Site, bool) {
	err := dao.db.Transaction(func(tx *gorm.DB) error {
		if len(eqs) > 0 {
			if err := tx.Create(&eqs).Error; err != nil {
				return err
			}
		}
		if len(ueqs) > 0 {
			for _, v := range ueqs {
				if err := tx.Model(&model.Site{}).Where("site_serial_number = ?", v.SiteSerialNumber).Updates(&v).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		log.Println("Failed to import Site list: ", err)
		return eqs, false
	}

	return eqs, true
}

func (dao *SiteDAO) FindSiteBySiteGroupId(sgid int) ([]model.Site, bool) {
	// 新的容器
	st := []model.Site{}

	// 新结构体 判断查询值
	err := dao.db.Table("site").Order("id desc").Where(" site_group_id = ?", sgid).Scan(&st).Error
	if err != nil || len(st) == 0 {
		log.Println("[SiteDao] Failed to find one SITE by gourpId: ", err)
		return nil, false
	}

	return st, true
}

// 查询站点detail
func (dao *SiteDAO) FindSiteDetail(s model.Site) ([]model.Site, bool) {
	sites := []model.Site{}

	// 查询之后, 反序列化绑定到新结构体
	err := dao.db.Find(&s, datatypes.JSONQuery("site_details").Equals(34.26534, "latitude")).Scan(&sites).Error
	if err != nil || len(sites) == 0 {
		log.Println("[SiteDao] Failed to find SITE by details: ", err)
		return sites, false
	}

	return sites, true
}

// 删除站点
func (dao *SiteDAO) DeleteSite(s model.Site) bool {

	// 根据id 删除
	err := dao.db.Delete(&s).Error
	if err != nil {
		log.Println("[SiteDao] Failed to delete SITE: ", err)
		return false
	}

	return true
}

// 修改站点信息
func (dao *SiteDAO) UpdateSite(s model.Site) (model.Site, bool) {

	// 值更新非空部分
	err := dao.db.Table("site").Updates(&s).Error
	if err != nil {
		log.Println("[SiteDao] Failed to update SITE: ", err)
		return s, false
	}

	return s, true
}
