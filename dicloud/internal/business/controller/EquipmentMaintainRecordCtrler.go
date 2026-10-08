package controller

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	"dicloud/util/gormutil"
	"dicloud/util/response"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type IEquipmentMaintainRecordCtrler interface {
	Create(ctx *gin.Context)
	Find(ctx *gin.Context)
	ExportExcel(ctx *gin.Context)
	ImportExcel(ctx *gin.Context)
}

type EquipmentMaintainRecordCtrler struct{}

// maintainRecordRequest 只允许客户端提交业务筛选条件，不接受权限范围字段。
type maintainRecordRequest struct {
	EquipmentID       int    `json:"equipmentId"`
	EquipmentCode     string `json:"equipmentCode"`
	EquipmentTypeName string `json:"equipmentTypeName"`
	MinimumCount      int    `json:"minimumCount"`
	StartTime         int64  `json:"startTime"`
	EndTime           int64  `json:"endTime"`
}

// maintainUserScope 对管理员和普通用户统一生成服务端数据权限。
func maintainUserScope(ctx *gin.Context) (usermodel.User, []int, bool) {
	value, exists := ctx.Get("userInfo")
	user, ok := value.(usermodel.User)
	if !exists || !ok || user.ID == 0 {
		response.Fail(ctx, "登录信息无效", nil)
		return user, nil, false
	}
	var corpIDs []int
	if user.Isadmin != 1 {
		for _, id := range user.CorpIds {
			// 0 表示设备尚未分配部门，不能把它当作组织根节点展开全部部门。
			if id > 0 {
				corpIDs = append(corpIDs, int(id))
			}
		}
		// 空部门列表不能视为管理员，也不查询全部部门。
		if len(corpIDs) > 0 {
			corpIDs = service.FindAllSonCorp(ctx, corpIDs)
			// 设备列表的权限口径包含尚未分配部门的设备；运维列表、
			// 新增和导入也必须使用同一口径，避免同一用户看到的数据不一致。
			if len(corpIDs) > 0 {
				corpIDs = append(corpIDs, 0)
			}
		}
	}
	return user, corpIDs, true
}

func bindMaintainFilter(ctx *gin.Context, now int64) (dao.MaintainRecordFilter, error) {
	var request maintainRecordRequest
	if err := ctx.ShouldBindJSON(&request); err != nil && !errors.Is(err, io.EOF) {
		return dao.MaintainRecordFilter{}, errors.New("运维查询参数格式错误")
	}
	for key, target := range map[string]*int64{"startTime": &request.StartTime, "endTime": &request.EndTime} {
		if raw, present := ctx.GetQuery(key); present {
			value, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || value < 0 {
				return dao.MaintainRecordFilter{}, errors.New("时间参数必须为非负秒级时间戳")
			}
			*target = value
		}
	}
	if request.EndTime == 0 {
		request.EndTime = now
	}
	if request.StartTime < 0 || request.EndTime < request.StartTime {
		return dao.MaintainRecordFilter{}, errors.New("开始时间不能晚于结束时间")
	}
	if request.EquipmentID < 0 || request.MinimumCount < 0 {
		return dao.MaintainRecordFilter{}, errors.New("设备编号和最少运维次数不能为负数")
	}
	return dao.MaintainRecordFilter{
		EquipmentID: request.EquipmentID, EquipmentCode: request.EquipmentCode,
		EquipmentTypeName: request.EquipmentTypeName, MinimumCount: request.MinimumCount,
		StartTime: request.StartTime, EndTime: request.EndTime,
	}, nil
}

