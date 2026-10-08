package dao

import (
	"dicloud/models/model"
	"dicloud/util/gormutil"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CorpDAO struct {
	db *gorm.DB
}

func NewCorpDAO(ctx *gin.Context) *CorpDAO {
	return &CorpDAO{db: gormutil.GetGormDB(ctx, "")}
}

func (dao *CorpDAO) List(platId int) ([]model.Corp, error) {
	var data []model.Corp
	err := dao.db.Table("corp").Order("id asc").Where("plat_id = ? OR plat_id = 0", platId).Find(&data).Error
	if err != nil {
		log.Printf("Failed to list corp: %v", err)
		return data, err
	}

	return data, nil
}

func (dao *CorpDAO) ExistName(c model.Corp) (model.Corp, bool) {
	cp := model.Corp{}

	err := dao.db.Table("corp").Where("corp_name = ? and (plat_id = 0 OR plat_id = ?)", c.CorpName, c.PlatId).Scan(&cp).Error
	if err != nil || cp.ID == 0 {
		log.Printf("Failed corp not exist: %v", err)
		return c, false
	}

	return cp, true
}

// 是否存在id
func (dao *CorpDAO) ExistId(c model.Corp) (model.Corp, bool) {
	cp := model.Corp{}

	// 自动封装查询id
	err := dao.db.Table("corp").Find(&c).Scan(&cp).Error
	if err != nil || cp.ID == 0 {
		log.Printf("Failed corp id not exist: %v", err)
		return c, false
	}

	return cp, true
}

// 添加
func (dao *CorpDAO) Add(c model.Corp) (model.Corp, bool) {

	err := dao.db.Create(&c).Error
	if err != nil {
		log.Printf("Failed corp not exist: %v", err)
		return c, false
	}

	return c, true
}

// 删除
func (dao *CorpDAO) Delete(c model.Corp) (model.Corp, bool) {

	err := dao.db.Table("corp").Delete(&c).Error
	if err != nil {
		log.Printf("Failed to delete corp: %v", err)
		return c, false
	}

	return c, true
}

// 更新
func (dao *CorpDAO) Update(c model.Corp) (model.Corp, bool) {

	err := dao.db.Table("corp").Updates(&c).Error
	if err != nil {
		log.Printf("Failed to update corp: %v", err)
		return c, false
	}

	return c, true
}

func (dao *CorpDAO) Find(c model.Corp) (model.Corp, bool) {

	err := dao.db.Model(&c).Order("id asc").Where("(plat_id = 0 OR plat_id = ?)", c.PlatId).Find(&c).Error
	if err != nil {
		log.Printf("Failed to find corp: %v", err)
		return c, false
	}

	return c, true
}

func (dao *CorpDAO) FindCorpList(c model.Corp, platId int) ([]model.Corp, bool) {
	var data []model.Corp
	err := dao.db.Table("corp").Order("id asc").Where("plat_id = ? OR plat_id = 0", platId).Find(&data).Error
	if err != nil {
		log.Printf("Failed to list corp: %v", err)
		return data, false
	}

	return data, true
}

// 通过id查询
func (dao *CorpDAO) FindByIds(c model.Corp, corpIds []int) ([]model.Corp, bool) {
	var dataList []model.Corp

	err := dao.db.Model(&c).Order("id asc").Where("(plat_id = 0 OR plat_id = ?) AND id in ? ", c.PlatId, corpIds).Find(&dataList).Error
	if err != nil {
		log.Printf("Failed to find corp by ids: %v", err)
		return dataList, false
	}

	return dataList, true
}

// 查询所有子部门
func (dao *CorpDAO) FindAllSon(corpIds []int) ([]int, bool) {

	// 递归查询
	mapSearch := make(map[int]int)
	for i := 0; i < len(corpIds); i++ {
		// 不重复查询
		if mapSearch[corpIds[i]] > 0 {
			continue
		}
		mapSearch[corpIds[i]] = corpIds[i]
		// 查询
		dataTem, res := findSonCorp(*dao.db, corpIds[i])
		if res && len(dataTem) > 0 {
			for _, vCorp := range dataTem {
				corpIds = append(corpIds, int(vCorp.ID))
			}
		}
	}

	return corpIds, true
}

func findSonCorp(db gorm.DB, parentId int) ([]model.Corp, bool) {
	c := []model.Corp{}

	err := db.Model(&c).Where("parent_id = ? ", parentId).Find(&c).Error
	if err != nil {
		log.Printf("Failed to find son corp: %v", err)
		return c, false
	}

	return c, true
}

// 查询所有管理父部门
func (dao *CorpDAO) FindAllParent(corpIds []int) ([]int, bool) {

	// 递归查询
	mapSearch := make(map[int]int)
	for i := 0; i < len(corpIds); i++ {
		// 不重复查询
		if mapSearch[corpIds[i]] > 0 {
			continue
		}
		mapSearch[corpIds[i]] = corpIds[i]
		// 查询
		dataTem, res := findCorpSelf(*dao.db, corpIds[i])
		if res && len(dataTem) > 0 {
			for _, vCorp := range dataTem {
				// 查父节点
				corpIds = append(corpIds, int(vCorp.ParentId))
			}
		}
	}

	return corpIds, true
}

func findCorpSelf(db gorm.DB, id int) ([]model.Corp, bool) {
	c := []model.Corp{}

	err := db.Model(&c).Where("id = ? ", id).Find(&c).Error
	if err != nil {
		log.Printf("Failed to find self corp: %v", err)
		return c, false
	}

	return c, true
}
