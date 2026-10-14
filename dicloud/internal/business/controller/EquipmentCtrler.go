package controller

import (
	authdao "dicloud/internal/auth/dao"
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

/* 站点管理 */
type IEquipmentCtrler interface {
	AddEquipment(ctx *gin.Context)
	FindEquipment(ctx *gin.Context)
	CountEquipmentByCorp(ctx *gin.Context)
	UpdateEquipment(ctx *gin.Context)
	DeleteEquipment(ctx *gin.Context)
	ExportExcel(ctx *gin.Context)
	ImportExcel(ctx *gin.Context)
	ImportTemplate(ctx *gin.Context)
	FindEquipmentAllData(ctx *gin.Context)
}

type EquipmentCtrler struct {
}

// 添加设备
func (r EquipmentCtrler) AddEquipment(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	/* 数据校验 */
	if len(rEquipment.EquipmentName) == 0 {
		response.Fail(ctx, "设备名称为空", nil)
		return
	}
	rEquipment.EquipmentTypeName = strings.TrimSpace(rEquipment.EquipmentTypeName)
	if len(rEquipment.EquipmentTypeName) == 0 {
		response.Fail(ctx, "电池类型不能为空", nil)
		return
	}
	rEquipment.EquipmentCode = strings.TrimSpace(rEquipment.EquipmentCode)
	equipmentCodeProvided := rEquipment.EquipmentCode != ""

	// 站点匹配
	if rEquipment.SiteId > 0 {
		siteDao := dao.NewSiteDAO(ctx)
		site, res := siteDao.FindOneSiteById(model.Site{ID: uint(rEquipment.SiteId)})
		if !res {
			response.Fail(ctx, "不存在的站点", nil)
			return
		}
		rEquipment.SiteName = site.SiteName
	} else {
		rEquipment.SiteName = "-"
	}

	/* warehouse */
	if len(rEquipment.WarehouseName) == 0 {
		rEquipment.WarehouseName = "-"
	}

	/* equipment type */

	/* 数据填充 */
	// if rEquipment.EquipmentSerialNumber == nil {
	// 	rEquipment.EquipmentSerialNumber = []byte(`{}`)
	// }
	if len(rEquipment.EquipmentCode) == 0 {
		rEquipment.EquipmentCode = "-"
	}
	if len(rEquipment.VideoProtoType) == 0 {
		rEquipment.VideoProtoType = "1"
	}
	if rEquipment.VideoUrls == nil {
		rEquipment.VideoUrls = pq.StringArray{}
	}
	if rEquipment.ConcernUserIds == nil {
		rEquipment.ConcernUserIds = pq.Int64Array{}
	}
	if rEquipment.ControlUserIds == nil {
		rEquipment.ControlUserIds = pq.Int64Array{}
	}
	if rEquipment.EzvizCloud == nil {
		rEquipment.EzvizCloud = []byte(`{}`)
	}
	if len(rEquipment.ApiKey) == 0 {
		rEquipment.ApiKey = "-"
	}
	if rEquipment.ApiKeys == nil {
		rEquipment.ApiKeys = []byte(`{}`)
	}
	if rEquipment.EquipmentDetails == nil {
		rEquipment.EquipmentDetails = []byte(`{}`)
	}
	if len(rEquipment.Manufacturer) == 0 {
		rEquipment.Manufacturer = "-"
	}
	if len(rEquipment.ManufacturerTime) == 0 {
		rEquipment.ManufacturerTime = "-"
	}
	if len(rEquipment.Batch) == 0 {
		rEquipment.Batch = "-"
	}
	if len(rEquipment.Specifications) == 0 {
		rEquipment.Specifications = "-"
	}
	if len(rEquipment.BmsManufacturer) == 0 {
		rEquipment.BmsManufacturer = "-"
	}
	if len(rEquipment.BmsVersion) == 0 {
		rEquipment.BmsVersion = "-"
	}

	// 创建人
	rEquipment.CreateUserId = int(userInfo.ID)

	/** 创建 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	// 同站点-重名校验
	if len(rEquipment.EquipmentName) > 0 {
		_, res := equipmentDao.EquipmentNameExist(rEquipment)
		if res {
			response.Fail(ctx, "已存在同名设备", nil)
			return
		}
	}
	if equipmentCodeProvided {
		_, res := equipmentDao.ExistEquipmentCode(rEquipment)
		if res {
			response.Fail(ctx, "已存在相同电池ID", nil)
			return
		}
	}
	eqm, res := equipmentDao.CreateEquipment(rEquipment)
	if !res {
		response.Fail(ctx, "添加失败", nil)
		return
	}

	response.Success(ctx, "添加成功", gin.H{"data": eqm})
}

// 查询设备
func (r EquipmentCtrler) FindEquipment(ctx *gin.Context) {

	var (
		pageStr     string
		pageSizeStr string
		filterFunc  string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)
	includeLatestCirculation := ctx.Query("includeLatestCirculation") == "1" || strings.EqualFold(ctx.Query("includeLatestCirculation"), "true")

	/* 鉴权查询 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	if userInfo.Isadmin == 1 {

		// 部门筛选
		if rEquipment.CorpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, int(rEquipment.CorpId))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rEquipment.CorpId = 0
			equipments, total, res := equipmentDao.FindEquipmentByCorp(rEquipment, corpIds, offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			respondEquipmentList(ctx, equipments, total, includeLatestCirculation)
			return
		} else {

			// 查询全部
			equipments, total, res := equipmentDao.FindEquipmentAll(rEquipment, offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			respondEquipmentList(ctx, equipments, total, includeLatestCirculation)
			return
		}

	} else {

		filterFunc = ctx.Query("filterFunc")

		/* 用户关注设备 - 查询 */
		if len(filterFunc) == 0 || filterFunc == "ConE" {
			// 默认: concern equipment

			// 鉴权查询 - 根据用户id查询
			equipments, total, res := equipmentDao.FindEquipmentByUser(rEquipment, int(userInfo.ID), offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			respondEquipmentList(ctx, equipments, total, includeLatestCirculation)
			return
		}

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		if filterFunc == "BinC" {
			// binded corp

			// 查询用户部门
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			// 鉴权部门
			if rEquipment.CorpId > 0 && len(corpIds) > 0 {
				idExist := -1
				for _, ID := range corpIds {
					if ID == rEquipment.CorpId {
						idExist = 1
						break
					}
				}
				if idExist == -1 {
					log.Println("[BUSINESS] not exist corp: ", rEquipment.CorpId)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}

				// 重新筛选
				corpIds = []int{rEquipment.CorpId}
				// 子部门查询
				corpIds = service.FindAllSonCorp(ctx, corpIds)
			}

			if len(corpIds) > 0 {

				// 根据部门 - 鉴权查寻
				equipments, total, res := equipmentDao.FindEquipmentByCorp(rEquipment, corpIds, offset, limit)
				if !res {
					response.Success(ctx, "success", nil)
					return
				}
				respondEquipmentList(ctx, equipments, total, includeLatestCirculation)
				return
			}

			response.Success(ctx, "success", gin.H{"total": 0})
			return
		}

		response.Fail(ctx, "无效筛选", nil)
	}

}

func respondEquipmentList(ctx *gin.Context, equipments []model.Equipment, total int64, includeLatestCirculation bool) {
	if includeLatestCirculation {
		equipmentIDs := make([]int, 0, len(equipments))
		for _, equipment := range equipments {
			equipmentIDs = append(equipmentIDs, int(equipment.ID))
		}

		latestLogs, ok := dao.NewEquipmentCirculationLogDao(ctx).FindLatestByEquipmentIDs(equipmentIDs)
		if !ok {
			response.Fail(ctx, "查询设备流转信息失败", nil)
			return
		}
		latestByEquipmentID := make(map[int]*model.EquipmentCirculationLog, len(latestLogs))
		for i := range latestLogs {
			latestByEquipmentID[latestLogs[i].EquipmentId] = &latestLogs[i]
		}
		for i := range equipments {
			equipments[i].LatestCirculationLog = latestByEquipmentID[int(equipments[i].ID)]
		}
	}

	response.Success(ctx, "success", gin.H{"data": equipments, "total": total})
}

// 计算部门设备数量
func (r EquipmentCtrler) CountEquipmentByCorp(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 部门
	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	var corpIdsParam = rEquipment.CorpList

	var (
		corpLists []int
		countList []int
	)

	/* 鉴权查询 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	if userInfo.Isadmin == 1 {

		if len(corpIdsParam) > 0 {

			for _, V := range corpIdsParam {
				var corpIds []int
				// 单独部门操作
				corpIds = append(corpIds, V)
				// 子部门查询
				corpIds = service.FindAllSonCorp(ctx, corpIds)

				countTem, res := equipmentDao.CountEquipmentByCorp(rEquipment, corpIds)
				if !res {
					response.Success(ctx, "success", nil)
					return
				}

				corpLists = append(corpLists, V)
				countList = append(countList, int(countTem))
			}
		} else {

			// 全部查询 - 全部门
			var corpIds []int
			corpIds = append(corpIds, 0)
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			dataList, res := equipmentDao.CountEquipmentGroupCorp(rEquipment, corpIds)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}

			if len(dataList) > 0 {
				for _, v := range dataList {
					corpLists = append(corpLists, v.CorpId)
					countList = append(countList, v.Count)
				}
			}
		}

	} else {

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		// binded corp

		if len(corpIdsParam) > 0 {

			// 查询用户部门
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			for _, vTem := range corpIdsParam {

				// 鉴权部门
				if len(corpIds) > 0 {
					idExist := -1
					for _, ID := range corpIds {
						if ID == vTem {
							idExist = 1
							break
						}
					}
					if idExist == -1 {
						log.Println("[BUSINESS] not exist corp: ", vTem)
						response.Fail(ctx, "非法部门, 未授权", nil)
						return
					}
				} else {
					log.Println("[BUSINESS] not boot corp for user: ", vTem)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}

				searchIds := service.FindAllSonCorp(ctx, []int{vTem})
				// 根据部门 - 鉴权查寻
				count, res := equipmentDao.CountEquipmentByCorp(rEquipment, searchIds)
				if !res {
					response.Success(ctx, "success", nil)
					return
				}

				corpLists = append(corpLists, vTem)
				countList = append(countList, int(count))
			}
		} else {
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			corpIds = service.FindAllSonCorp(ctx, corpIds)
			corpIds = append(corpIds, 0)
			corpIds = normalutil.Unique(corpIds)

			dataList, res := equipmentDao.CountEquipmentGroupCorp(rEquipment, corpIds)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			for _, v := range dataList {
				corpLists = append(corpLists, v.CorpId)
				countList = append(countList, v.Count)
			}
		}
	}

	response.Success(ctx, "查询成功", gin.H{"corp": corpLists, "count": countList})
}

// 更新设备
func (r EquipmentCtrler) UpdateEquipment(ctx *gin.Context) {

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	/* 校验信息 */

	if rEquipment.ID == 0 {
		log.Println("[BUSINESS] equipment update id is null")
		response.Fail(ctx, "修改失败", nil)
		return
	}
	rEquipment.EquipmentTypeName = strings.TrimSpace(rEquipment.EquipmentTypeName)
	rEquipment.EquipmentCode = strings.TrimSpace(rEquipment.EquipmentCode)
	if len(rEquipment.EquipmentTypeName) == 0 {
		response.Fail(ctx, "电池类型不能为空", nil)
		return
	}

	// 其他设备同名校验
	equipmentDao := dao.NewEquipmentDAO(ctx)
	if rEquipment.EquipmentCode != "" {
		existing, res := equipmentDao.ExistEquipmentCode(rEquipment)
		if res && existing.ID != rEquipment.ID {
			response.Fail(ctx, "已存在相同电池ID", nil)
			return
		}
	}
	if len(rEquipment.EquipmentName) > 0 {
		_, res := equipmentDao.EquipmentNameExist(rEquipment)
		if res {
			response.Fail(ctx, "已存在同名设备", nil)
			return
		}
	}

	s, b := equipmentDao.UpdateEquipment(rEquipment)
	if !b {
		response.Fail(ctx, "修改失败", nil)
		return
	}

	response.Success(ctx, "修改成功", gin.H{"data": s})
}

// 删除设备
func (r EquipmentCtrler) DeleteEquipment(ctx *gin.Context) {

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	/* 校验信息 */

	if rEquipment.ID == 0 {
		log.Println("[BUSINESS] equipment delete id is null")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 删除
	equipmentDao := dao.NewEquipmentDAO(ctx)
	b := equipmentDao.DeleteEquipment(rEquipment)
	if !b {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "删除成功", nil)
}

// 导出设备
func (r EquipmentCtrler) ExportExcel(ctx *gin.Context) {

	var (
		filterFunc string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	/* 鉴权查询 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	var equipments []model.Equipment
	if userInfo.Isadmin == 1 {

		// 部门筛选
		if rEquipment.CorpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, int(rEquipment.CorpId))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rEquipment.CorpId = 0
			equipments, _, _ = equipmentDao.FindEquipmentByCorp(rEquipment, corpIds, 0, 9999)

		} else {

			// 查询全部
			equipments, _, _ = equipmentDao.FindEquipmentAll(rEquipment, 0, 9999)
		}

	} else {

		filterFunc = ctx.Query("filterFunc")

		/* 用户关注设备 - 查询 */
		if len(filterFunc) == 0 || filterFunc == "ConE" {
			// 默认: concern equipment

			// 鉴权查询 - 根据用户id查询
			equipments, _, _ = equipmentDao.FindEquipmentByUser(rEquipment, int(userInfo.ID), 0, 9999)
		}

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		if filterFunc == "BinC" {
			// binded corp

			// 查询用户部门
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			// 鉴权部门
			if rEquipment.CorpId > 0 && len(corpIds) > 0 {
				idExist := -1
				for _, ID := range corpIds {
					if ID == rEquipment.CorpId {
						idExist = 1
						break
					}
				}
				if idExist == -1 {
					log.Println("[BUSINESS] not exist corp: ", rEquipment.CorpId)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}

				// 重新筛选
				corpIds = []int{rEquipment.CorpId}
				// 子部门查询
				corpIds = service.FindAllSonCorp(ctx, corpIds)
			}

			// 根据部门 - 鉴权查寻
			equipments, _, _ = equipmentDao.FindEquipmentByCorp(rEquipment, corpIds, 0, 9999)
		}
	}

	/* 导出 */
	if len(equipments) > 0 {

		// 全部门ID对应名称
		var corpMap = make(map[int]string)
		corpDao := authdao.NewCorpDAO(ctx)
		corps, _ := corpDao.FindCorpList(usermodel.Corp{}, 1)
		if len(corps) > 0 {
			for _, v := range corps {
				corpMap[int(v.ID)] = v.CorpName
			}
		}

		// 开始导出数据
		f := excelize.NewFile()
		defer f.Close()
		index := f.GetActiveSheetIndex()

		for i, v := range model.EquipmentExportHeader {
			//rune('A'+i)：根据当前索引 i 计算出 Excel 表中的列名。rune('A') 是字符 'A' 的 Unicode 码点值，加上索引 i 后将其转换回字符，从而得到列名。例如，当 i=0 时，结果是 'A'；当 i=1 时，结果是 'B'，依此类推。
			cell := fmt.Sprintf("%s1", string(rune('A'+i)))
			f.SetCellValue("Sheet1", cell, v)
		}

		for idx, v := range equipments {
			f.SetCellValue("Sheet1", fmt.Sprintf("A%d", idx+2), v.EquipmentCode)
			f.SetCellValue("Sheet1", fmt.Sprintf("B%d", idx+2), v.EquipmentTypeName)
			f.SetCellValue("Sheet1", fmt.Sprintf("C%d", idx+2), v.Manufacturer)
			f.SetCellValue("Sheet1", fmt.Sprintf("D%d", idx+2), v.ManufacturerTime)
			f.SetCellValue("Sheet1", fmt.Sprintf("E%d", idx+2), v.Batch)
			f.SetCellValue("Sheet1", fmt.Sprintf("F%d", idx+2), v.Specifications)

			// 部门
			if len(corpMap[v.CorpId]) > 0 {
				f.SetCellValue("Sheet1", fmt.Sprintf("G%d", idx+2), corpMap[v.CorpId])
			} else {
				f.SetCellValue("Sheet1", fmt.Sprintf("G%d", idx+2), v.CorpId)
			}

			f.SetCellValue("Sheet1", fmt.Sprintf("H%d", idx+2), v.BmsManufacturer)
			f.SetCellValue("Sheet1", fmt.Sprintf("I%d", idx+2), v.BmsVersion)
		}

		f.SetActiveSheet(index)
		writeEquipmentWorkbook(ctx, "电池出厂信息.xlsx", f, 200, "导出成功")
		return
	}

	response.Fail(ctx, "导出失败", nil)
}

// ImportTemplate 直接返回电池出厂信息导入模板。
func (ctrler EquipmentCtrler) ImportTemplate(ctx *gin.Context) {
	f := excelize.NewFile()
	defer f.Close()
	for i, title := range model.EquipmentExportHeader {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		if err := f.SetCellValue("Sheet1", cell, title); err != nil {
			response.Fail(ctx, "生成导入模板失败", nil)
			return
		}
	}
	textStyle, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err == nil {
		_ = f.SetCellStyle("Sheet1", "A2", "I1001", textStyle)
	}
	_ = f.SetColWidth("Sheet1", "A", "I", 18)
	if err := addImportTemplateExamples(f, model.EquipmentExportHeader, [][]string{
		{"BAT000001", "锂电池", "示例生产厂家", "2026-01-01", "PC202601", "48V100Ah", "请填写已有部门名称", "示例BMS厂家", "V1.0"},
		{"BAT000002", "钠电池", "示例生产厂家", "2026-01-02", "PC202602", "48V100Ah", "请填写已有部门名称", "示例BMS厂家", "V1.1"},
	}, []string{
		"填写说明：红色内容仅为示例，请将真实数据填写到第一个“Sheet1”工作表，从第2行开始；本页不参与导入。",
		"电池ID、电池类型不能为空；电池ID不能与文件中其他行或平台已有电池重复。所属公司请填写平台中已有的完整部门名称。",
		"请保留表头及列顺序；电池ID按文本填写，生产日期建议使用年-月-日格式，例如2026-01-01，其余字段按实际出厂信息填写。",
	}); err != nil {
		response.Fail(ctx, "生成导入模板失败", nil)
		return
	}
	writeEquipmentWorkbook(ctx, "电池出厂信息导入模板.xlsx", f, 200, "导入模板下载成功")
}

// 导入Excel数据
func (ctrler EquipmentCtrler) ImportExcel(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 获取上传文件
	file, fileHeader, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Println("[BUSINESS] upload file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer file.Close()
	if fileHeader.Size <= 0 || fileHeader.Size > 5*1024*1024 || !strings.EqualFold(filepath.Ext(fileHeader.Filename), ".xlsx") {
		response.Fail(ctx, "请上传不超过5MB的xlsx文件", nil)
		return
	}

	f, err := excelize.OpenReader(file)
	if err != nil {
		log.Println("[BUSINESS] excelize open file fail", err)
		response.Fail(ctx, "无法打开文件", nil)
		return
	}

	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		response.Fail(ctx, "Excel中不存在工作表", nil)
		return
	}
	rows, err := f.GetRows(sheets[0])
	if err != nil {
		response.Fail(ctx, "获取表数据失败", nil)
		return
	}

	// 全部门ID对应名称
	var corpMap = make(map[string]int)
	corpDao := authdao.NewCorpDAO(ctx)
	corps, _ := corpDao.FindCorpList(usermodel.Corp{}, 1)
	if len(corps) > 0 {
		for _, v := range corps {
			corpMap[v.CorpName] = int(v.ID)
		}
	}

	parsedEquipments, issues := validateEquipmentImportRows(rows, corpMap)
	equipmentDao := dao.NewEquipmentDAO(ctx)
	issues = append(issues, equipmentImportExistingCodeIssues(parsedEquipments, rows, func(code string) bool {
		_, exists := equipmentDao.ExistEquipmentCode(model.Equipment{EquipmentCode: code})
		return exists
	})...)
	if len(issues) > 0 {
		if err := writeEquipmentImportIssues(f, sheets[0], issues); err != nil {
			log.Println("[BUSINESS] create equipment import validation file fail", err)
			response.Fail(ctx, "生成校验失败文件出错", nil)
			return
		}
		ctx.Header("X-Import-Validation-Failed", "true")
		ctx.Header("X-Import-Error-Count", strconv.Itoa(len(issues)))
		writeEquipmentWorkbook(ctx, equipmentImportErrorFilename(fileHeader.Filename), f, 400, "导入校验失败")
		return
	}

	equipments := make([]model.Equipment, 0, len(parsedEquipments))
	for _, equipment := range parsedEquipments {
		equipment.CreateUserId = int(userInfo.ID)
		equipment.SiteName = "-"
		baseEquipmentDataFill(&equipment)
		equipments = append(equipments, equipment)
	}

	// 保存
	if len(equipments) > 0 {
		dataList, res := equipmentDao.ImportEquipments(equipments, nil)
		if !res {
			response.Fail(ctx, "导入失败", nil)
			return
		}
		response.Success(ctx, "导入成功",
			gin.H{"data": dataList, "successCount": len(parsedEquipments),
				"dumplicateCount": 0, "failCount": 0})
		return
	}

	response.Fail(ctx, "导入失败", nil)
}

type equipmentImportIssue struct {
	Row     int
	Message string
}

func validateEquipmentImportRows(rows [][]string, corpMap map[string]int) ([]model.Equipment, []equipmentImportIssue) {
	if len(rows) == 0 {
		return nil, []equipmentImportIssue{{Row: 1, Message: "Excel不能为空"}}
	}
	for index, title := range model.EquipmentExportHeader {
		if strings.TrimSpace(equipmentImportCell(rows[0], index)) != title {
			return nil, []equipmentImportIssue{{Row: 1, Message: "表头必须为：" + strings.Join(model.EquipmentExportHeader, "、")}}
		}
	}

	equipments := make([]model.Equipment, 0, len(rows)-1)
	issues := make([]equipmentImportIssue, 0)
	seenCodes := make(map[string]int)
	for index, row := range rows[1:] {
		rowNumber := index + 2
		if equipmentImportRowEmpty(row) {
			continue
		}
		equipmentCode := strings.TrimSpace(equipmentImportCell(row, 0))
		equipmentTypeName := strings.TrimSpace(equipmentImportCell(row, 1))
		rowIssues := make([]string, 0, 2)
		if equipmentCode == "" {
			rowIssues = append(rowIssues, "电池ID不能为空")
		} else if firstRow, exists := seenCodes[equipmentCode]; exists {
			rowIssues = append(rowIssues, fmt.Sprintf("电池ID与第%d行重复", firstRow))
		} else {
			seenCodes[equipmentCode] = rowNumber
		}
		if equipmentTypeName == "" {
			rowIssues = append(rowIssues, "电池类型不能为空")
		}
		if len(rowIssues) > 0 {
			issues = append(issues, equipmentImportIssue{Row: rowNumber, Message: strings.Join(rowIssues, "；")})
			continue
		}

		corpName := strings.TrimSpace(equipmentImportCell(row, 6))
		equipments = append(equipments, model.Equipment{
			EquipmentName:     equipmentCode,
			EquipmentCode:     equipmentCode,
			EquipmentTypeName: equipmentTypeName,
			Manufacturer:      strings.TrimSpace(equipmentImportCell(row, 2)),
			ManufacturerTime:  strings.TrimSpace(equipmentImportCell(row, 3)),
			Batch:             strings.TrimSpace(equipmentImportCell(row, 4)),
			Specifications:    strings.TrimSpace(equipmentImportCell(row, 5)),
			CorpId:            corpMap[corpName],
			BmsManufacturer:   strings.TrimSpace(equipmentImportCell(row, 7)),
			BmsVersion:        strings.TrimSpace(equipmentImportCell(row, 8)),
		})
	}
	if len(equipments) == 0 && len(issues) == 0 {
		issues = append(issues, equipmentImportIssue{Row: 2, Message: "Excel中没有可导入的数据"})
	}
	return equipments, issues
}

func equipmentImportExistingCodeIssues(equipments []model.Equipment, rows [][]string, codeExists func(string) bool) []equipmentImportIssue {
	rowNumbers := make(map[string]int, len(equipments))
	for index, row := range rows[1:] {
		code := strings.TrimSpace(equipmentImportCell(row, 0))
		if code != "" {
			rowNumbers[code] = index + 2
		}
	}

	issues := make([]equipmentImportIssue, 0)
	for _, equipment := range equipments {
		if codeExists(equipment.EquipmentCode) {
			issues = append(issues, equipmentImportIssue{
				Row:     rowNumbers[equipment.EquipmentCode],
				Message: "电池ID已存在，禁止导入",
			})
		}
	}
	return issues
}

func equipmentImportCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func equipmentImportRowEmpty(row []string) bool {
	for index := range model.EquipmentExportHeader {
		if strings.TrimSpace(equipmentImportCell(row, index)) != "" {
			return false
		}
	}
	return true
}

func writeEquipmentImportIssues(f *excelize.File, sheet string, issues []equipmentImportIssue) error {
	const errorColumn = 10
	headerCell, _ := excelize.CoordinatesToCellName(errorColumn, 1)
	if err := f.SetCellValue(sheet, headerCell, "校验错误信息"); err != nil {
		return err
	}
	if styleID, err := f.GetCellStyle(sheet, "I1"); err == nil && styleID > 0 {
		_ = f.SetCellStyle(sheet, headerCell, headerCell, styleID)
	}
	errorStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "FF0000"},
		Alignment: &excelize.Alignment{Vertical: "top", WrapText: true},
		NumFmt:    49,
	})
	if err != nil {
		return err
	}
	for _, issue := range issues {
		row := issue.Row
		// Workbook-level errors use row 2 so the error-column header remains intact.
		if row < 2 {
			row = 2
		}
		cell, _ := excelize.CoordinatesToCellName(errorColumn, row)
		if err := f.SetCellValue(sheet, cell, issue.Message); err != nil {
			return err
		}
		if err := f.SetCellStyle(sheet, cell, cell, errorStyle); err != nil {
			return err
		}
	}
	return f.SetColWidth(sheet, "J", "J", 48)
}

