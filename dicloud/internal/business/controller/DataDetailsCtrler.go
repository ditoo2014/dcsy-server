package controller

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/util/response"
	"log"

	"github.com/gin-gonic/gin"
)

type IDataDetailsCtrler interface {
	AddDataDetails(ctx *gin.Context)
	AddDataDetailsList(ctx *gin.Context)
	FindDataDetails(ctx *gin.Context)
	DeleteDataDetails(ctx *gin.Context)
	UpdateDataDetails(ctx *gin.Context)
	UpdateDataDetailsList(ctx *gin.Context)
}

type DataDetailsCtrler struct {
}

// 查询数据点
func (ddc DataDetailsCtrler) FindDataDetails(ctx *gin.Context) {

	/* 获取信息 */

	var rDataDetails model.DataDetails
	ctx.Bind(&rDataDetails)

	// 查询全部, 不分页
	dataDetailsDao := dao.NewDataDetailsDao(ctx)
	ddss, res := dataDetailsDao.FindDataDetails(rDataDetails)
	if !res {
		response.Success(ctx, "success", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": ddss})
}

// 添加数据点
func (ddc DataDetailsCtrler) AddDataDetails(ctx *gin.Context) {

	/* 获取信息 */

	var rDataDetails model.DataDetails
	ctx.Bind(&rDataDetails)

	if rDataDetails.EquipmentId == 0 {
		response.Fail(ctx, "设备参数不合法", nil)
		return
	}
	if len(rDataDetails.LabelId) == 0 || len(rDataDetails.LabelName) == 0 {
		response.Fail(ctx, "参数不合法", nil)
		return
	}
	if len(rDataDetails.Name) == 0 {
		rDataDetails.Name = rDataDetails.LabelName
	}
	if rDataDetails.ContentDataDetails == nil {
		rDataDetails.ContentDataDetails = []byte(`{}`)
	}

	/* 校验数据 */

	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(rDataDetails.EquipmentId)})
	if !res {
		log.Println("[BUSINESS] datadetail controller equipment not exist")
		response.Fail(ctx, "添加失败", nil)
		return
	}

	// 添加单个数据
	dataDetailsDao := dao.NewDataDetailsDao(ctx)
	dds, res := dataDetailsDao.AddDataDetails(rDataDetails)
	if !res {
		log.Println("[BUSINESS] datadetail controller add fail")
		response.Fail(ctx, "添加失败", nil)
		return
	}

	response.Success(ctx, "添加成功", gin.H{"data": dds})
}

// 批量添加
func (ddc DataDetailsCtrler) AddDataDetailsList(ctx *gin.Context) {

	/* 获取信息 */

	var rDataDetailsList []model.DataDetails
	ctx.Bind(&rDataDetailsList)

	if len(rDataDetailsList) == 0 {
		response.Success(ctx, "success", nil)
		return
	}

	rDataDetails := rDataDetailsList[0]
	if rDataDetails.EquipmentId == 0 {
		response.Fail(ctx, "设备参数不合法", nil)
		return
	}

	/* 校验数据 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(rDataDetailsList[0].EquipmentId)})
	if !res {
		log.Println("[BUSINESS] datadetail controller equipment not exist")
		response.Fail(ctx, "添加失败", nil)
		return
	}

	// 批量添加
	dataDetailsDao := dao.NewDataDetailsDao(ctx)
	dds, res := dataDetailsDao.AddDataDetailsList(rDataDetailsList)
	if !res {
		log.Println("[BUSINESS] datadetail controller add fail")
		response.Fail(ctx, "添加失败", nil)
		return
	}

	response.Success(ctx, "添加成功", gin.H{"data": dds})
}

// 添加数据点
func (ddc DataDetailsCtrler) UpdateDataDetails(ctx *gin.Context) {

	/* 获取信息 */

	var rDataDetails model.DataDetails
	ctx.Bind(&rDataDetails)

	/* 校验数据 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(rDataDetails.EquipmentId)})
	if !res {
		log.Println("[BUSINESS] datadetail controller equipment not exist")
		response.Fail(ctx, "添加失败", nil)
		return
	}

	// 更新单个数据
	dataDetailsDao := dao.NewDataDetailsDao(ctx)
	res = dataDetailsDao.UpdateDataDetails(rDataDetails)
	if !res {
		log.Println("[BUSINESS] datadetail controller add fail")
		response.Fail(ctx, "添加失败", nil)
		return
	}

	response.Success(ctx, "添加成功", gin.H{"data": rDataDetails})
}

// 批量更新数据详情
func (ddc DataDetailsCtrler) UpdateDataDetailsList(ctx *gin.Context) {

	/* 获取信息 */

	var rDataDetailsList []model.DataDetails
	ctx.Bind(&rDataDetailsList)

	if len(rDataDetailsList) == 0 {
		response.Success(ctx, "success", nil)
		return
	}

	/* 校验数据 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(rDataDetailsList[0].EquipmentId)})
	if !res {
		log.Println("[BUSINESS] datadetail controller equipment not exist")
		response.Fail(ctx, "更新失败", nil)
		return
	}

	// 批量更新
	dataDetailsDao := dao.NewDataDetailsDao(ctx)
	res = dataDetailsDao.UpdateDataDetailsList(rDataDetailsList)
	if !res {
		log.Println("[BUSINESS] datadetail controller update fail")
		response.Fail(ctx, "更新失败", nil)
		return
	}

	response.Success(ctx, "更新成功", gin.H{"data": rDataDetailsList})
}

func (ddc DataDetailsCtrler) DeleteDataDetails(ctx *gin.Context) {

	/* 获取信息 */

	var rDataDetails model.DataDetails
	ctx.Bind(&rDataDetails)

	/* 校验数据 */
	if rDataDetails.ID == 0 {
		log.Println("[BUSINESS] datadetail controller delete fail: id is zero")
		response.Fail(ctx, "删除失败", nil)
	}

	// 删除
	dataDetailsDao := dao.NewDataDetailsDao(ctx)
	dds, res := dataDetailsDao.DeleteDataDetails(rDataDetails)
	if !res {
		log.Println("[BUSINESS] datadetail controller delete fail")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "删除成功", gin.H{"data": dds})
}
