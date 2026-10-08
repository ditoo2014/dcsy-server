package controller

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/util/response"
	"log"

	"github.com/gin-gonic/gin"
)

type ISiteGroupCtrler interface {
	AddSiteGroupArr(ctx *gin.Context)
	DeleteSiteGroup(ctx *gin.Context)
	UpdateSiteGroupArr(ctx *gin.Context)
	FindSiteGroupArr(ctx *gin.Context)
}

type SiteGroupCtrler struct {
}

// 添加
func (sgc SiteGroupCtrler) AddSiteGroupArr(ctx *gin.Context) {

	/* 获取信息 */

	var rSiteGroupArr model.SiteGroupArr
	ctx.Bind(&rSiteGroupArr)

	/* 全部添加 */
	if len(rSiteGroupArr.SiteGroupArrayList) == 0 {
		log.Println("[SiteGroupCtrler] add siteGroupArr data is null.")
		response.Fail(ctx, "空数据", nil)
		return
	}

	// 添加
	siteGroupDao := dao.NewSiteGroupDao(ctx)
	res := siteGroupDao.AddSiteGroupArr(rSiteGroupArr.SiteGroupArrayList)
	if !res {
		response.Fail(ctx, "添加失败", nil)
		return
	}

	response.Success(ctx, "添加成功", nil)
}

// 删除
func (sgc SiteGroupCtrler) DeleteSiteGroup(ctx *gin.Context) {

	/* 获取信息 */

	var rSiteGroup model.SiteGroup
	ctx.Bind(&rSiteGroup)

	/* 校验id */
	if rSiteGroup.ID == 0 {
		log.Println("[SiteGroupCtrler] delete siteGroup id is null.")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 是否有关联的站点
	siteDao := dao.NewSiteDAO(ctx)
	_, res := siteDao.FindSiteBySiteGroupId(int(rSiteGroup.ID))
	if res {
		response.Fail(ctx, "请先删除关联站点", nil)
		return
	}

	// 删除
	siteGroupDao := dao.NewSiteGroupDao(ctx)
	res = siteGroupDao.DeleteSiteGroup(rSiteGroup)
	if !res {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "删除成功", nil)
}

func (sgc SiteGroupCtrler) UpdateSiteGroupArr(ctx *gin.Context) {

	/* 获取信息 */

	var rSiteGroupArr model.SiteGroupArr
	ctx.Bind(&rSiteGroupArr)

	/* 全部修改 */
	if len(rSiteGroupArr.SiteGroupArrayList) == 0 {
		log.Println("[SiteGroupCtrler] update siteGroupArr data is null.")
		response.Fail(ctx, "空数据", nil)
		return
	}

	// 添加
	siteGroupDao := dao.NewSiteGroupDao(ctx)
	res := siteGroupDao.UpdateSiteGroupArr(rSiteGroupArr.SiteGroupArrayList)
	if !res {
		response.Fail(ctx, "修改失败", nil)
		return
	}

	response.Success(ctx, "修改成功", nil)
}

func (sgc SiteGroupCtrler) FindSiteGroupArr(ctx *gin.Context) {

	// 查询全部
	siteGroupDao := dao.NewSiteGroupDao(ctx)
	sga, res := siteGroupDao.FindSiteGroupArr()
	if !res {
		response.Fail(ctx, "数据为空", nil)
		return
	}

	sgal := model.SiteGroupArr{}
	sgal.SiteGroupArrayList = sga
	response.Success(ctx, "查询成功", gin.H{"data": sgal})
}
