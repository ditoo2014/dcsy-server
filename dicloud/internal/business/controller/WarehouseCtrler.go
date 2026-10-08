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
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type IWarehouseCtrler interface {
	AddWarehouse(ctx *gin.Context)
	UpdateWarehouse(ctx *gin.Context)
	DeleteWarehouse(ctx *gin.Context)
	FindWarehouse(ctx *gin.Context)
	ExportExcel(ctx *gin.Context)
	ImportExcel(ctx *gin.Context)
	ImportTemplate(ctx *gin.Context)
}

type WarehouseCtrler struct {
}

// 添加仓库信息
func (whc WarehouseCtrler) AddWarehouse(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 绑定信息 */

	var rWarehouse model.Warehouse
	ctx.Bind(&rWarehouse)
	rWarehouse.Name = strings.TrimSpace(rWarehouse.Name)
	rWarehouse.WarehouseCode = strings.TrimSpace(rWarehouse.WarehouseCode)

	/* 校验数据 */
	if len(rWarehouse.Name) == 0 {
		log.Println("[Warehouse] controller create house fail")
		response.Fail(ctx, "名称为空", nil)
		return
	}
	if len(rWarehouse.WarehouseCode) == 0 {
		log.Println("[Warehouse] controller create house fail")
		response.Fail(ctx, "库房编码为空", nil)
		return
	}

	warehouseDao := dao.NewWarehouseDao(ctx)
	_, res := warehouseDao.ExistName(rWarehouse)
	if res {
		response.Fail(ctx, "已存在同名库房", nil)
		return
	}
	_, res = warehouseDao.ExistWarehouseCode(rWarehouse)
	if res {
		response.Fail(ctx, "已存在同编码库房", nil)
		return
	}

	/* 填充信息 */
	rWarehouse.CreateUserId = int(userInfo.ID)
	if rWarehouse.CorpId == 0 {
		// 设置默认部门
		var corpIds []int
		for _, v := range userInfo.CorpIds {
			corpIds = append(corpIds, int(v))
		}
		if len(corpIds) > 0 {
			rWarehouse.CorpId = int(corpIds[0])
		}
	}
	if len(rWarehouse.Address) == 0 {
		rWarehouse.Address = "-"
	}
	if len(rWarehouse.ManagerName) == 0 {
		rWarehouse.ManagerName = "-"
	}
	if len(rWarehouse.ManagerPhone) == 0 {
		rWarehouse.ManagerPhone = "-"
	}
	if len(rWarehouse.WarehouseCode) == 0 {
		rWarehouse.WarehouseCode = "-"
	}
	if rWarehouse.WarehouseDetails == nil {
		rWarehouse.WarehouseDetails = []byte(`{}`)
	}

	// 创建
	wh, res := warehouseDao.Create(rWarehouse)
	if !res {
		response.Fail(ctx, "创建失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": wh})
}

// 修改仓库信息
func (whc WarehouseCtrler) UpdateWarehouse(ctx *gin.Context) {

	/* 绑定信息 */
	var rWarehouse model.Warehouse
	ctx.Bind(&rWarehouse)

	if rWarehouse.ID == 0 {
		log.Println("[Warehouse] controller update house fail ID is null")
		response.Fail(ctx, "更新失败", nil)
		return
	}
	rWarehouse.Name = strings.TrimSpace(rWarehouse.Name)
	rWarehouse.WarehouseCode = strings.TrimSpace(rWarehouse.WarehouseCode)
	if rWarehouse.Name == "" {
		response.Fail(ctx, "名称为空", nil)
		return
	}

	// 更新
	warehouseDao := dao.NewWarehouseDao(ctx)
	if _, nameExists := warehouseDao.ExistName(rWarehouse); nameExists {
		response.Fail(ctx, "已存在同名库房", nil)
		return
	}
	if rWarehouse.WarehouseCode != "" {
		existing, codeExists := warehouseDao.ExistWarehouseCode(rWarehouse)
		if codeExists && existing.ID != rWarehouse.ID {
			response.Fail(ctx, "已存在同编码库房", nil)
			return
		}
	}
	wh, res := warehouseDao.Update(rWarehouse)
	if !res {
		response.Fail(ctx, "更新失败", nil)
		return
	}

	response.Success(ctx, "success", gin.H{"data": wh})
}

