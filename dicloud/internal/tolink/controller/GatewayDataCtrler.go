package controller

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/util/response"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type IGatewayDataCtrler interface {
	GatewayDataOperate(ctx *gin.Context)
}

type GatewayDataCtrler struct {
}

func (ctrler GatewayDataCtrler) GatewayDataOperate(ctx *gin.Context) {

	/* 获取网关信息 */

	gatewayInfoCache, _ := ctx.Get("gatewayInfo")
	gatewayInfo := gatewayInfoCache.(model.Gateway)

	/* 验证信息 */

	/* 获取信息 */

	var rGatewayDataUnit model.GatewayData
	ctx.Bind(&rGatewayDataUnit)

	/* 获取操作信息 */

	function := ctx.Query("Function")

	/* 执行操作 */

	gatewayDataDao := dao.NewGatewayDataDao(ctx)

	switch {
	case function == "create":

		if 1 > len(rGatewayDataUnit.Name) {
			response.Fail(ctx, "数据点名称为空", nil)
			return
		}
		if rGatewayDataUnit.Dbconfig == 0 {
			response.Fail(ctx, "数据库配置为空", nil)
			return
		}

		rGatewayDataUnit.GatewayId = gatewayInfo.GatewayId
		_, res := gatewayDataDao.ExistDataName(rGatewayDataUnit)
		if res {
			response.Fail(ctx, "数据点名称已存在", nil)
			return
		}

		/* 自生成数据点ID */
		for _idx := 100; _idx > 0; _idx-- {

			rGatewayDataUnit.DataId = fmt.Sprintf("%d", time.Now().UnixNano()%10000000000000000)
			// 是否存在随机生成的 data_id
			if _, res := gatewayDataDao.ExistDataId(rGatewayDataUnit); res {
				rGatewayDataUnit.DataId = ""
				break
			}
			// 不存在, 跳出循环
			if rGatewayDataUnit.DataId != "" {
				break
			}
		}

		// 自动生成id失败
		if rGatewayDataUnit.DataId == "" {
			response.Fail(ctx, "系统错误", nil)
			return
		}

		// 数据库创建失败
		_, res = gatewayDataDao.CreateData(rGatewayDataUnit)
		if !res {
			response.Fail(ctx, "创建失败", nil)
			return
		}

	case function == "delete":

		if 1 > len(rGatewayDataUnit.DataId) && rGatewayDataUnit.ID == 0 {
			response.Fail(ctx, "数据点ID为空", nil)
			return
		}

		// TODO: 删除时序数据
		err := dao.DeleteDBOneById(rGatewayDataUnit)
		if err != nil {
			response.Fail(ctx, "删除时序失败", nil)
			return
		}

		// 删除配置数据点
		if rGatewayDataUnit.ID > 0 {
			gatewayDataDao.DeleteDataDirect(rGatewayDataUnit)
		} else {
			// 通过数据点id删除
			rGatewayDataUnit.GatewayId = gatewayInfo.GatewayId
			_, res := gatewayDataDao.DeleteData(rGatewayDataUnit)
			if !res {
				response.Fail(ctx, "删除失败", nil)
				return
			}
		}

	case function == "update":

		if 1 > len(rGatewayDataUnit.DataId) {
			response.Fail(ctx, "数据点ID为空", nil)
			return
		}
		if rGatewayDataUnit.Dbconfig == 0 {
			response.Fail(ctx, "数据库配置为空", nil)
			return
		}

		// 数据库id更新
		if rGatewayDataUnit.ID > 0 {
			_, res := gatewayDataDao.ExistData(rGatewayDataUnit)
			if !res {
				response.Fail(ctx, "数据点不存在", nil)
				return
			}
			// 根据id, 数据库更新
			_, res = gatewayDataDao.UpdateDataDirect(rGatewayDataUnit)
			if res {
				response.Success(ctx, "操作成功", nil)
				return
			}
		}

		// 数据点id更新
		rGatewayDataUnit.GatewayId = gatewayInfo.GatewayId
		_, res := gatewayDataDao.ExistDataId(rGatewayDataUnit)
		if res {
			response.Fail(ctx, "数据点不存在", nil)
			return
		}

		_, res = gatewayDataDao.UpdateData(rGatewayDataUnit)
		if !res {
			response.Fail(ctx, "更新失败", nil)
			return
		}

	// ????
	case function == "deletedb":

		// TODO: 删除时序数据
		err := dao.DeleteDBOneById(rGatewayDataUnit)
		if err != nil {
			response.Fail(ctx, "删除时序失败", nil)
			return
		}

	default:

		response.Fail(ctx, "功能指令错误", nil)
		return
	}

	/* 返回成功 */

	response.Success(ctx, "操作成功", nil)
}
