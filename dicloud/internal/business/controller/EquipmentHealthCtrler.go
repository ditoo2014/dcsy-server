package controller

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/dto"
	"dicloud/internal/business/model"
	businessservice "dicloud/internal/business/service"
	commonmodel "dicloud/internal/common/model"
	commonservice "dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	"dicloud/util/response"
	"errors"
	"io"
	"log"

	"github.com/gin-gonic/gin"
)

type IEquipmentHealthCtrler interface {
	AddEquipmentHealth(ctx *gin.Context)
	UpdateEquipmentHealth(ctx *gin.Context)
	DeleteEquipmentHealth(ctx *gin.Context)
	FindEquipmentHealth(ctx *gin.Context)
	FindEquipmentHealthStatistics(ctx *gin.Context)
}

type EquipmentHealthCtrler struct {
}

// 添加设备健康度判断标准信息
func (whc EquipmentHealthCtrler) AddEquipmentHealth(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	if userInfo.Isadmin != 1 {
		response.Fail(ctx, "权限不足，只能管理员管理", nil)
		return
	}

	/* 绑定信息 */

	var rEquipmentHealthStandard model.EquipmentHealthStandard
	ctx.Bind(&rEquipmentHealthStandard)

	/* 校验数据 */
	if len(rEquipmentHealthStandard.LevelStandards) != len(commonmodel.StatisticHealthCodeArr) {
		response.Fail(ctx, "统计标准不符合规则", nil)
		return
	}

	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)

	sdTem, res := healthStandardDao.Find()
	if res || sdTem.ID > 0 {
		response.Fail(ctx, "已存在统计标准，无法创建", nil)
		return
	}

	rEquipmentHealthStandard.StandardName = "健康度统计标准"
	// 创建
	wh, res := healthStandardDao.Create(rEquipmentHealthStandard)
	if !res {
		response.Fail(ctx, "创建失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": wh})
}

// 修改设备健康度判断标准信息
func (whc EquipmentHealthCtrler) UpdateEquipmentHealth(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	if userInfo.Isadmin != 1 {
		response.Fail(ctx, "权限不足，只能管理员管理", nil)
		return
	}

	/* 绑定信息 */
	var rEquipmentHealthStandard model.EquipmentHealthStandard
	ctx.Bind(&rEquipmentHealthStandard)

	if rEquipmentHealthStandard.ID == 0 {
		log.Println("[EquipmentHealthStandard] controller update house fail ID is null")
		response.Fail(ctx, "更新失败", nil)
		return
	}

	/* 校验数据 */
	if len(rEquipmentHealthStandard.LevelStandards) != len(commonmodel.StatisticHealthCodeArr) {
		response.Fail(ctx, "统计标准不符合规则", nil)
		return
	}

	// 更新
	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)
	wh, res := healthStandardDao.Update(rEquipmentHealthStandard)
	if !res {
		response.Fail(ctx, "更新失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": wh})
}

// 删除设备健康度判断标准信息
func (whc EquipmentHealthCtrler) DeleteEquipmentHealth(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	if userInfo.Isadmin != 1 {
		response.Fail(ctx, "权限不足，只能管理员管理", nil)
		return
	}

	/* 绑定信息 */
	var rEquipmentHealthStandard model.EquipmentHealthStandard
	ctx.Bind(&rEquipmentHealthStandard)

	if rEquipmentHealthStandard.ID == 0 {
		log.Println("[EquipmentHealthStandard] controller delete house fail ID is null")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 删除
	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)
	res := healthStandardDao.Delete(rEquipmentHealthStandard)
	if !res {
		response.Fail(ctx, "创建失败", nil)
		return
	}

	response.Success(ctx, "success", nil)
}

// 查询设备健康度判断标准信息列表
func (whc EquipmentHealthCtrler) FindEquipmentHealth(ctx *gin.Context) {

	/* 绑定信息 */
	var rEquipmentHealthStandard model.EquipmentHealthStandard
	ctx.Bind(&rEquipmentHealthStandard)

	/* 鉴权查询 */
	healthStandardDao := dao.NewEquipmentHealthStandardDao(ctx)

	// 查询
	whs, res := healthStandardDao.Find()
	if !res {
		response.Success(ctx, "success", nil)
		return
	}
	response.Success(ctx, "success", gin.H{"data": whs})

}

// FindEquipmentHealthStatistics queries the realtime health distribution for online batteries.
func (whc EquipmentHealthCtrler) FindEquipmentHealthStatistics(ctx *gin.Context) {
	userInfoCache, exist := ctx.Get("userInfo")
	userInfo, validUser := userInfoCache.(usermodel.User)
	if !exist || !validUser {
		response.Fail(ctx, "用户信息无效", nil)
		return
	}

	var request dto.EquipmentHealthStatisticsRequest
	if err := ctx.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(ctx, "请求参数错误", nil)
		return
	}

	corpIds, authorized := equipmentHealthStatisticsCorpIds(ctx, userInfo, request.CorpId)
	if !authorized {
		response.Fail(ctx, "非法部门, 未授权", nil)
		return
	}

	healthStatisticsDAO := dao.NewEquipmentHealthStatisticsDAO(ctx)
	equipmentIDs, err := healthStatisticsDAO.FindOnlineEquipmentIDs(corpIds)
	if err != nil {
		log.Println("[EquipmentHealthStatistics] query online batteries failed: ", err)
		response.Fail(ctx, "查询失败", nil)
		return
	}
	marktriggers := businessservice.GetEquipmentHealthMarktriggerList(ctx, equipmentIDs)
	log.Printf("[EquipmentHealthStatistics] online=%d healthPoints=%d", len(equipmentIDs), len(marktriggers))
	statistics, err := healthStatisticsDAO.FindRealtimeStatistics(int64(len(equipmentIDs)), marktriggers)
	if err != nil {
		log.Println("[EquipmentHealthStatistics] query failed: ", err)
		response.Fail(ctx, "查询失败", nil)
		return
	}

	response.Success(ctx, "查询成功", gin.H{
		"totalCount":     statistics.TotalCount,
		"validCount":     statistics.ValidCount,
		"unknownCount":   statistics.UnknownCount,
		"coverageRate":   statistics.CoverageRate,
		"percentageBase": statistics.PercentageBase,
		"updatedAt":      statistics.UpdatedAt,
		"levels":         statistics.Levels,
	})
}

func equipmentHealthStatisticsCorpIds(ctx *gin.Context, userInfo usermodel.User, requestedCorpId *int) ([]int, bool) {
	if userInfo.Isadmin == 1 {
		if requestedCorpId == nil {
			return nil, true
		}
		return nonNilCorpIds(commonservice.FindAllSonCorp(ctx, []int{*requestedCorpId})), true
	}

	userCorpIds := make([]int, 0, len(userInfo.CorpIds))
	for _, corpId := range userInfo.CorpIds {
		userCorpIds = append(userCorpIds, int(corpId))
	}
	userCorpIds = commonservice.FindAllSonCorp(ctx, userCorpIds)

	if requestedCorpId == nil {
		return nonNilCorpIds(userCorpIds), true
	}
	for _, corpId := range userCorpIds {
		if corpId == *requestedCorpId {
			return nonNilCorpIds(commonservice.FindAllSonCorp(ctx, []int{*requestedCorpId})), true
		}
	}

	return nil, false
}

func nonNilCorpIds(corpIds []int) []int {
	if corpIds == nil {
		return []int{}
	}
	return corpIds
}
