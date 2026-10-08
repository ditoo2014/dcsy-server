package controller

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	"dicloud/util/response"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type IEquipmentGeographyPositionCtrler interface {
	Create(ctx *gin.Context)
	Find(ctx *gin.Context)
}

type EquipmentGeographyPositionCtrler struct {
}

// 记录设备经传地理位置
func (egpCtrler EquipmentGeographyPositionCtrler) Create(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 绑定数据 */

	var iEgp = model.EquipmentGeographyPosition{}
	ctx.Bind(&iEgp)

	/* 校验数据 */
	if iEgp.EquipmentId == 0 {
		log.Println("[EquipmentGeographyPositionCtrler] equipment id is null")
		response.Fail(ctx, "无效数据", nil)
		return
	}

	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(iEgp.EquipmentId)})
	if !res {
		log.Println("[EquipmentGeographyPositionCtrler] equipment id find fail")
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 填充默认数据
	iEgp.CreateUserId = int(userInfo.ID)
	iEgp.CreateTime = time.Now().Unix()

	equipmentGopPstDao := dao.NewEquipmentGeograpyhPositionDao(ctx)
	egpTem, res := equipmentGopPstDao.Create(iEgp)
	if !res {
		log.Println("[EquipmentGeographyPositionCtrler] create fail")
		response.Fail(ctx, "记录位置失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": egpTem})
}

// 查询设备历经地理位置
func (egpCtrler EquipmentGeographyPositionCtrler) Find(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 绑定数据 */

	var iEgp = model.EquipmentGeographyPosition{}
	ctx.Bind(&iEgp)

	equipmentDao := dao.NewEquipmentDAO(ctx)
	equipmentGopPstDao := dao.NewEquipmentGeograpyhPositionDao(ctx)
	if userInfo.Isadmin == 1 {

		// 如果id为0, 不存在, 默认查询一条数据; 即查询全部位置信息;
		dataTem, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(iEgp.EquipmentId)})
		if !res || dataTem.ID == 0 {
			response.Success(ctx, "设备不存在", nil)
			return
		}

		// 查询
		egpList, res := equipmentGopPstDao.Create(iEgp)
		if !res {
			log.Println("[EquipmentGeographyPositionCtrler] find fail")
			response.Fail(ctx, "查询失败", nil)
			return
		}

		response.Success(ctx, "success", gin.H{"data": egpList})
	} else {

		// 查询用户部门
		var corpIds []int
		for _, v := range userInfo.CorpIds {
			corpIds = append(corpIds, int(v))
		}
		// 子部门查询
		corpIds = service.FindAllSonCorp(ctx, corpIds)

		dataTem, res := equipmentDao.FindEquipmentByIdAndCorp(model.Equipment{ID: uint(iEgp.EquipmentId)}, corpIds)
		if !res || dataTem.ID == 0 {
			response.Success(ctx, "设备不存在", nil)
			return
		}

		// 查询
		egpList, res := equipmentGopPstDao.Create(iEgp)
		if !res {
			log.Println("[EquipmentGeographyPositionCtrler] find fail")
			response.Fail(ctx, "查询失败", nil)
			return
		}

		response.Success(ctx, "success", gin.H{"data": egpList})
	}
}