// 删除仓库信息
func (whc WarehouseCtrler) DeleteWarehouse(ctx *gin.Context) {

	/* 绑定信息 */
	var rWarehouse model.Warehouse
	ctx.Bind(&rWarehouse)

	if rWarehouse.ID == 0 {
		log.Println("[Warehouse] controller delete house fail ID is null")
		response.Fail(ctx, "删除失败", nil)
		return
	}

	// 删除
	warehouseDao := dao.NewWarehouseDao(ctx)
	res := warehouseDao.Delete(rWarehouse)
	if !res {
		response.Fail(ctx, "创建失败", nil)
		return
	}

	response.Success(ctx, "success", nil)
}

// 查询仓库信息列表
func (whc WarehouseCtrler) FindWarehouse(ctx *gin.Context) {

	var (
		pageStr     string
		pageSizeStr string
	)

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 绑定信息 */
	var rWarehouse model.Warehouse
	ctx.Bind(&rWarehouse)

	// 页码, 页大小
	pageStr = ctx.Query("pageNum")
	pageSizeStr = ctx.Query("pageSize")
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	offset, limit := normalutil.GetPageOffsetLimit(page, pageSize)

	/* 鉴权查询 */
	warehouseDao := dao.NewWarehouseDao(ctx)
	if userInfo.Isadmin == 1 {

		if rWarehouse.CorpId > 0 {
			var corpIds []int
			// 查询用户部门
			// 单独部门操作
			corpIds = append(corpIds, int(rWarehouse.CorpId))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rWarehouse.CorpId = 0
			// 查询
			whs, total, res := warehouseDao.ListByCorp(rWarehouse, offset, limit, corpIds)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": whs, "total": total})
			return
		} else {

			whs, total, res := warehouseDao.List(rWarehouse, offset, limit)
			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": whs, "total": total})
			return
		}

	} else {

		/* 用户绑定部门 - 部门绑定设备 - 查询 */
		// binded corp

		// 查询用户部门
		var corpIds []int
		for _, v := range userInfo.CorpIds {
			corpIds = append(corpIds, int(v))
		}
		// 子部门查询
		corpIds = service.FindAllSonCorp(ctx, corpIds)

		// 鉴权部门
		if rWarehouse.CorpId > 0 && len(corpIds) > 0 {
			idExist := -1
			for _, ID := range corpIds {
				if ID == rWarehouse.CorpId {
					idExist = 1
					break
				}
			}
			if idExist == -1 {
				log.Println("[BUSINESS] not exist corp: ", rWarehouse.CorpId)
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}

			// 重新筛选
			corpIds = []int{rWarehouse.CorpId}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)
		}

		if len(corpIds) > 0 {

			// 根据部门 - 鉴权查寻
			whs, total, res := warehouseDao.ListByCorp(rWarehouse, offset, limit, corpIds)

			if !res {
				response.Success(ctx, "success", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": whs, "total": total})
			return
		}

		response.Success(ctx, "success", gin.H{"total": 0})
	}

}

// 导出库房
func (r WarehouseCtrler) ExportExcel(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 绑定信息 */
	var rWarehouse model.Warehouse
	ctx.Bind(&rWarehouse)

	/* 鉴权查询 */
	warehouseDao := dao.NewWarehouseDao(ctx)
	whs := []model.Warehouse{}
	if userInfo.Isadmin == 1 {

		if rWarehouse.CorpId > 0 {
			var corpIds []int
			// 查询用户部门
			// 单独部门操作
			corpIds = append(corpIds, int(rWarehouse.CorpId))

			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)

			rWarehouse.CorpId = 0
			// 查询
			whs, _, _ = warehouseDao.ListByCorp(rWarehouse, 0, 9999, corpIds)
		} else {

			whs, _, _ = warehouseDao.List(rWarehouse, 0, 9999)
		}
	} else {

		/* 用户绑定部门 - 部门绑定设备 - 查询 */

		// 查询用户部门
		var corpIds []int
		for _, v := range userInfo.CorpIds {
			corpIds = append(corpIds, int(v))
		}
		// 子部门查询
		corpIds = service.FindAllSonCorp(ctx, corpIds)

		// 鉴权部门
		if rWarehouse.CorpId > 0 && len(corpIds) > 0 {
			idExist := -1
			for _, ID := range corpIds {
				if ID == rWarehouse.CorpId {
					idExist = 1
					break
				}
			}
			if idExist == -1 {
				log.Println("[BUSINESS] not exist corp: ", rWarehouse.CorpId)
				response.Fail(ctx, "非法部门, 未授权", nil)
				return
			}

			// 重新筛选
			corpIds = []int{rWarehouse.CorpId}
			// 子部门查询
			corpIds = service.FindAllSonCorp(ctx, corpIds)
		}

		if len(corpIds) > 0 {

			// 根据部门 - 鉴权查寻
			whs, _, _ = warehouseDao.ListByCorp(rWarehouse, 0, 9999, corpIds)

		}

	}

	/* 导出 */
	if len(whs) > 0 {

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

		for i, v := range model.WarehouseExportHeader {
			//rune('A'+i)：根据当前索引 i 计算出 Excel 表中的列名。rune('A') 是字符 'A' 的 Unicode 码点值，加上索引 i 后将其转换回字符，从而得到列名。例如，当 i=0 时，结果是 'A'；当 i=1 时，结果是 'B'，依此类推。
			cell := fmt.Sprintf("%s1", string(rune('A'+i)))
			f.SetCellValue("Sheet1", cell, v)
		}

		for idx, v := range whs {
			f.SetCellValue("Sheet1", fmt.Sprintf("A%d", idx+2), v.Name)
			f.SetCellValue("Sheet1", fmt.Sprintf("B%d", idx+2), v.WarehouseCode)
			f.SetCellValue("Sheet1", fmt.Sprintf("C%d", idx+2), v.Address)
			f.SetCellValue("Sheet1", fmt.Sprintf("D%d", idx+2), v.ManagerName)
			f.SetCellValue("Sheet1", fmt.Sprintf("E%d", idx+2), v.ManagerPhone)

			// 部门
			if len(corpMap[v.CorpId]) > 0 {
				f.SetCellValue("Sheet1", fmt.Sprintf("F%d", idx+2), corpMap[v.CorpId])
			} else {
				f.SetCellValue("Sheet1", fmt.Sprintf("F%d", idx+2), v.CorpId)
			}

			var mapTem = make(map[string]any)
			bytesData, _ := v.WarehouseDetails.MarshalJSON()
			json.Unmarshal(bytesData, &mapTem)
			f.SetCellValue("Sheet1", fmt.Sprintf("G%d", idx+2), mapTem["longitude"])
			f.SetCellValue("Sheet1", fmt.Sprintf("H%d", idx+2), mapTem["latitude"])
		}

		f.SetActiveSheet(index)

		writeWarehouseWorkbook(ctx, "库房信息.xlsx", f, http.StatusOK, "导出成功")
		return
	}

	response.Fail(ctx, "导出失败", nil)
}

