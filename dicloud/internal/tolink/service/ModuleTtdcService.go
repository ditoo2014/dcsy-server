package service

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	"dicloud/pkg/cache"
	"dicloud/util/response"
	"dicloud/util/tdengine"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

// 查询模块通道数据
func FindModuleTtdc(ctx *gin.Context, gatewayInfo model.Gateway) (resMsg gin.H, resErr error) {

	// 查询数据
	moduleTtdcDao := dao.NewModuleTtdcDAO(ctx)
	dataTem, res := moduleTtdcDao.FindModule(gatewayInfo)
	if !res {
		response.Fail(ctx, "查询失败", nil)
		resErr = errors.New("查询失败")
		return
	}
	resMsg = gin.H{"data": dataTem}

	return
}

// 创建模块通道数据
func CreateModuleTtdc(ctx *gin.Context, rGatewayModuleUnit model.GatewayModuleTtdc) (resMsg gin.H, resErr error) {

	/* 创建 */
	ttdcDao := dao.NewModuleTtdcDAO(ctx)

	_, res := ttdcDao.ExistName(rGatewayModuleUnit)
	if res {
		response.Fail(ctx, "模块名称已存在", nil)
		resErr = errors.New("模块名称已存在")
		return
	}

	_, sucCount, dupCount, err := ttdcDao.CreateModule(rGatewayModuleUnit)
	if nil != err {
		log.Println("[ModuleTtdcService] CreateModule err: ", err)
		response.Fail(ctx, "系统错误", nil)
		resErr = errors.New("系统错误")
		return
	}

	resMsg = gin.H{"sucCount": sucCount, "dupCount": dupCount}
	return
}

/* 通过gatewayId删除所有数据 */
func DeleteModuleTtdcByGatewayId(ctx *gin.Context, gatewayId string) (resErr error) {

	// 删除网关数据点
	// 删除模块通道数据点+配置
	ttdcDao := dao.NewModuleTtdcDAO(ctx)
	erro := ttdcDao.DeleteModuleByGatewayId(gatewayId)
	if nil != erro {
		log.Println("[ModuleTtdcService] DeleteByGatewayId: ", erro)
		response.Fail(ctx, "删除网关数据失败", nil)
		resErr = errors.New("删除网关数据失败")
		return
	}
	if err := deleteTtdcRuntimeData(ctx, gatewayId); err != nil {
		log.Printf("[ModuleTtdcService] 删除网关运行数据失败 gatewayId=%s err=%v", gatewayId, err)
		response.Fail(ctx, "删除网关实时数据或历史数据失败", nil)
		return errors.New("删除网关实时数据或历史数据失败")
	}

	return
}

// deleteTtdcRuntimeData only removes storage owned by the merged TTDC battery
// path. Numeric history uses the gateway table, string history uses the STR
// companion table, and their Redis hashes also hold the table-create marker.
func deleteTtdcRuntimeData(ctx *gin.Context, gatewayID string) error {
	gatewayID = strings.TrimSpace(gatewayID)
	if gatewayID == "" {
		return errors.New("网关序列号不能为空")
	}
	db := tdengine.GetTDengine()
	if db == nil {
		return errors.New("TDengine连接未初始化")
	}
	databaseName := quoteTtdcTDIdentifier(tdengine.DBNAME)
	for _, tableID := range []string{gatewayID, gatewayID + "STR"} {
		if _, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s.%s", databaseName, quoteTtdcTDIdentifier(tableID))); err != nil {
			return fmt.Errorf("删除TDengine历史表%s失败: %w", tableID, err)
		}
	}
	if cache.GetRedisClient(ctx) == nil {
		return errors.New("Redis连接未初始化")
	}
	for _, key := range []string{"gateway:" + gatewayID, "gateway:" + gatewayID + "STR"} {
		if err := cache.Delete(ctx, key); err != nil {
			return fmt.Errorf("删除Redis数据%s失败: %w", key, err)
		}
	}
	return nil
}

func quoteTtdcTDIdentifier(value string) string {
	return "`" + strings.ReplaceAll(value, "`", "``") + "`"
}

/* 删除单个通道数据 */
func DeleteModuleTtdc(ctx *gin.Context, rGatewayModuleUnit model.GatewayModuleTtdc) (resMsg gin.H, resErr error) {

	ttdcDao := dao.NewModuleTtdcDAO(ctx)

	if rGatewayModuleUnit.ID == 0 {
		log.Println("[ModuleTtdcService] DeleteModule id is null")
		response.Fail(ctx, "模块不存在", nil)
		return
	}

	gm, res := ttdcDao.DeleteModule(rGatewayModuleUnit)
	if !res {
		resErr = errors.New("删除失败")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	resMsg = gin.H{"data": gm}
	return
}

/* 更新通道模块 */
func UpdateModuleTtdc(ctx *gin.Context, function string, gatewayInfo *model.Gateway, drive model.GatewayModuleTtdc) (resMsg gin.H, resErr error) {

	moduleTtdcDao := dao.NewModuleTtdcDAO(ctx)

	/* 更新内容 */

	switch function {
	case "download":

		// 暂时没有下发, 不考虑
		// if drive.ID == 0 {
		// 	resErr = errors.New("通道id不能为空")
		// 	response.Fail(ctx, "通道id不能为空", nil)
		// 	return
		// }

		// cmdStr, err := moduleTtdcDao.GetDownloadStr(drive)
		// if nil != err {
		// 	log.Println("[ModuleTtdcService] format download str fail", err)
		// 	resErr = errors.New("format download str fail")
		// 	response.Fail(ctx, err.Error(), nil)
		// 	return
		// }
		// // 下发 mqtt 配置到网关
		// CmdMqtt(ctx, drive.GatewayId, cmdStr)

	case "exportexcel":

		if drive.ID == 0 {
			resErr = errors.New("通道id不能为空")
			response.Fail(ctx, "通道id不能为空", nil)
			return
		}

		fileName, err := moduleTtdcDao.ExportExcel(drive)
		if nil != err {
			resErr = errors.New("err.Error()")
			response.Fail(ctx, err.Error(), nil)
			return
		}

		resMsg = gin.H{"url": "/var/www/dicloud/filecache/" + fileName + ".xlsx"}

	case "importexcel":

		err := moduleTtdcDao.ImportExcel(ctx, drive)
		if nil != err {
			resErr = errors.New(err.Error())
			response.Fail(ctx, err.Error(), nil)
			return
		}

	default:

		if drive.ID == 0 {
			resErr = errors.New("通道id不能为空")
			response.Fail(ctx, "通道id不能为空", nil)
			return
		}

		// 校验是否存在
		dataTem := model.GatewayModuleTtdc{ID: drive.ID}
		dataTem, res := moduleTtdcDao.ExistModuleId(dataTem)
		if !res {
			log.Println("[ModuleTtdcService] UpdateModule drive id not exist")
			resErr = errors.New("通道不存在")
			response.Fail(ctx, "通道不存在", nil)
		}

		// 确定网关数据, 不能修改
		drive.GatewayId = dataTem.GatewayId
		_, res = moduleTtdcDao.UpdateModule(function, drive)
		if !res {
			resErr = errors.New("操作通道失败")
			response.Fail(ctx, "操作通道失败", nil)
			return
		}
	}

	return
}
