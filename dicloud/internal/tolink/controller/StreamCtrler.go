package controller

import (
	"dicloud/internal/tolink/dao"
	"dicloud/internal/tolink/model"
	usermodel "dicloud/models/model"
	"dicloud/util/response"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	// github.com/xuri/excelize/v2 问题: 2.9.1版本出现math.MaxUint32 overflows int 异常;
	// 获取: go get -u github.com/xuri/excelize/v2@master 新包解决
	"github.com/xuri/excelize/v2"
)

type IStreamCtrler interface {
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	FindPage(ctx *gin.Context)
	Label(ctx *gin.Context)
	GetLabel(ctx *gin.Context)
	ImportExcel(ctx *gin.Context)
	ExportExcel(ctx *gin.Context)
	BatchImportTemplate(ctx *gin.Context)
	BatchImport(ctx *gin.Context)
	// FindOne(ctx *gin.Context)
}

type StreamCtrler struct {
}

// ExportExcel 到处excel表格
func (s StreamCtrler) ExportExcel(ctx *gin.Context) {

	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)

	streamDao := dao.NewStreamDAO(ctx)

	outputData, err := streamDao.GetLabel(streamInfo)
	if err != nil {
		response.Fail(ctx, "获取数据失败", nil)
		return
	}

	//创建一个新的excel文件
	f := excelize.NewFile()

	//表头
	headers := []string{"标签名", "网关ID", "关联网关", "关联数据", "处理公式", "逆处理公式"}
	for i, header := range headers { //i 是 headers 切片的索引（从 0 开始），header 是当前遍历到的表头名称。
		cell := fmt.Sprintf("%s1", string(rune('A'+i))) //rune('A'+i)：根据当前索引 i 计算出 Excel 表中的列名。rune('A') 是字符 'A' 的 Unicode 码点值，加上索引 i 后将其转换回字符，从而得到列名。例如，当 i=0 时，结果是 'A'；当 i=1 时，结果是 'B'，依此类推。
		//fmt.Sprintf("%s1", string(rune('A'+i)))：将列名拼接上行号 1，得到单元格的位置。例如，当 i=0 时，结果是 "A1"；当 i=1 时，结果是 "B1"。
		f.SetCellValue("Sheet1", cell, header) //f.SetCellValue("Sheet1", cell, header)：在 Sheet1 表的 cell 位置设置单元格的值为 header。例如，如果 header 是 "ID" 且 cell 是 "A1"，则在 A1 单元格中写入 "ID"。
	}

	for i, s := range outputData {
		f.SetCellValue("Sheet1", fmt.Sprintf("A%d", i+2), s.Name)
		f.SetCellValue("Sheet1", fmt.Sprintf("B%d", i+2), s.GatewayId)
		f.SetCellValue("Sheet1", fmt.Sprintf("C%d", i+2), s.GatewayName)
		f.SetCellValue("Sheet1", fmt.Sprintf("D%d", i+2), s.DataName)
		f.SetCellValue("Sheet1", fmt.Sprintf("E%d", i+2), s.Formula)
		f.SetCellValue("Sheet1", fmt.Sprintf("F%d", i+2), s.Unformula)
	}

	//设置保存文件的路径
	filePath := fmt.Sprintf("数据池：%s-%s.xlsx", streamInfo.Name, time.Now().Format("2006-01-02-15-04-05"))

	// 保存 Excel 文件，确保路径使用 .xlsx 扩展名
	if err := f.SaveAs(filePath); err != nil {
		fmt.Println("err:", err)
		response.Fail(ctx, "导出失败", gin.H{"error": err})
		return
	}

	// 设置响应头
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", url.QueryEscape(filePath), url.QueryEscape(filePath)))
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.File(filePath)
	defer os.Remove(filePath)

}

