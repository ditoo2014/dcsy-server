package controller

import (
	authdao "dicloud/internal/auth/dao"
	"dicloud/internal/business/client"
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/dto"
	"dicloud/internal/common/grpc/tolink_grpc"
	usermodel "dicloud/models/model"
	"dicloud/util/response"
	snowflakeutil "dicloud/util/snowflake"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type IAutoSyncAllCommonDataCtrler interface {
	ImportCommonData(ctx *gin.Context)
	DeleteCommonDataByEquipmentId(ctx *gin.Context)
}

type AutoSyncAllCommonDataCtrler struct {
}

// 平台通用数据导入接口, 包含设备信息/网关数据点信息, 自动封装导入其他数据
func (ctrler AutoSyncAllCommonDataCtrler) ImportCommonData(ctx *gin.Context) {

	/** 逻辑整理
	===========================================
	== 导入数据
	== 转为对象
	== 封装数据 Gateway/Equipment 等
	== 调用保存 - 可转为并发方式 - TODO: 连接管理(同一个连接会不会影响数据 - MVCC), 测试是否放到外部处理(减少连接创建时间 - newClient会不会影响性能)
	===========================================
	*/

	/* 获取用户信息 */
	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(usermodel.User)

	/**
	=================
	== 获取上传文件
	=================
	*/
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Println("[BUSINESS-PLAT] upload file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer file.Close()

	/**
	=====================================================
	== 临时文件处理 - 可跳过复制临时文件, 直接操作上传文件
	=====================================================
	*/
	filePath := fmt.Sprintf("equipment_import_%d.csv", time.Now().Unix())
	tempFile, err := os.Create(filePath)
	if err != nil {
		log.Println("[BUSINESS-PLAT] create temp file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer os.Remove(filePath)
	defer tempFile.Close()

	/**
	===========
	== 复制文件
	===========
	*/
	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Println("[BUSINESS-PLAT] copy to temp file fail", err)
		response.Fail(ctx, "文件复制出错", nil)
		return
	}
	/**
	=====================
	== 重新读取复制文件
	=====================
	*/
	tempFile, err = os.Open(filePath)
	if err != nil {
		log.Println("[BUSINESS-PLAT] csv open file fail", err)
		response.Fail(ctx, "打开文件出错", nil)
		return
	}
	defer tempFile.Close()
	reader := csv.NewReader(tempFile)

	/**
	=====================
	== 全部门ID对应名称
	=====================
	*/
	var corpMap = make(map[string]int)
	corpDao := authdao.NewCorpDAO(ctx)
	corps, _ := corpDao.FindCorpList(usermodel.Corp{}, 1)
	if len(corps) > 0 {
		for _, v := range corps {
			corpMap[v.CorpName] = int(v.ID)
		}
	}

	/**
	=====================
	== 全站点ID对应名称
	=====================
	*/
	var siteMap = make(map[string]int)
	siteDao := dao.NewSiteDAO(ctx)
	sites, _ := siteDao.FindSiteAll(model.Site{}, 0, 10000)
	if len(sites) > 0 {
		for _, v := range sites {
			siteMap[v.SiteName] = int(v.ID)
			siteMap[v.SiteSerialNumber] = int(v.ID)
		}
	}

	var total = 0                         // (处理)总个数
	var problemCount = 0                  // 失败个数
	var formatCount = 0                   // 成功识别个数
	var commonDatas []*dto.CommonDataDTO  // 设备数据列表 - 使用指针操作, 防止锁失控
	var dataPosMap = make(map[string]int) // 记录设备位置 - CSV表读取顺序位置

	// 二维表操作
	for {
		record, err := reader.Read()
		if err == io.EOF {
			// finished!
			fmt.Println("[BUSINESS-PLAT] EOF exit.")
			break
		}

		total++
		if total == 1 {
			// 第一行为标题
			continue
		}
		if err != nil {
			fmt.Println(err)
		}

		// 检查数据是否有空值
		var val_valued = true
		for idx := range record {
			if len(strings.ReplaceAll(record[idx], "\\N", "")) == 0 {
				val_valued = false
				break
			}
		}
		if !val_valued {
			problemCount++
			continue
		}
		csvSiteId := record[0]        // 站点ID
		csvSiteName := record[1]      // 站点名称
		csvSiteLongitude := record[2] // 站点经度
		csvSiteLatitude := record[3]  // 站点纬度
		csvCorpName := record[4]      // 部门名称
		csvModel := record[5]         // 机型
		csvEquipmentCode := record[6] // 设备电池编号
		csvEquipmentName := record[7] // 设备名称
		csvDataId := record[8]        // 数据点ID
		csvDataName := record[9]      // 数据点名称

		/**
		============
		== 数据填充
		============
		*/
		var commonData = &dto.CommonDataDTO{}
		var equipment = &commonData.Equipment
		var gateway = &commonData.Gateway
		alreadyExist := -1
		var equipmentCode = strings.ReplaceAll(csvEquipmentCode, " ", "")
		if dataPosMap[equipmentCode] > 0 {
			alreadyExist = dataPosMap[equipmentCode] - 1
			// 操作既往数据, 不新增
			commonData = commonDatas[alreadyExist]
			equipment = &commonData.Equipment
			gateway = &commonData.Gateway
		}

		/**
		======================
		== Equipment/Gatway基础数据
		======================
		*/
		if alreadyExist < 0 {

			/**
			========================
			== 站点ID, 即时新增填充
			========================
			*/
			// 站点名称
			equipment.SiteName = csvSiteName
			// 站点存在
			if siteMap[csvSiteId] > 0 {
				// 站点ID
				equipment.SiteId = siteMap[csvSiteId]
			}
			if siteMap[csvSiteName] > 0 {
				// 站点ID
				if equipment.SiteId == 0 {
					equipment.SiteId = siteMap[csvSiteName]
				}
			}

			// 新建站点
			if equipment.SiteId == 0 {
				// 不存在站点
				var siteTem = model.Site{
					SiteSerialNumber: csvSiteId,
					SiteName:         csvSiteName,
					CorpId:           0,
					// CreateUserId:     int(userInfo.ID),
					CreateUserId: 0,
				}

				/**
				===========================================================
				== 默认信息:
				== corpName = 成都市分公司,
				== latitude = 30.570995, longitude = 104.062263,
				== cityName = 成都市, provinceName = 四川省
				===========================================================
				*/
				var detailsMap = map[string]any{}
				detailsMap["longitude"] = csvSiteLongitude
				detailsMap["latitude"] = csvSiteLatitude
				detailsTem, err := json.Marshal(detailsMap)
				if err == nil {
					siteTem.SiteDetails = detailsTem
				}

				// 部门数据填充
				if corpMap[csvCorpName] > 0 {
					siteTem.CorpId = corpMap[csvCorpName]
				} else if corpMap["成都市分公司"] > 0 {
					// 填充默认部门
					siteTem.CorpId = corpMap["成都市分公司"]
				}

				baseSiteDataFill(&siteTem)
				siteTem, res := siteDao.CreateSite(siteTem)
				if res {
					equipment.SiteId = int(siteTem.ID)
					equipment.SiteName = siteTem.SiteName
					siteMap[siteTem.SiteName] = int(siteTem.ID)
					siteMap[siteTem.SiteSerialNumber] = int(siteTem.ID)
				}
			}

			// 部门数据填充
			if corpMap[csvCorpName] > 0 {
				equipment.CorpId = corpMap[csvCorpName]
				gateway.CorpId = int64(corpMap[csvCorpName])
			} else {
				if corpMap["成都市分公司"] > 0 {
					// 填充默认部门
					equipment.CorpId = corpMap["成都市分公司"]
					gateway.CorpId = int64(corpMap["成都市分公司"])
				} else {
					equipment.CorpId = 0
					gateway.CorpId = 0
				}
			}

			/**
			======================
			== 固定顺序导入基础数据
			======================
			*/
			//========RelatedData==========
			apiKey := primitive.NewObjectID().Hex()
			/**
			=========================================================
			=== TODO: 可判断同序号设备是否存在, 存在使用已存在的apiKey
			=========================================================
			*/
			equipment.EquipmentName = strings.ReplaceAll(strings.ReplaceAll(csvEquipmentName, "\n", ""), "/", "-")
			equipment.EquipmentCode = equipmentCode
			equipment.PositionStatus = 3 // 默认: 上站状态

			// TODO: 默认添加网关信息, 区分采集点个数/指定算法预测类型
			detailMapTem := make(map[string]any)
			detailMapTem["model"] = csvModel
			switch csvModel {
			case "DC-FSU-Server-CSV-16":
				detailMapTem["ditoo_prediction_soc"] = 0
				detailMapTem["ditoo_ecores"] = 16
				detailMapTem["ditoo_tempnum"] = 4
			default:
			}
			mapStr, err := json.Marshal(detailMapTem)
			if err == nil {
				equipment.EquipmentDetails = []byte(mapStr)
			} else {
				log.Println("[BUSINESS-PLAT] default equipment details fill failed")
			}

			baseEquipmentDataFill(equipment)
			// equipment.CreateUserId = int(userInfo.ID)
			equipment.CreateUserId = 0
			equipment.ApiKey = apiKey

			// gateway.CreateUserId = int64(userInfo.ID)
			gateway.CreateUserId = 0
			gateway.GatewayId = equipment.EquipmentCode
			gateway.Name = equipment.EquipmentName
			gateway.Model = csvModel
			gateway.ApiKey = apiKey
			gateway.Card = "-"

			// 默认数据添加
			defaultDataFromat(commonData)
			commonDatas = append(commonDatas, commonData)
			// 记录已存在
			dataPosMap[equipmentCode] = len(commonDatas)
		}

		/**
		==================
		== 处理设备数据点
		==================
		*/
		{
			var existData = false
			if len(commonData.Gateway.GatewayDatas) > 0 {
				for _, v := range commonData.Gateway.GatewayDatas {
					// 数据点ID是否存在
					if v.DataId == csvDataId {
						existData = true
						break
					}
				}
			}
			// 添加数据点
			if !existData {

				// 转换数据点名称
				var dataName = ""
				if len(dto.DP_NormalDataTransFormat[csvDataName]) > 0 {
					dataName = dto.DP_NormalDataTransFormat[csvDataName]
				} else {
					dataName = csvDataName
				}
				//========RelatedData==========
				labelId := fmt.Sprint(snowflakeutil.NextID())
				//=============================
				var dataDetail = model.DataDetails{
					Name:               dataName,
					LabelName:          dataName,
					LabelId:            labelId,
					ContentDataDetails: []byte(`{}`),
				}

				var gatewayData = tolink_grpc.GatewayData{
					GatewayId: equipmentCode,
					Name:      dataName,
					DataId:    csvDataId,
					LabelId:   labelId,
					DataType:  0,
					Dbconfig:  -1,
				}

				equipment.DataDetails = append(equipment.DataDetails, dataDetail)
				gateway.GatewayDatas = append(gateway.GatewayDatas, &gatewayData)
			}
		}

		formatCount++
	}

	var successAddCount = 0
	var failAddCount = 0
	if len(commonDatas) > 0 {
		/**
		==========================
		== 调用远程过程 - 同步数据
		==========================
		*/
		for idx := range commonDatas {

			// 创建一个channel用于传输结果
			ch := make(chan bool)
			go callRemoteTolink(ctx, commonDatas[idx], ch)

			res := <-ch
			if !res {
				failAddCount++
				continue
			}

			successAddCount++
		}
	}

	// 返回导入结果
	response.Success(ctx, "导入完成",
		gin.H{"total": total - 1,
			"formatCount":     formatCount,
			"problemCount":    problemCount,
			"equipmentCount":  len(commonDatas),
			"successAddCount": successAddCount,
			"failAddCount":    failAddCount,
		})
}

// 访问远程过程 - 添加设备, 网关数据
func callRemoteTolink(ctx *gin.Context, commonData *dto.CommonDataDTO, ch chan bool) {
	res, err := client.AddBusinessTolinkCommonData(ctx, &commonData.Gateway, &commonData.Equipment, commonData.StatisticDataList)
	if !res || err != nil {
		log.Println("[BUSINESS-PLAT] ImportCommonData Call fail", err)
	}
	ch <- res
}

// 默认数据点添加
func defaultDataFromat(commondata *dto.CommonDataDTO) {

	for k, v := range dto.DP_DefaultNormalDataMap {

		//========RelatedData==========
		labelId := fmt.Sprint(snowflakeutil.NextID())
		//=============================
		var dataDetail = model.DataDetails{
			Name:               v,
			LabelName:          v,
			LabelId:            labelId,
			ContentDataDetails: []byte(`{}`),
		}

		var gatewayData = tolink_grpc.GatewayData{
			GatewayId: commondata.Gateway.GatewayId,
			Name:      v,
			DataId:    k,
			LabelId:   labelId,
			DataType:  0,
			Dbconfig:  -1,
		}

		commondata.Equipment.DataDetails = append(commondata.Equipment.DataDetails, dataDetail)
		commondata.Gateway.GatewayDatas = append(commondata.Gateway.GatewayDatas, &gatewayData)
	}

}

// 删除平台通用数据 - equipment/gateway/stream
func (ctrler AutoSyncAllCommonDataCtrler) DeleteCommonDataByEquipmentId(ctx *gin.Context) {

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	/* 校验信息 */

	if rEquipment.ID == 0 {
		log.Println("[BUSINESS-PLAT] equipment delete id is null")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 是否存在 - 查找关联数据
	equipmentDao := dao.NewEquipmentDAO(ctx)
	equipmentTem, res := equipmentDao.FindEquipmentById(rEquipment)
	if !res {
		log.Println("[BUSINESS-PLAT] equipment delete id is not exist")
		response.Fail(ctx, "删除失败", nil)
		return
	}
	var g = tolink_grpc.Gateway{
		GatewayId: equipmentTem.EquipmentCode,
		Name:      equipmentTem.EquipmentName,
	}

	// 关联删除数据, 开启事务
	res, err := client.DeleteBusinessTolinkCommonData(ctx, &g, &equipmentTem)
	if !res || err != nil {
		log.Println("[BUSINESS-PLAT] DeleteCommonData Call fail", err)
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 删除
	response.Success(ctx, "删除成功", nil)
}