func writeWarehouseWorkbook(ctx *gin.Context, fileName string, f *excelize.File, responseCode int, responseMessage string) {
	ctx.Set(response.ContextResponseHTTPStatus, http.StatusOK)
	ctx.Set(response.ContextResponseCode, responseCode)
	ctx.Set(response.ContextResponseMsg, responseMessage)
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="warehouse.xlsx"; filename*=UTF-8''%s`, url.PathEscape(fileName)))
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition, X-Import-Validation-Failed, X-Import-Error-Count")
	ctx.Status(http.StatusOK)
	if err := f.Write(ctx.Writer); err != nil {
		log.Println("[BUSINESS] write warehouse workbook fail", err)
	}
}

// 导入Excel数据
func (ctrler WarehouseCtrler) ImportExcel(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	// 获取上传文件
	file, fileHeader, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Println("[BUSINESS-Warehouse] upload file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer file.Close()

	// 临时文件处理
	filePath := fmt.Sprintf("equipment_import_%d.xlsx", time.Now().Unix())
	// 创建临时文件
	tempFile, err := os.Create(filePath)
	if err != nil {
		log.Println("[BUSINESS-Warehouse] create temp file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer os.Remove(filePath)
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Println("[BUSINESS-Warehouse] copy to temp file fail", err)
		response.Fail(ctx, "文件复制出错", nil)
		return
	}

	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		log.Println("[BUSINESS-Warehouse] excelize open file fail", err)
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

	whDao := dao.NewWarehouseDao(ctx)
	parsedWarehouses, issues := validateWarehouseImportRows(rows, corpMap)
	issues = append(issues, warehouseImportExistingNameIssues(parsedWarehouses, rows, func(name string) bool {
		_, exists := whDao.ExistName(model.Warehouse{Name: name})
		return exists
	})...)
	issues = append(issues, warehouseImportExistingCodeIssues(parsedWarehouses, rows, func(code string) bool {
		_, exists := whDao.ExistWarehouseCode(model.Warehouse{WarehouseCode: code})
		return exists
	})...)
	issues = mergeWarehouseImportIssues(issues)
	if len(issues) > 0 {
		if err := writeWarehouseImportIssues(f, sheets[0], issues); err != nil {
			log.Println("[BUSINESS] create warehouse import validation file fail", err)
			response.Fail(ctx, "生成校验失败文件出错", nil)
			return
		}
		ctx.Header("X-Import-Validation-Failed", "true")
		ctx.Header("X-Import-Error-Count", strconv.Itoa(len(issues)))
		writeWarehouseWorkbook(ctx, warehouseImportErrorFilename(fileHeader.Filename), f, 400, "导入校验失败")
		return
	}

	whs := make([]model.Warehouse, 0, len(parsedWarehouses))
	for _, wh := range parsedWarehouses {
		baseWarehouseDataFill(&wh)
		wh.CreateUserId = int(userInfo.ID)
		whs = append(whs, wh)
	}

	// 保存
	if len(whs) > 0 {
		dataList, res := whDao.ImportWarehouses(whs, nil)
		if !res {
			response.Fail(ctx, "导入失败", nil)
			return
		}
		response.Success(ctx, "导入成功",
			gin.H{"data": dataList, "successCount": len(whs),
				"dumplicateCount": 0, "failCount": 0})
		return
	}

	response.Fail(ctx, "导入失败", nil)
}

type warehouseImportIssue struct {
	Row     int    `json:"row"`
	Message string `json:"message"`
}

func validateWarehouseImportRows(rows [][]string, corpMap map[string]int) ([]model.Warehouse, []warehouseImportIssue) {
	if len(rows) == 0 {
		return nil, []warehouseImportIssue{{Row: 1, Message: "Excel不能为空"}}
	}
	for index, title := range model.WarehouseExportHeader {
		if strings.TrimSpace(warehouseImportCell(rows[0], index)) != title {
			return nil, []warehouseImportIssue{{Row: 1, Message: "表头必须为：" + strings.Join(model.WarehouseExportHeader, "、")}}
		}
	}

	warehouses := make([]model.Warehouse, 0, len(rows)-1)
	issues := make([]warehouseImportIssue, 0)
	seenNames := make(map[string]int)
	seenCodes := make(map[string]int)
	for index, row := range rows[1:] {
		rowNumber := index + 2
		if warehouseImportRowEmpty(row) {
			continue
		}
		name := strings.TrimSpace(warehouseImportCell(row, 0))
		code := strings.TrimSpace(warehouseImportCell(row, 1))
		rowIssues := make([]string, 0, 3)
		if name == "" {
			rowIssues = append(rowIssues, "库房名称不能为空")
		} else if firstRow, exists := seenNames[name]; exists {
			rowIssues = append(rowIssues, fmt.Sprintf("库房名称与第%d行重复", firstRow))
		} else {
			seenNames[name] = rowNumber
		}
		if code == "" {
			rowIssues = append(rowIssues, "库房编码不能为空")
		} else if firstRow, exists := seenCodes[code]; exists {
			rowIssues = append(rowIssues, fmt.Sprintf("库房编码与第%d行重复", firstRow))
		} else {
			seenCodes[code] = rowNumber
		}
		if len(rowIssues) > 0 {
			issues = append(issues, warehouseImportIssue{Row: rowNumber, Message: strings.Join(rowIssues, "；")})
			continue
		}

		warehouse := model.Warehouse{
			Name:          name,
			WarehouseCode: code,
			Address:       strings.TrimSpace(warehouseImportCell(row, 2)),
			ManagerName:   strings.TrimSpace(warehouseImportCell(row, 3)),
			ManagerPhone:  strings.TrimSpace(warehouseImportCell(row, 4)),
			CorpId:        corpMap[strings.TrimSpace(warehouseImportCell(row, 5))],
		}
		warehouseDetails := map[string]any{
			"longitude": warehouseImportCell(row, 6),
			"latitude":  warehouseImportCell(row, 7),
		}
		warehouse.WarehouseDetails, _ = json.Marshal(warehouseDetails)
		warehouses = append(warehouses, warehouse)
	}
	if len(warehouses) == 0 && len(issues) == 0 {
		issues = append(issues, warehouseImportIssue{Row: 2, Message: "Excel中没有可导入的数据"})
	}
	return warehouses, issues
}

func warehouseImportExistingNameIssues(warehouses []model.Warehouse, rows [][]string, nameExists func(string) bool) []warehouseImportIssue {
	rowNumbers := make(map[string]int, len(warehouses))
	for index, row := range rows[1:] {
		name := strings.TrimSpace(warehouseImportCell(row, 0))
		if name != "" {
			if _, exists := rowNumbers[name]; !exists {
				rowNumbers[name] = index + 2
			}
		}
	}

	issues := make([]warehouseImportIssue, 0)
	for _, warehouse := range warehouses {
		if nameExists(warehouse.Name) {
			issues = append(issues, warehouseImportIssue{
				Row:     rowNumbers[warehouse.Name],
				Message: "库房名称已存在，禁止导入",
			})
		}
	}
	return issues
}

func warehouseImportExistingCodeIssues(warehouses []model.Warehouse, rows [][]string, codeExists func(string) bool) []warehouseImportIssue {
	rowNumbers := make(map[string]int, len(warehouses))
	for index, row := range rows[1:] {
		code := strings.TrimSpace(warehouseImportCell(row, 1))
		if code != "" {
			if _, exists := rowNumbers[code]; !exists {
				rowNumbers[code] = index + 2
			}
		}
	}

	issues := make([]warehouseImportIssue, 0)
	for _, warehouse := range warehouses {
		if codeExists(warehouse.WarehouseCode) {
			issues = append(issues, warehouseImportIssue{
				Row:     rowNumbers[warehouse.WarehouseCode],
				Message: "库房编码已存在，禁止导入",
			})
		}
	}
	return issues
}

func mergeWarehouseImportIssues(issues []warehouseImportIssue) []warehouseImportIssue {
	merged := make([]warehouseImportIssue, 0, len(issues))
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

func warehouseImportCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return row[index]
}

func warehouseImportRowEmpty(row []string) bool {
	for index := range model.WarehouseExportHeader {
		if strings.TrimSpace(warehouseImportCell(row, index)) != "" {
			return false
		}
	}
	return true
}

func writeWarehouseImportIssues(f *excelize.File, sheet string, issues []warehouseImportIssue) error {
	errorColumn := len(model.WarehouseExportHeader) + 1
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

func warehouseImportErrorFilename(originalFilename string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(originalFilename)), filepath.Ext(originalFilename))
	if base == "" || base == "." {
		base = "库房信息导入"
	}
	return base + "_校验失败.xlsx"
}

// 基础数据填充
func baseWarehouseDataFill(rWarehouse *model.Warehouse) {

	if len(rWarehouse.WarehouseCode) == 0 {
		rWarehouse.WarehouseCode = "-"
	}
	if len(rWarehouse.Address) == 0 {
		rWarehouse.Address = "-"
	}
	if len(rWarehouse.ManagerName) == 0 {
		rWarehouse.ManagerName = "-"
	}
	if len(rWarehouse.ManagerPhone) == 0 {
		rWarehouse.ManagerPhone = "-"
	}
	if rWarehouse.WarehouseDetails == nil {
		rWarehouse.WarehouseDetails = []byte(`{}`)
	}

}