// ImportExcel 导入excel表
func (s StreamCtrler) ImportExcel(ctx *gin.Context) {

	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)
	fmt.Println("streamInfo:", streamInfo)

	//获取上传文件
	file, err := ctx.FormFile("file")
	if err != nil {
		response.Fail(ctx, "获取上传文件失败", gin.H{"error": err})
		return
	}

	// 保存上传的文件到临时文件
	tempFile, err := os.CreateTemp("", "upload-*.xlsx")
	if err != nil {
		response.Fail(ctx, "保存临时文件失败", gin.H{"error": err})
		return
	}
	// 确保函数返回时删除临时文件
	defer os.Remove(tempFile.Name())

	defer tempFile.Close()

	if err := ctx.SaveUploadedFile(file, tempFile.Name()); err != nil {
		response.Fail(ctx, "获取上传文件失败", gin.H{"error": err})
		return
	}

	//打开excel表，并处理excel表文件
	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		response.Fail(ctx, "打开excel表失败", gin.H{"error": err})
		return
	}
	defer f.Close()

	// 获取第一个工作表的所有行
	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		response.Fail(ctx, "获取表行失败", gin.H{"error": err})
		return
	}

	var streamLabelUnitList []model.StreamLabelUnit

	for _, col := range rows[1:] {
		fmt.Println("col:", col)
		streamLabelUnit := model.StreamLabelUnit{
			StreamId:    streamInfo.ID,
			LabelId:     strings.ReplaceAll(uuid.New().String(), "-", ""),
			Name:        col[0],
			GatewayName: col[2],
			GatewayId:   col[1],
			DataName:    col[3],
			DataId:      time.Now().String(), //此处需要修改
			Formula:     col[4],
			Unformula:   col[5],
		}
		streamLabelUnitList = append(streamLabelUnitList, streamLabelUnit)
	}

	streamDao := dao.NewStreamDAO(ctx)

	err = streamDao.CreateStreamLabelUnit(streamLabelUnitList)
	if err != nil {
		response.Fail(ctx, "导入失败", gin.H{"error": err})
		return
	}

	response.Success(ctx, "导入成功", nil)
}

// GetLabel 获取标签
func (s StreamCtrler) GetLabel(ctx *gin.Context) {

	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)
	fmt.Println("streamInfo:", streamInfo)
	function := ctx.Query("Function")
	streamDao := dao.NewStreamDAO(ctx)

	switch function {
	case "intact":
		list, err := streamDao.GetLabel(streamInfo)
		if err != nil {
			response.Fail(ctx, "查询失败", gin.H{"error": err})
			return
		}
		response.Success(ctx, "查询成功", gin.H{"list": list})
	case "list":
		list, err := streamDao.GetLabel(streamInfo)
		if err != nil {
			response.Fail(ctx, "查询失败", gin.H{"error": err})
			return
		}
		response.Success(ctx, "查询成功", gin.H{"list": list})
	default:
		response.Fail(ctx, "功能指令错误", nil)
		return
	}

}

// Label 添加标签
func (s StreamCtrler) Label(ctx *gin.Context) {

	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)
	var streamLabelUnit []model.StreamLabelUnit
	if err := ctx.Bind(&streamLabelUnit); err != nil {
		response.Fail(ctx, "参数错误", gin.H{"error": err})
		return
	}
	/* 获取操作信息 */

	function := ctx.Query("Function")

	streamDao := dao.NewStreamDAO(ctx)

	switch function {
	case "create":

		for v := range streamLabelUnit {
			if len(streamLabelUnit[v].GatewayId) == 0 {
				response.Fail(ctx, "网关ID为空", nil)
				return
			}
			if len(streamLabelUnit[v].DataId) == 0 {
				response.Fail(ctx, "数据点ID为空", nil)
				return
			}
			if len(streamLabelUnit[v].Name) == 0 {
				response.Fail(ctx, "标签名称为空", nil)
				return
			}
			if len(streamLabelUnit[v].GatewayName) == 0 {
				response.Fail(ctx, "网关名称为空", nil)
				return
			}
			if len(streamLabelUnit[v].DataName) == 0 {
				response.Fail(ctx, "数据名称为空", nil)
				return
			}
			if len(streamLabelUnit[v].Formula) == 0 {
				response.Fail(ctx, "标签处理公式为空", nil)
				return
			}
			if len(streamLabelUnit[v].Unformula) == 0 {
				response.Fail(ctx, "标签逆处理公式为空", nil)
				return
			}
			streamLabelUnit[v].LabelId = strings.ReplaceAll(uuid.New().String(), "-", "")
			streamLabelUnit[v].StreamId = streamInfo.ID
		}
		//fmt.Println("data:", streamLabelUnit)

		//删除以前的数据
		err := streamDao.DeleteStreamLabelUnitByStreamID(streamInfo)
		if err != nil {
			response.Fail(ctx, "处理失败", gin.H{"error": err})
			return
		}
		//存数据库
		err = streamDao.CreateStreamLabelUnit(streamLabelUnit)
		if err != nil {
			response.Fail(ctx, "创建标签失败", gin.H{"error": err})
			return
		}
		response.Success(ctx, "创建成功", nil)
	case "update":
		for v := range streamLabelUnit {
			if len(streamLabelUnit[v].GatewayId) == 0 {
				response.Fail(ctx, "网关ID为空", nil)
				return
			}
			if len(streamLabelUnit[v].DataId) == 0 {
				response.Fail(ctx, "数据点ID为空", nil)
				return
			}
			if len(streamLabelUnit[v].Name) == 0 {
				response.Fail(ctx, "标签名称为空", nil)
				return
			}
			if len(streamLabelUnit[v].GatewayName) == 0 {
				response.Fail(ctx, "网关名称为空", nil)
				return
			}
			if len(streamLabelUnit[v].DataName) == 0 {
				response.Fail(ctx, "数据名称为空", nil)
				return
			}
			if len(streamLabelUnit[v].Formula) == 0 {
				response.Fail(ctx, "标签处理公式为空", nil)
				return
			}
			if len(streamLabelUnit[v].Unformula) == 0 {
				response.Fail(ctx, "标签逆处理公式为空", nil)
				return
			}
		}
		//根据label_id修改
		err := streamDao.UpdateStreamLabelUnit(streamLabelUnit[0])
		if err != nil {
			response.Fail(ctx, "修改失败", gin.H{"error": err})
			return
		}
		response.Success(ctx, "修改成功", nil)

	case "delete":
		if len(streamLabelUnit) == 0 {
			response.Fail(ctx, "标签ID为空", nil)
			return
		}
		err := streamDao.DeleteStreamLabelUnit(streamLabelUnit)
		if err != nil {
			response.Fail(ctx, "删除失败", gin.H{"error": err})
			return
		}
		response.Success(ctx, "删除成功", nil)
	default:

		response.Fail(ctx, "功能指令错误", nil)
		return

	}

}

