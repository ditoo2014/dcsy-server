package dao

import (
	"dicloud/internal/tolink/model"
	entity "dicloud/models/model"
	"dicloud/util/gormutil"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type StreamDAO struct {
	db *gorm.DB
}

// 创建新的 StreamDAO 实例
func NewStreamDAO(ctx *gin.Context) *StreamDAO {
	return &StreamDAO{db: gormutil.GetGormDB(ctx, "")}
}

// 数据池是否存在同名
func (dao *StreamDAO) ExistName(g model.Stream) (model.Stream, bool) {
	c := model.Stream{}

	err := dao.db.Table("stream").Where("name = ? or api_key = ?", g.Name, g.ApiKey).Scan(&c).Error
	if err != nil || c.ID == 0 {
		log.Printf("Failed stream not exist: %v", err)
		return c, false
	}

	return c, true
}

// 添加数据池
func (dao *StreamDAO) AddStream(g model.Stream) (model.Stream, error) {
	err := dao.db.Create(&g).Error
	if err != nil {
		log.Printf("Failed to add stream: %v", err)
		return g, err
	}

	return g, nil
}

// ExistApiKey 判断ApiKey是否存在
func (dao *StreamDAO) ExistApiKey(stream model.Stream) (data model.Stream, flag bool) {
	dao.db.Model(&model.Stream{}).Preload("List").Where("api_key = ?", stream.ApiKey).Find(&data)
	if data.ID == 0 {
		return model.Stream{}, false
	}
	return data, true
}

func (dao *StreamDAO) FindByApiKeys(apiKeys []string) ([]model.Stream, error) {
	streams := []model.Stream{}
	if len(apiKeys) == 0 {
		return streams, nil
	}
	err := dao.db.Model(&model.Stream{}).Preload("List").Where("api_key IN ?", apiKeys).Find(&streams).Error
	return streams, err
}

// Update 更新数据池信息
func (dao *StreamDAO) Update(stream model.Stream) error {
	result := dao.db.Model(&model.Stream{}).Where("api_key = ?", stream.ApiKey).Updates(map[string]interface{}{"admin": stream.Admin, "name": stream.Name, "stream_code": stream.StreamCode})
	return result.Error
}

// Delete 删除数据池
func (dao *StreamDAO) Delete(stream model.Stream) error {

	result := dao.db.Model(&model.Stream{}).Where("api_key = ?", stream.ApiKey).Delete(&model.Stream{})
	return result.Error
}

// FindAll 查询数据池
func (dao *StreamDAO) StreamLabels(stream model.Stream) (model.Stream, error) {
	var data model.Stream
	result := dao.db.Model(&model.Stream{}).Preload("List").Where("api_key = ?", stream.ApiKey).Find(&data)
	return data, result.Error
}

func (dao *StreamDAO) FindPage(pageInfo model.StreamPageSearch, user entity.User) []model.Stream {
	var data []model.Stream
	offset := (pageInfo.Page - 1) * pageInfo.Size
	likeQuery := "%" + pageInfo.Search + "%"

	if user.Isadmin == 1 { //管理员可查询全部的数据池
		if len(pageInfo.Search) == 0 {
			dao.db.Model(&model.Stream{}).Limit(pageInfo.Size).Offset(offset).Find(&data)
		} else {
			dao.db.Model(&model.Stream{}).Where("admin like ?", likeQuery).Or("name like ?", likeQuery).Limit(pageInfo.Size).Offset(offset).Find(&data)
		}
	} else { //用户只能查询自己的全部数据
		dao.db.Model(&model.Stream{}).Where("admin = ?", user.Username).Or("name like ?", likeQuery).Limit(pageInfo.Size).Offset(offset).Find(&data)
	}
	return data
}

func (dao *StreamDAO) GetFindPageTotal(pageInfo model.StreamPageSearch, user entity.User) interface{} {
	var total int64
	likeQuery := "%" + pageInfo.Search + "%"

	if user.Isadmin == 1 { //管理员可查询全部的数据池
		if len(pageInfo.Search) == 0 {
			dao.db.Model(&model.Stream{}).Count(&total)
		} else {
			dao.db.Model(&model.Stream{}).Where("admin like ?", likeQuery).Or("name like ?", likeQuery).Count(&total)
		}
	} else { //用户只能查询自己的全部数据
		dao.db.Model(&model.Stream{}).Where("admin = ?", user.Username).Or("name like ?", likeQuery).Count(&total)
	}
	return total
}

// CreateStreamLabelUnit 创建标签
func (dao *StreamDAO) CreateStreamLabelUnit(streamLabelUnitList []model.StreamLabelUnit) error {
	result := dao.db.Model(&model.StreamLabelUnit{}).Create(&streamLabelUnitList)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// UpdateStreamLabelUnit 根据label_id修改数据
func (dao *StreamDAO) UpdateStreamLabelUnit(streamLabelUnit model.StreamLabelUnit) error {
	result := dao.db.Model(&model.StreamLabelUnit{}).Where("label_id = ?", streamLabelUnit.LabelId).Updates(map[string]interface{}{"gateway_name": streamLabelUnit.GatewayName,
		"gateway_id": streamLabelUnit.GatewayId, "data_id": streamLabelUnit.DataId, "data_name": streamLabelUnit.DataName, "formula": streamLabelUnit.Formula, "unformula": streamLabelUnit.Unformula})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

// DeleteStreamLabelUnit 删除所有标签
func (dao *StreamDAO) DeleteStreamLabelUnit(unit []model.StreamLabelUnit) error {
	tx := dao.db.Begin()
	defer func() {
		if r := recover(); r != nil { //recover() 会捕捉到 panic
			tx.Rollback()
		}
	}()

	for _, v := range unit {
		if err := tx.Model(&model.StreamLabelUnit{}).Where("label_id = ?", v.LabelId).Delete(&model.StreamLabelUnit{}).Error; err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit().Error
}

// GetLabel 获取标签
func (dao *StreamDAO) GetLabel(stream model.Stream) (data []model.StreamLabelUnit, err error) {

	result := dao.db.Model(&model.StreamLabelUnit{}).Where("stream_id = ?", stream.ID).Find(&data)
	fmt.Println("data:", len(data))
	if result.Error != nil {
		return nil, result.Error
	}
	return
}

func (dao *StreamDAO) GetLabelUnitByLabelId(labelId string) (data model.StreamLabelUnit) {

	dao.db.Model(&model.StreamLabelUnit{}).Where("label_id = ?", labelId).First(&data)
	return
}

func (dao *StreamDAO) DeleteStreamLabelUnitByStreamID(StreamInfo model.Stream) error {

	result := dao.db.Model(&model.StreamLabelUnit{}).Where("stream_id = ?", StreamInfo.ID).Delete(&model.StreamLabelUnit{})
	return result.Error
}
