package controller

import (
	authdao "dicloud/internal/auth/dao"
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	normalutil "dicloud/util/normal"
	"dicloud/util/response"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

/* 站点管理 */
type ISiteCtrler interface {
	AddSite(ctx *gin.Context)
	DeleteSite(ctx *gin.Context)
	UpdateSite(ctx *gin.Context)
	FindSite(ctx *gin.Context)
	FindSiteDetail(ctx *gin.Context)
	ExportExcel(ctx *gin.Context)
	ImportExcel(ctx *gin.Context)
	ImportTemplate(ctx *gin.Context)
}

type SiteCtrler struct {
}

// 保存站点
func (r SiteCtrler) AddSite(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rSite model.Site
	ctx.Bind(&rSite)
	rSite.SiteName = strings.TrimSpace(rSite.SiteName)
	rSite.SiteSerialNumber = strings.TrimSpace(rSite.SiteSerialNumber)

	/* 校验信息 */
	if len(rSite.SiteName) == 0 {
		response.Fail(ctx, "站点名、站点APIKey、站点所属分组不能为空", nil)
		return
	}
	if rSite.SiteSerialNumber == "" {
		response.Fail(ctx, "站址编码不能为空", nil)
		return
	}
	if rSite.ApiKeys == nil {
		rSite.ApiKeys = []byte(`{}`)
	}
	if rSite.SiteDetails == nil {
		rSite.SiteDetails = []byte(`{}`)
	}
	rSite.CreateUserId = int(userInfo.ID)

	siteDao := dao.NewSiteDAO(ctx)
	_, res := siteDao.SiteNameExist(rSite)
	if res {
		response.Fail(ctx, "已存在同名站点", nil)
		return
	}
	_, res = siteDao.ExistSiteSearial(rSite)
	if res {
		response.Fail(ctx, "站址编码已存在，不能添加", nil)
		return
	}

	// 创建
	site, res := siteDao.CreateSite(rSite)
	if !res {
		response.Fail(ctx, "创建失败", nil)
		return
	}
	fmt.Println(site)

	response.Success(ctx, "创建成功", gin.H{"data": site})
}

func baseSiteDataFill(rSite *model.Site) {

	if len(rSite.ApiKey) == 0 {
		rSite.ApiKey = "-"
	}
	if rSite.ApiKeys == nil {
		rSite.ApiKeys = []byte(`{}`)
	}
	if rSite.SiteDetails == nil {
		rSite.SiteDetails = []byte(`{}`)
	}
	if len(rSite.SiteGroupName) == 0 {
		rSite.SiteGroupName = "-"
	}

}

// 删除站点
func (r SiteCtrler) DeleteSite(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rSite model.Site
	ctx.Bind(&rSite)

	/* 校验信息 */

	if rSite.ID == 0 {
		log.Println("[BUSINESS] site delete id is null")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 是否有关联设备
	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentBySiteId(int(rSite.ID), 1, 10)
	if res {
		response.Fail(ctx, "请先删除关联设备", nil)
		return
	}

	// 删除
	siteDao := dao.NewSiteDAO(ctx)
	b := siteDao.DeleteSite(rSite)
	if !b {
		response.Fail(ctx, "删除失败", nil)
		return
	}

	response.Success(ctx, "删除成功", nil)
}

// 更新站点
func (r SiteCtrler) UpdateSite(ctx *gin.Context) {

	/* 获取信息 */

	var rSite model.Site
	ctx.Bind(&rSite)

	/* 校验信息 */

	if rSite.ID == 0 {
		log.Println("[BUSINESS] site update id is null")
		response.Fail(ctx, "修改失败", nil)
		return
	}
	rSite.SiteName = strings.TrimSpace(rSite.SiteName)
	rSite.SiteSerialNumber = strings.TrimSpace(rSite.SiteSerialNumber)
	if rSite.SiteName == "" {
		response.Fail(ctx, "站点名称不能为空", nil)
		return
	}
	if rSite.SiteSerialNumber == "" {
		response.Fail(ctx, "站址编码不能为空", nil)
		return
	}

	// 其他站点同名校验
	siteDao := dao.NewSiteDAO(ctx)
	_, res := siteDao.SiteNameExist(rSite)
	if res {
		response.Fail(ctx, "已存在同名站点", nil)
		return
	}
	_, res = siteDao.ExistSiteSearial(rSite)
	if res {
		response.Fail(ctx, "站址编码已存在，不能修改", nil)
		return
	}

	s, b := siteDao.UpdateSite(rSite)
	if !b {
		response.Fail(ctx, "修改失败", nil)
		return
	}

	response.Success(ctx, "修改成功", gin.H{"data": s})
}

// 查询站点
func (r SiteCtrler) FindSite(ctx *gin.Context) {

	var (
		pageStr     string
		pageSizeStr string
		filterFunc  string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rSite model.Site
	ctx.Bind(&rSite)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	// fmt.Println("pageNum = ", pageStr, ", pageSize = ", pageSizeStr)
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	/* 鉴权查询 */
	siteDao := dao.NewSiteDAO(ctx)
	if userInfo.Isadmin == 1 {

		// 部门筛选
		if rSite.CorpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, int(rSite.CorpId))
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rSite.CorpId = 0
			// 查询全部数据
			sites, res := siteDao.FindSiteByCorp(rSite, corpIds, offset, limit)
			if !res {
				response.Fail(ctx, "查询站址失败", nil)
				return
			}
			total, res := siteDao.CountSiteByCorp(rSite, corpIds)
			if !res {
				response.Fail(ctx, "统计站址数量失败", nil)
				return
			}

			response.Success(ctx, "success", gin.H{"data": sites, "total": total})
			return
		} else {

			// 查询全部数据
			sites, res := siteDao.FindSiteAll(rSite, offset, limit)
			if !res {
				response.Fail(ctx, "查询站址失败", nil)
				return
			}
			total, res := siteDao.CountSiteAll(rSite)
			if !res {
				response.Fail(ctx, "统计站址数量失败", nil)
				return
			}

			response.Success(ctx, "success", gin.H{"data": sites, "total": total})
			return
		}
	} else {

		/* 区分是哪种查询 1: 根据用户关注设备查询 2: 根据用户绑定部门查询 */
		filterFunc = ctx.Query("filterFunc")

		/* TODO: 用户部门 查询; 用户关注 查询? TODO: 测试int数组查询模式(finished!) TODO: 通过用户查询关注设备id*/
		/* 用户关注设备 - 查询 */
		if len(filterFunc) == 0 || filterFunc == "ConE" {
			// 默认: concern equipment

			// 根据设备关注用户 - 查询用户关注设备
			equipmentDao := dao.NewEquipmentDAO(ctx)
			eqs, _, _ := equipmentDao.FindEquipmentByUser(model.Equipment{}, int(userInfo.ID), offset, limit)
			var siteIds []int
			for _, v := range eqs {
				siteIds = append(siteIds, int(v.SiteId))
			}
			siteIds = normalutil.Unique(siteIds)

			// 鉴权查寻
			sites, res := siteDao.FindSite(rSite, siteIds, offset, limit)
			if !res {
				response.Fail(ctx, "查询站址失败", nil)
				return
			}
			total, res := siteDao.CountSite(rSite, siteIds)
			if !res {
				response.Fail(ctx, "统计站址数量失败", nil)
				return
			}

			response.Success(ctx, "success", gin.H{"data": sites, "total": total})
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
			if rSite.CorpId > 0 && len(corpIds) > 0 {
				idExist := -1
				for _, ID := range corpIds {
					if ID == rSite.CorpId {
						idExist = 1
						break
					}
				}
				if idExist == -1 {
					log.Println("[BUSINESS] not exist corp: ", rSite.CorpId)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}

				// 重新筛选
				corpIds = []int{rSite.CorpId}
				// 子部门查询
				corpIds = service.FindAllSonCorp(ctx, corpIds)
			}

			if len(corpIds) > 0 {
				// 根据部门 - 鉴权查寻
				sites, res := siteDao.FindSiteByCorp(rSite, corpIds, offset, limit)
				if !res {
					fmt.Println("[BUSINESS] search site by corpIds fail")
					response.Fail(ctx, "查询站址失败", nil)
					return
				}
				total, res := siteDao.CountSiteByCorp(rSite, corpIds)
				if !res {
					response.Fail(ctx, "统计站址数量失败", nil)
					return
				}

				response.Success(ctx, "success", gin.H{"data": sites, "total": total})
				return
			}

			response.Success(ctx, "success", gin.H{"data": []model.Site{}, "total": 0})
			return
		}

		response.Fail(ctx, "无效筛选", nil)
	}

}

func (r SiteCtrler) FindSiteDetail(ctx *gin.Context) {

	/* 获取用户信息 */

	// userInfoCache, _ := ctx.Get("userInfo")
	// userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rSite model.Site
	ctx.Bind(&rSite)

	siteDao := dao.NewSiteDAO(ctx)
	sites, res := siteDao.FindSiteDetail(rSite)
	if res {
		fmt.Println(sites)
		if len(sites[0].SiteDetails) > 0 {
			data := make(map[string]any)
			json.Unmarshal([]byte(sites[0].SiteDetails), &data)
			// fmt.Println("line 103 json: ", data["longitude"])
			// fmt.Println("line 104 json: ", string(sites[0].SiteDetails))
		}
	}

	// response.Success(ctx, "success", gin.H{"data": sites})
	ctx.JSON(200, sites)
}

// 导出站点
func (r SiteCtrler) ExportExcel(ctx *gin.Context) {

	var (
		filterFunc string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rSite model.Site
	ctx.Bind(&rSite)

	/* 鉴权查询 */
	siteDao := dao.NewSiteDAO(ctx)
	var sites []model.Site
	if userInfo.Isadmin == 1 {

		// 部门筛选
		if rSite.CorpId > 0 {
			var corpIds []int
			// 单独部门操作
			corpIds = append(corpIds, int(rSite.CorpId))
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rSite.CorpId = 0
			// 查询全部数据
			sites, _ = siteDao.FindSiteByCorp(rSite, corpIds, 0, 9999)
		} else {

			// 查询全部数据
			sites, _ = siteDao.FindSiteAll(rSite, 0, 9999)
		}
	} else {

		/* 区分是哪种查询 1: 根据用户关注设备查询 2: 根据用户绑定部门查询 */
		filterFunc = ctx.Query("filterFunc")

		/* TODO: 用户部门 查询; 用户关注 查询? TODO: 测试int数组查询模式(finished!) TODO: 通过用户查询关注设备id*/
		/* 用户关注设备 - 查询 */
		if len(filterFunc) == 0 || filterFunc == "ConE" {
			// 默认: concern equipment

			// 根据设备关注用户 - 查询用户关注设备
			equipmentDao := dao.NewEquipmentDAO(ctx)
			eqs, _, _ := equipmentDao.FindEquipmentByUser(model.Equipment{}, int(userInfo.ID), 0, 9999)
			var siteIds []int
			for _, v := range eqs {
				siteIds = append(siteIds, int(v.SiteId))
			}
			siteIds = normalutil.Unique(siteIds)

			// 鉴权查寻
			sites, _ = siteDao.FindSite(rSite, siteIds, 0, 9999)
		}

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		if filterFunc == "BinC" {

			// 查询用户部门
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			// 鉴权部门
			if rSite.CorpId > 0 && len(corpIds) > 0 {
				idExist := -1
				for _, ID := range corpIds {
					if ID == rSite.CorpId {
						idExist = 1
						break
					}
				}
				if idExist == -1 {
					log.Println("[BUSINESS] not exist corp: ", rSite.CorpId)
					response.Fail(ctx, "非法部门, 未授权", nil)
					return
				}

				// 重新筛选
				corpIds = []int{rSite.CorpId}
				// 子部门查询
				corpIds = service.FindAllSonCorp(ctx, corpIds)
			}

			if len(corpIds) > 0 {
				// 根据部门 - 鉴权查寻
				sites, _ = siteDao.FindSiteByCorp(rSite, corpIds, 0, 9999)
			}

		}
	}

	/* 导出 */
	if len(sites) > 0 {

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

		for i, v := range model.SiteExportHeader {
			//rune('A'+i)：根据当前索引 i 计算出 Excel 表中的列名。rune('A') 是字符 'A' 的 Unicode 码点值，加上索引 i 后将其转换回字符，从而得到列名。例如，当 i=0 时，结果是 'A'；当 i=1 时，结果是 'B'，依此类推。
			cell := fmt.Sprintf("%s1", string(rune('A'+i)))
			f.SetCellValue("Sheet1", cell, v)
		}

		for idx, v := range sites {
			f.SetCellValue("Sheet1", fmt.Sprintf("A%d", idx+2), v.SiteName)
			f.SetCellValue("Sheet1", fmt.Sprintf("B%d", idx+2), v.SiteSerialNumber)

			// 部门
			if len(corpMap[v.CorpId]) > 0 {
				f.SetCellValue("Sheet1", fmt.Sprintf("C%d", idx+2), corpMap[v.CorpId])
			} else {
				f.SetCellValue("Sheet1", fmt.Sprintf("C%d", idx+2), v.CorpId)
			}

			var mapTem = make(map[string]any)
			bytesData, _ := v.SiteDetails.MarshalJSON()
			json.Unmarshal(bytesData, &mapTem)

			f.SetCellValue("Sheet1", fmt.Sprintf("D%d", idx+2), mapTem["addr"])
			f.SetCellValue("Sheet1", fmt.Sprintf("E%d", idx+2), mapTem["longitude"])
			f.SetCellValue("Sheet1", fmt.Sprintf("F%d", idx+2), mapTem["latitude"])
		}

		f.SetActiveSheet(index)

		writeSiteWorkbook(ctx, "站点信息.xlsx", f, http.StatusOK, "导出成功")
		return
	}

	response.Fail(ctx, "导出失败", nil)
}

func writeSiteWorkbook(ctx *gin.Context, fileName string, f *excelize.File, responseCode int, responseMessage string) {
	ctx.Set(response.ContextResponseHTTPStatus, http.StatusOK)
	ctx.Set(response.ContextResponseCode, responseCode)
	ctx.Set(response.ContextResponseMsg, responseMessage)
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="site.xlsx"; filename*=UTF-8''%s`, url.PathEscape(fileName)))
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition, X-Import-Validation-Failed, X-Import-Error-Count")
	ctx.Status(http.StatusOK)
	if err := f.Write(ctx.Writer); err != nil {
		log.Println("[BUSINESS] write site workbook fail", err)
	}
}

// 导入Excel数据
func (ctrler SiteCtrler) ImportExcel(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 获取上传文件
	file, fileHeader, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Println("[BUSINESS-Site] upload file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		log.Println("[BUSINESS-Site] excelize open file fail", err)
		response.Fail(ctx, "无法打开文件", nil)
		return
	}

	defer f.Close()

	// 获取第一个工作表的所有行
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

	siteDao := dao.NewSiteDAO(ctx)
	parsedSites, issues := validateSiteImportRows(rows, corpMap)
	issues = append(issues, siteImportExistingNameIssues(parsedSites, rows, func(name string) bool {
		_, exists := siteDao.SiteNameExist(model.Site{SiteName: name})
		return exists
	})...)
	issues = append(issues, siteImportExistingCodeIssues(parsedSites, rows, func(code string) bool {
		_, exists := siteDao.ExistSiteSearial(model.Site{SiteSerialNumber: code})
		return exists
	})...)
	issues = mergeSiteImportIssues(issues)
	if len(issues) > 0 {
		if err := writeSiteImportIssues(f, sheets[0], issues); err != nil {
			log.Println("[BUSINESS] create site import validation file fail", err)
			response.Fail(ctx, "生成校验失败文件出错", nil)
			return
		}
		ctx.Header("X-Import-Validation-Failed", "true")
		ctx.Header("X-Import-Error-Count", strconv.Itoa(len(issues)))
		writeSiteWorkbook(ctx, siteImportErrorFilename(fileHeader.Filename), f, 400, "导入校验失败")
		return
	}

	sites := make([]model.Site, 0, len(parsedSites))
	for _, site := range parsedSites {
		baseSiteDataFill(&site)
		site.CreateUserId = int(userInfo.ID)
		sites = append(sites, site)
	}

	// 保存
	if len(sites) > 0 {
		dataList, res := siteDao.ImportSites(sites, nil)
		if !res {
			response.Fail(ctx, "导入失败", nil)
			return
		}
		response.Success(ctx, "导入成功",
			gin.H{"data": dataList, "successCount": len(sites),
				"dumplicateCount": 0, "failCount": 0})
		return
	}

	response.Fail(ctx, "导入失败", nil)
}

type siteImportIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

func validateSiteImportRows(rows [][]string, corpMap map[string]int) ([]model.Site, []siteImportIssue) {
	if len(rows) == 0 {
		return nil, []siteImportIssue{{Row: 1, Message: "Excel不能为空"}}
	}
	for index, title := range model.SiteExportHeader {
		if strings.TrimSpace(siteImportCell(rows[0], index)) != title {
			return nil, []siteImportIssue{{Row: 1, Message: "表头必须为：" + strings.Join(model.SiteExportHeader, "、")}}
		}
	}

	sites := make([]model.Site, 0, len(rows)-1)
	issues := make([]siteImportIssue, 0)
	seenNames := make(map[string]int)
	seenCodes := make(map[string]int)
	for index, row := range rows[1:] {
		rowNumber := index + 2
		if siteImportRowEmpty(row) {
			continue
		}
		name := strings.TrimSpace(siteImportCell(row, 0))
		code := strings.TrimSpace(siteImportCell(row, 1))
		rowIssues := make([]string, 0, 3)
		if name == "" {
			rowIssues = append(rowIssues, "站点名称不能为空")
		} else if firstRow, exists := seenNames[name]; exists {
			rowIssues = append(rowIssues, fmt.Sprintf("站点名称与第%d行重复", firstRow))
		} else {
			seenNames[name] = rowNumber
		}
		if code == "" {
			rowIssues = append(rowIssues, "站址编码不能为空")
		} else if firstRow, exists := seenCodes[code]; exists {
			rowIssues = append(rowIssues, fmt.Sprintf("站址编码与第%d行重复", firstRow))
		} else {
			seenCodes[code] = rowNumber
		}
		if len(rowIssues) > 0 {
			issues = append(issues, siteImportIssue{Row: rowNumber, Message: strings.Join(rowIssues, "；")})
			continue
		}

		site := model.Site{
			SiteName:         name,
			SiteSerialNumber: code,
			CorpId:           corpMap[strings.TrimSpace(siteImportCell(row, 2))],
		}
		siteDetails := map[string]any{
			"addr":      strings.TrimSpace(siteImportCell(row, 3)),
			"longitude": strings.TrimSpace(siteImportCell(row, 4)),
			"latitude":  strings.TrimSpace(siteImportCell(row, 5)),
		}
		site.SiteDetails, _ = json.Marshal(siteDetails)
		sites = append(sites, site)
	}
	if len(sites) == 0 && len(issues) == 0 {
		issues = append(issues, siteImportIssue{Row: 2, Message: "Excel中没有可导入的数据"})
	}
	return sites, issues
}

func siteImportExistingNameIssues(sites []model.Site, rows [][]string, nameExists func(string) bool) []siteImportIssue {
	rowNumbers := make(map[string]int, len(sites))
	for index, row := range rows[1:] {
		name := strings.TrimSpace(siteImportCell(row, 0))
		if name != "" {
			if _, exists := rowNumbers[name]; !exists {
				rowNumbers[name] = index + 2
			}
		}
	}

	issues := make([]siteImportIssue, 0)
	for _, site := range sites {
		if nameExists(site.SiteName) {
			issues = append(issues, siteImportIssue{
				Row:     rowNumbers[site.SiteName],
				Message: "站点名称已存在，禁止导入",
			})
		}
	}
	return issues
}

func siteImportExistingCodeIssues(sites []model.Site, rows [][]string, codeExists func(string) bool) []siteImportIssue {
	rowNumbers := make(map[string]int, len(sites))
	for index, row := range rows[1:] {
		code := strings.TrimSpace(siteImportCell(row, 1))
		if code != "" {
			if _, exists := rowNumbers[code]; !exists {
				rowNumbers[code] = index + 2
			}
		}
	}

	issues := make([]siteImportIssue, 0)
	for _, site := range sites {
		if codeExists(site.SiteSerialNumber) {
			issues = append(issues, siteImportIssue{
				Row:     rowNumbers[site.SiteSerialNumber],
				Message: "站址编码已存在，禁止导入",
			})
		}
	}
	return issues
}

func mergeSiteImportIssues(issues []siteImportIssue) []siteImportIssue {
	merged := make([]siteImportIssue, 0, len(issues))
	rowIndexes := make(map[int]int)
	for _, issue := range issues {
		if index, exists := rowIndexes[issue.Row]; exists {
			merged[index].Message += "；" + issue.Message
			continue
		}
		rowIndexes[issue.Row] = len(merged)
		merged = append(merged, issue)
	}
	return merged
}

func siteImportCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func siteImportRowEmpty(row []string) bool {
	for index := range model.SiteExportHeader {
		if strings.TrimSpace(siteImportCell(row, index)) != "" {
			return false
		}
	}
	return true
}

func writeSiteImportIssues(f *excelize.File, sheet string, issues []siteImportIssue) error {
	errorColumn := len(model.SiteExportHeader) + 1
	headerCell, _ := excelize.CoordinatesToCellName(errorColumn, 1)
	if err := f.SetCellValue(sheet, headerCell, "校验错误信息"); err != nil {
		return err
	}
	previousHeaderCell, _ := excelize.CoordinatesToCellName(errorColumn-1, 1)
	if styleID, err := f.GetCellStyle(sheet, previousHeaderCell); err == nil && styleID > 0 {
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
	columnName, _ := excelize.ColumnNumberToName(errorColumn)
	return f.SetColWidth(sheet, columnName, columnName, 48)
}

func siteImportErrorFilename(originalFilename string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(originalFilename)), filepath.Ext(originalFilename))
	if base == "" || base == "." {
		base = "站点信息导入"
	}
	return base + "_校验失败.xlsx"
}