// FindPage 数据池的分页查询加关键字搜索功能
func (s StreamCtrler) FindPage(ctx *gin.Context) {
	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	var pageInfo model.StreamPageSearch
	if err := ctx.BindJSON(&pageInfo); err != nil {
		response.Fail(ctx, "参数错误", gin.H{"error": err})
		return
	}
	streamDao := dao.NewStreamDAO(ctx)

	data := streamDao.FindPage(pageInfo, userInfo)
	total := streamDao.GetFindPageTotal(pageInfo, userInfo)

	response.Success(ctx, "查询成功", gin.H{"stream": data, "total": total})
}

// Delete 删除数据池
func (s StreamCtrler) Delete(ctx *gin.Context) {

	/* 获取流信息 */

	streamInfoCache, _ := ctx.Get("streamInfo")
	streamInfo := streamInfoCache.(model.Stream)
	streamDao := dao.NewStreamDAO(ctx)
	err := streamDao.Delete(streamInfo)
	if err != nil {
		response.Fail(ctx, "删除失败", nil)
		return
	}
	response.Success(ctx, "删除成功", nil)
}

// Update 修改数据池信息
func (s StreamCtrler) Update(ctx *gin.Context) {
	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rStream model.Stream
	ctx.Bind(&rStream)
	if 0 == len(rStream.ApiKey) {
		response.Fail(ctx, "格式错误", nil)
		return
	}
	if userInfo.Isadmin == 1 {
		if len(rStream.Admin) == 0 {
			rStream.Admin = "无"
		}
	} else {
		rStream.Admin = userInfo.Username
	}
	streamDao := dao.NewStreamDAO(ctx)
	err := streamDao.Update(rStream)
	if err != nil {
		response.Fail(ctx, "修改失败", gin.H{"error": err})
		return
	}
	response.Success(ctx, "修改成功", nil)
}

func (s StreamCtrler) Create(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 获取信息 */

	var rStream model.Stream
	ctx.Bind(&rStream)

	if len(rStream.Name) == 0 {
		response.Fail(ctx, "格式错误", nil)
		return
	}

	if userInfo.Isadmin == 1 {
		if rStream.Admin == "" {
			rStream.Admin = "无"
		}

	} else {
		rStream.Admin = userInfo.TrueName
	}

	/* 验证流存在 */
	streamDao := dao.NewStreamDAO(ctx)

	_, res := streamDao.ExistName(rStream)
	if res {
		response.Fail(ctx, "流名称已存在", nil)
		return
	}

	/* 添加数据库 */
	rStream.CreateUserId = int(userInfo.ID)
	rStream.ApiKey = rStream.GetStreamApiKey()
	rs, err := streamDao.AddStream(rStream)
	if nil != err {
		log.Println("[StreamCtrler] Create rStream.Create", err)
		response.Fail(ctx, "系统错误", gin.H{"error": err})
		return
	}

	/* 返回成功 */

	response.Success(ctx, "创建成功", gin.H{"data": rs})
}