// Create 新增运维记录，检查目标设备属于当前用户有权操作的部门。
func (EquipmentMaintainRecordCtrler) Create(ctx *gin.Context) {
	user, corpIDs, ok := maintainUserScope(ctx)
	if !ok {
		return
	}
	var record model.EquipmentMaintainRecord
	if err := ctx.ShouldBindJSON(&record); err != nil || record.EquipmentId <= 0 {
		response.Fail(ctx, "设备编号或运维记录格式错误", nil)
		return
	}
	query := gormutil.GetGormDB(ctx, "").Model(&model.Equipment{}).Where("id = ?", record.EquipmentId)
	if user.Isadmin != 1 {
		query = query.Where("corp_id IN ?", corpIDs)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil || count == 0 {
		response.Fail(ctx, "设备不存在或无权操作该设备", nil)
		return
	}
	if len(record.MaintainDetails) == 0 {
		record.MaintainDetails = json.RawMessage(`{}`)
	}
	if record.MaintainTitle == "" {
		record.MaintainTitle = "-"
	}
	if record.MaintainContent == "" {
		record.MaintainContent = "-"
	}
	record.ID = 0
	record.MaintainUserId = int(user.ID)
	record.MaintainUserName = user.Username
	record.CreateTime = time.Now().Unix()
	if _, ok := dao.NewEquipmentMaintainRecordDao(ctx).Create(record); !ok {
		response.Fail(ctx, "记录失败", nil)
		return
	}
	response.Success(ctx, "记录成功", nil)
}

// Find 不指定设备时查询所有有权限的设备，数量、分页和导出共用相同条件。
func (EquipmentMaintainRecordCtrler) Find(ctx *gin.Context) {
	user, corpIDs, ok := maintainUserScope(ctx)
	if !ok {
		return
	}
	filter, err := bindMaintainFilter(ctx, time.Now().Unix())
	if err != nil {
		response.Fail(ctx, err.Error(), nil)
		return
	}
	filter.IsAdmin, filter.CorpIDs = user.Isadmin == 1, corpIDs
	page := positiveQueryInt(ctx, "pageNum", 1)
	pageSize := positiveQueryInt(ctx, "pageSize", 20)
	if pageSize > 10000 {
		pageSize = 10000
	}
	rows, total, err := dao.NewEquipmentMaintainRecordDao(ctx).FindScoped(filter, (page-1)*pageSize, pageSize)
	if err != nil {
		log.Printf("[BUSINESS] 查询运维记录失败: %v", err)
		response.Fail(ctx, "查询运维记录失败", nil)
		return
	}
	response.Success(ctx, "查询成功", gin.H{"data": rows, "total": total})
}

// ExportExcel 直接返回工作簿流，不依赖 Linux 临时目录或额外的静态下载服务器。
func (EquipmentMaintainRecordCtrler) ExportExcel(ctx *gin.Context) {
	user, corpIDs, ok := maintainUserScope(ctx)
	if !ok {
		return
	}
	filter, err := bindMaintainFilter(ctx, time.Now().Unix())
	if err != nil {
		response.Fail(ctx, err.Error(), nil)
		return
	}
	filter.IsAdmin, filter.CorpIDs = user.Isadmin == 1, corpIDs
	rows, _, err := dao.NewEquipmentMaintainRecordDao(ctx).FindScoped(filter, 0, 0)
	if err != nil {
		log.Printf("[BUSINESS] 查询运维导出数据失败: %v", err)
		response.Fail(ctx, "查询运维导出数据失败", nil)
		return
	}
	f, err := buildMaintainWorkbook(rows)
	if err != nil {
		log.Printf("[BUSINESS] 生成运维工作簿失败: %v", err)
		response.Fail(ctx, "生成运维工作簿失败", nil)
		return
	}
	defer f.Close()
	writeMaintainWorkbook(ctx, f)
}

func buildMaintainWorkbook(rows []dao.MaintainRecordRow) (*excelize.File, error) {
	f := excelize.NewFile()
	header := make([]any, len(model.EquipmentMaintainRecordHeader))
	for i, name := range model.EquipmentMaintainRecordHeader {
		header[i] = name
	}
	if err := f.SetSheetRow("Sheet1", "A1", &header); err != nil {
		f.Close()
		return nil, err
	}
	for index, row := range rows {
		details := decodeMaintainDetails(row.MaintainDetails)
		content := strings.TrimSpace(row.MaintainTitle)
		if content == "" || content == "-" {
			content = row.MaintainContent
		}
		operationTime := formatMaintainOperationTime(row.CreateTime)
		if operationTime == "" {
			if timestamp, ok := maintainDetailsTime(details); ok {
				operationTime = formatMaintainOperationTime(timestamp)
			}
		}
		imageURL := legalMaintainImageReference(details["imageUrl"])
		// 与页面的电池 ID 一致，旧设备无编码时使用名称，不导出不可识别的占位符。
		equipmentCode := maintainEquipmentIdentifier(row.EquipmentCode, row.EquipmentName)
		values := []any{equipmentCode, row.SiteSerialNumber, content, imageURL, operationTime, details["bs"]}
		if err := f.SetSheetRow("Sheet1", fmt.Sprintf("A%d", index+2), &values); err != nil {
			f.Close()
			return nil, err
		}
		if imageURL != "" {
			// 兼容旧绝对链接，但只读取配置上传目录内同名文件，绝不请求远程地址。
			if imageData, extension, err := readMaintainImage(imageURL); err == nil {
				if err := f.AddPictureFromBytes("Sheet1", fmt.Sprintf("D%d", index+2), &excelize.Picture{
					Extension: extension, File: imageData, Format: &excelize.GraphicOptions{AutoFit: true},
				}); err != nil {
					log.Printf("[BUSINESS] 运维导出图片不可用，记录 %d: %v", row.ID, err)
				}
			}
		}
	}
	return f, nil
}

func writeMaintainWorkbook(ctx *gin.Context, f *excelize.File) {
	buffer, err := f.WriteToBuffer()
	if err != nil {
		response.Fail(ctx, "生成运维工作簿失败", nil)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename=maintenance.xlsx; filename*=UTF-8''"+url.PathEscape("运维信息.xlsx"))
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", buffer.Bytes())
}

// ImportExcel 直接读取上传内容；仅导入当前用户有权限的设备，并安全处理短行。
func (EquipmentMaintainRecordCtrler) ImportExcel(ctx *gin.Context) {
	user, corpIDs, ok := maintainUserScope(ctx)
	if !ok {
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 20<<20)
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		response.Fail(ctx, "上传文件出错或文件超过 20MB", nil)
		return
	}
	defer file.Close()
	if ctx.Request.MultipartForm != nil {
		defer ctx.Request.MultipartForm.RemoveAll()
	}
	f, err := excelize.OpenReader(file)
	if err != nil {
		response.Fail(ctx, "无法打开 Excel 文件", nil)
		return
	}
	defer f.Close()
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		response.Fail(ctx, "工作簿没有工作表", nil)
		return
	}
	rows, err := readMaintainImportRows(f, sheets[0])
	if err != nil {
		response.Fail(ctx, "读取表格失败: "+err.Error(), nil)
		return
	}
	if err := validateMaintainImportHeader(rows); err != nil {
		response.Fail(ctx, err.Error(), nil)
		return
	}
	if err := validateMaintainEmbeddedPictures(f, sheets[0], rows); err != nil {
		// excelize 可以读取嵌入图片，但当前记录模型只保存图片引用；拒绝无引用图片，避免静默丢失。
		response.Fail(ctx, err.Error(), nil)
		return
	}
	query := gormutil.GetGormDB(ctx, "").Model(&model.Equipment{}).Select("id, equipment_code, equipment_name")
	if user.Isadmin != 1 {
		query = query.Where("corp_id IN ?", corpIDs)
	}
	var equipments []model.Equipment
	if err := query.Find(&equipments).Error; err != nil {
		response.Fail(ctx, "查询有权限的设备失败", nil)
		return
	}
	allowed := make(map[string]int, len(equipments))
	for _, equipment := range equipments {
		identifier := maintainEquipmentIdentifier(equipment.EquipmentCode, equipment.EquipmentName)
		if identifier == "" {
			continue
		}
		// 旧数据可能有重名；存在歧义时拒绝该标识，不能悄悄绑定到最后一台设备。
		if _, exists := allowed[identifier]; exists {
			allowed[identifier] = 0
		} else {
			allowed[identifier] = int(equipment.ID)
		}
	}
	records, duplicateCount, failCount := parseMaintainImportRowsDetailed(rows, allowed, user, time.Now().Unix())
	recordDAO := dao.NewEquipmentMaintainRecordDao(ctx)
	// 对历史库仅使用设备、运维时间、故障描述三项完全相同的可靠键去重；任何查询错误都中止导入。
	filteredRecords := make([]model.EquipmentMaintainRecord, 0, len(records))
	for _, record := range records {
		existing, queryErr := recordDAO.CountExactRecord(record)
		if queryErr != nil {
			response.Fail(ctx, "检查历史重复运维记录失败", nil)
			return
		}
		if existing > 0 {
			duplicateCount++
			continue
		}
		filteredRecords = append(filteredRecords, record)
	}
	records = filteredRecords
	if len(records) == 0 {
		// 文件内容全部是已存在的重复记录时属于幂等导入，不应让前端把“未新增”误报成系统失败。
		// 失败行仍通过 failCount 返回，用户可以据此修正表格后重试。
		if duplicateCount > 0 {
			response.Success(ctx, "导入完成，没有新增记录", gin.H{
				"data": []model.EquipmentMaintainRecord{}, "successCount": 0,
				"duplicateCount": duplicateCount, "dumplicateCount": duplicateCount,
				"failCount": failCount,
			})
			return
		}
		response.Fail(ctx, "没有可导入的运维记录，请检查设备权限和表格内容", gin.H{"duplicateCount": duplicateCount, "dumplicateCount": duplicateCount, "failCount": failCount})
		return
	}
	saved, ok := recordDAO.ImportEquipmentMaintainRecords(records)
	if !ok {
		response.Fail(ctx, "导入失败", nil)
		return
	}
	response.Success(ctx, "导入成功", gin.H{"data": saved, "successCount": len(records), "duplicateCount": duplicateCount, "dumplicateCount": duplicateCount, "failCount": failCount})
}