func equipmentImportErrorFilename(originalFilename string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(originalFilename)), filepath.Ext(originalFilename))
	if base == "" || base == "." {
		base = "电池出厂信息导入"
	}
	return base + "_校验失败.xlsx"
}

func writeEquipmentWorkbook(ctx *gin.Context, fileName string, f *excelize.File, responseCode int, responseMessage string) {
	ctx.Set(response.ContextResponseHTTPStatus, http.StatusOK)
	ctx.Set(response.ContextResponseCode, responseCode)
	ctx.Set(response.ContextResponseMsg, responseMessage)
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="equipment.xlsx"; filename*=UTF-8''%s`, url.PathEscape(fileName)))
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition, X-Import-Validation-Failed, X-Import-Error-Count")
	ctx.Status(http.StatusOK)
	if err := f.Write(ctx.Writer); err != nil {
		log.Println("[BUSINESS] write equipment workbook fail", err)
	}
}

// 基础数据填充
func baseEquipmentDataFill(rEquipment *model.Equipment) {

	if len(rEquipment.EquipmentTypeName) == 0 {
		rEquipment.EquipmentTypeName = "-"
	}
	if len(rEquipment.WarehouseName) == 0 {
		rEquipment.WarehouseName = "-"
	}
	if len(rEquipment.EquipmentCode) == 0 {
		rEquipment.EquipmentCode = "-"
	}
	if len(rEquipment.VideoProtoType) == 0 {
		rEquipment.VideoProtoType = "1"
	}
	if rEquipment.VideoUrls == nil {
		rEquipment.VideoUrls = pq.StringArray{}
	}
	if rEquipment.ConcernUserIds == nil {
		rEquipment.ConcernUserIds = pq.Int64Array{}
	}
	if rEquipment.ControlUserIds == nil {
		rEquipment.ControlUserIds = pq.Int64Array{}
	}
	if rEquipment.EzvizCloud == nil {
		rEquipment.EzvizCloud = []byte(`{}`)
	}
	if len(rEquipment.ApiKey) == 0 {
		rEquipment.ApiKey = "-"
	}
	if rEquipment.ApiKeys == nil {
		rEquipment.ApiKeys = []byte(`{}`)
	}
	if rEquipment.EquipmentDetails == nil {
		rEquipment.EquipmentDetails = []byte(`{}`)
	}
	if len(rEquipment.Manufacturer) == 0 {
		rEquipment.Manufacturer = "-"
	}
	if len(rEquipment.ManufacturerTime) == 0 {
		rEquipment.ManufacturerTime = "-"
	}
	if len(rEquipment.Batch) == 0 {
		rEquipment.Batch = "-"
	}
	if len(rEquipment.Specifications) == 0 {
		rEquipment.Specifications = "-"
	}
	if len(rEquipment.BmsManufacturer) == 0 {
		rEquipment.BmsManufacturer = "-"
	}
	if len(rEquipment.BmsVersion) == 0 {
		rEquipment.BmsVersion = "-"
	}

}

// 查询设备全部数据 - 带数据点数据
func (r EquipmentCtrler) FindEquipmentAllData(ctx *gin.Context) {

	var (
		pageStr     string
		pageSizeStr string
		filterFunc  string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rEquipment model.Equipment
	ctx.Bind(&rEquipment)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	/* 鉴权查询 */
	equipmentDao := dao.NewEquipmentDAO(ctx)
	if userInfo.Isadmin == 1 {

		// 部门筛选
		if rEquipment.CorpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, int(rEquipment.CorpId))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rEquipment.CorpId = 0
			equipments, total, res := equipmentDao.FindEquipmentAllDataByCorp(rEquipment, corpIds, offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": equipments, "total": total})
			return
		} else {

			// 查询全部
			equipments, total, res := equipmentDao.FindEquipmentAllDataAll(rEquipment, offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": equipments, "total": total})
			return
		}

	} else {

		filterFunc = ctx.Query("filterFunc")

		/* 用户关注设备 - 查询 */
		if len(filterFunc) == 0 || filterFunc == "ConE" {
			// 默认: concern equipment

			// 鉴权查询 - 根据用户id查询
			equipments, total, res := equipmentDao.FindEquipmentAllDataByUser(rEquipment, int(userInfo.ID), offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": equipments, "total": total})
			return
		}

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		if filterFunc == "BinC" {
			// binded corp

			// 查询用户部门
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			// 鉴权部门
			if rEquipment.CorpId > 0 && len(corpIds) > 0 {
				idExist := -1
				for _, ID := range corpIds {
					if ID == rEquipment.CorpId {
						idExist = 1
						break
					}
				}
				if idExist == -1 {
					log.Println("[BUSINESS] not exist corp: ", rEquipment.CorpId)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}

				// 重新筛选
				corpIds = []int{rEquipment.CorpId}
				// 子部门查询
				corpIds = service.FindAllSonCorp(ctx, corpIds)
			}

			if len(corpIds) > 0 {

				// 根据部门 - 鉴权查寻
				equipments, total, res := equipmentDao.FindEquipmentAllDataByCorp(rEquipment, corpIds, offset, limit)
				if !res {
					response.Success(ctx, "success", nil)
					return
				}
				response.Success(ctx, "success", gin.H{"data": equipments, "total": total})
				return
			}

			response.Success(ctx, "success", gin.H{"total": 0})
			return
		}

		response.Fail(ctx, "无效筛选", nil)
	}
}