// parseMaintainImportRows 保留旧测试和内部调用的二元返回形式；新的导入接口使用详细计数版本。
func parseMaintainImportRows(rows [][]string, allowed map[string]int, user usermodel.User, now int64) ([]model.EquipmentMaintainRecord, int) {
	records, _, failCount := parseMaintainImportRowsDetailed(rows, allowed, user, now)
	return records, failCount
}

// parseMaintainImportRowsDetailed 将运维时间作为业务记录时间，文件内同键记录只保留一次。
func parseMaintainImportRowsDetailed(rows [][]string, allowed map[string]int, user usermodel.User, now int64) ([]model.EquipmentMaintainRecord, int, int) {
	records := make([]model.EquipmentMaintainRecord, 0)
	seen := make(map[string]struct{})
	duplicateCount := 0
	failCount := 0
	for index, row := range rows {
		if index == 0 {
			continue
		}
		if isEmptyMaintainImportRow(row) {
			continue
		}
		row, ok := normalizeMaintainDataRow(row)
		if !ok {
			failCount++
			continue
		}
		equipmentCode := strings.TrimSpace(row[0])
		content := strings.TrimSpace(row[2])
		imageURL := legalMaintainImageReference(row[3])
		operationTime, err := parseMaintainOperationTime(row[4])
		if equipmentCode == "" || content == "" || err != nil || allowed[equipmentCode] <= 0 || (strings.TrimSpace(row[3]) != "" && imageURL == "") {
			failCount++
			continue
		}
		key := fmt.Sprintf("%d\x00%d\x00%s", allowed[equipmentCode], operationTime, content)
		if _, exists := seen[key]; exists {
			duplicateCount++
			continue
		}
		seen[key] = struct{}{}
		detailsMap := map[string]string{"create_time": formatMaintainOperationTime(operationTime)}
		if marker := strings.TrimSpace(row[5]); marker != "" {
			detailsMap["bs"] = marker
		}
		if imageURL != "" {
			detailsMap["imageUrl"] = imageURL
		}
		details, _ := json.Marshal(detailsMap)
		records = append(records, model.EquipmentMaintainRecord{
			EquipmentId: allowed[equipmentCode], MaintainContent: content, MaintainTitle: content,
			MaintainDetails: details, CreateTime: operationTime, MaintainUserId: int(user.ID), MaintainUserName: user.Username,
		})
	}
	return records, duplicateCount, failCount
}
