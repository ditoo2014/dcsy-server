package controller

import (
	"dicloud/internal/auth/dao"
	authmodel "dicloud/internal/auth/model"
	"dicloud/internal/common/service"
	"dicloud/models/model"
	"dicloud/util/response"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

type ICorpCtrler interface {
	List(ctx *gin.Context)
	AddCorp(ctx *gin.Context)
	DeleteCorp(ctx *gin.Context)
	UpdateCorp(ctx *gin.Context)
	FindCorp(ctx *gin.Context)
	ExportExcel(ctx *gin.Context)
	ImportExcel(ctx *gin.Context)
}

type CorpCtrler struct {
}

// 查询全部分组数据
func (c CorpCtrler) List(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	// 查询全部
	corpDao := dao.NewCorpDAO(ctx)
	data, err := corpDao.List(userInfo.PlatId)
	if err != nil {
		response.Fail(ctx, "查询失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

// 添加分组
func (c CorpCtrler) AddCorp(ctx *gin.Context) {

	// 绑定平台信息
	var corp = model.Corp{}
	ctx.Bind(&corp)

	/* 校验数据 */

	corpDao := dao.NewCorpDAO(ctx)
	_, res := corpDao.ExistName(corp)
	if res {
		response.Fail(ctx, "已存在同名分组", nil)

		return
	}

	// 添加
	data, res := corpDao.Add(corp)
	if !res {
		response.Fail(ctx, "保存失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

// 删除分组
func (c CorpCtrler) DeleteCorp(ctx *gin.Context) {
	var corp = model.Corp{}
	ctx.Bind(&corp)

	/* 校验 */

	corpDao := dao.NewCorpDAO(ctx)
	_, res := corpDao.ExistId(corp)
	if !res {
		response.Fail(ctx, "不存在的数据", nil)

		return
	}

	// 查询用户是否绑定部门
	userDao := dao.NewUserDAO(ctx)
	dataTem, _, _ := userDao.FindByExactCorpId(model.User{}, "", append([]int{}, int(corp.ID)), 0, 1)
	if len(dataTem) > 0 {
		response.Fail(ctx, "请先将用户解绑再删除", nil)
		return
	}

	// 删除
	data, res := corpDao.Delete(corp)
	if !res {
		response.Fail(ctx, "删除失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})
}

// 更新分组
func (c CorpCtrler) UpdateCorp(ctx *gin.Context) {

	// 绑定平台信息
	var corp = model.Corp{}
	ctx.Bind(&corp)

	/* 校验信息 */

	corpDao := dao.NewCorpDAO(ctx)
	_, res := corpDao.ExistName(corp)
	if res {
		response.Fail(ctx, "已存在同名分组", nil)

		return
	}

	_, res = corpDao.ExistId(corp)
	if !res {
		response.Fail(ctx, "不存在的数据", nil)

		return
	}

	if corp.ID == uint(corp.ParentId) {
		corp.ParentId = 0
	}

	// 更新
	data, res := corpDao.Update(corp)
	if !res {
		response.Fail(ctx, "删除失败", nil)

		return
	}

	response.Success(ctx, "success", gin.H{"data": data})

}

// 查询分组
func (c CorpCtrler) FindCorp(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	// 绑定平台信息
	var corp = model.Corp{}
	ctx.Bind(&corp)
	corp.PlatId = userInfo.PlatId

	/* 校验信息 */

	corpDao := dao.NewCorpDAO(ctx)

	if userInfo.Isadmin == 1 {

		data, res := corpDao.FindCorpList(corp, userInfo.PlatId)
		if !res {
			response.Fail(ctx, "无查询结果", nil)
			return
		}
		response.Success(ctx, "success", gin.H{"data": data})
	} else {

		if len(userInfo.CorpIds) > 0 {
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}

			// 子部门查询
			corpTem := service.FindAllSonCorp(ctx, corpIds)
			// 父部门查询
			corpIds = service.FindAllParentCorp(ctx, corpIds)
			// 所有管理部门 - 向上延伸, 向下拓展
			corpIds = append(corpIds, corpTem...)

			dataList, res := corpDao.FindByIds(corp, corpIds)
			if !res {
				response.Fail(ctx, "无查询结果", nil)
				return
			}
			response.Success(ctx, "success", gin.H{"data": dataList})
		} else {
			response.Fail(ctx, "请通知管理员分配部门, 无部门权限", nil)
			return
		}
	}
}

// 导出部门
func (r CorpCtrler) ExportExcel(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(model.User)

	// 绑定平台信息
	var corp = model.Corp{}
	ctx.Bind(&corp)
	corp.PlatId = userInfo.PlatId

	/* 获取信息 */

	corpDao := dao.NewCorpDAO(ctx)
	var corps = []model.Corp{}
	if userInfo.Isadmin == 1 {

		corps, _ = corpDao.FindCorpList(corp, userInfo.PlatId)
	} else {

		if len(userInfo.CorpIds) > 0 {
			var corpIds []int
			for _, v := range userInfo.CorpIds {
				corpIds = append(corpIds, int(v))
			}

			// 子部门查询
			corpTem := service.FindAllSonCorp(ctx, corpIds)
			// 父部门查询
			corpIds = service.FindAllParentCorp(ctx, corpIds)
			// 所有管理部门 - 向上延伸, 向下拓展
			corpIds = append(corpIds, corpTem...)

			corps, _ = corpDao.FindByIds(corp, corpIds)
		} else {
			response.Fail(ctx, "请通知管理员分配部门, 无部门权限", nil)
			return
		}
	}

	/* 导出 */
	if len(corps) > 0 {

		// 全部门ID对应名称
		var corpMap = make(map[int]string)
		corps, _ := corpDao.FindCorpList(model.Corp{}, 1)
		if len(corps) > 0 {
			for _, v := range corps {
				corpMap[int(v.ID)] = v.CorpName
			}
		}

		// 开始导出数据
		f := excelize.NewFile()
		index, err := f.NewSheet("Sheet1")
		if err != nil {
			log.Println("[Auth] excelize newFile fail: ", err)
			return
		}

		for i, v := range authmodel.CorpExportHeader {
			//rune('A'+i)：根据当前索引 i 计算出 Excel 表中的列名。rune('A') 是字符 'A' 的 Unicode 码点值，加上索引 i 后将其转换回字符，从而得到列名。例如，当 i=0 时，结果是 'A'；当 i=1 时，结果是 'B'，依此类推。
			cell := fmt.Sprintf("%s1", string(rune('A'+i)))
			f.SetCellValue("Sheet1", cell, v)
		}

		for idx, v := range corps {
			f.SetCellValue("Sheet1", fmt.Sprintf("A%d", idx+2), v.CorpName)

			// 部门
			if len(corpMap[v.ParentId]) > 0 {
				f.SetCellValue("Sheet1", fmt.Sprintf("B%d", idx+2), corpMap[v.ParentId])
			} else {
				f.SetCellValue("Sheet1", fmt.Sprintf("B%d", idx+2), 0)
			}

			f.SetCellValue("Sheet1", fmt.Sprintf("C%d", idx+2), v.Level)
		}

		f.SetActiveSheet(index)

		fileName := fmt.Sprintf("corp_export_%d.xlsx", time.Now().Unix())
		filePath := "/var/lib/dicloud/temp/" + fileName
		err = f.SaveAs(filePath)
		if err != nil {
			log.Println("[Auth] fail to save EXCEL file: ", err)
			response.Fail(ctx, "导出失败", nil)
			return
		}
		os.Chmod(filePath, 0777)
		// defer os.Remove(filePath)

		// 回写数据流办法
		// response.ResponseFileStream(ctx, fileName, f)

		response.Success(ctx, "导出成功", gin.H{"fileName": fileName})
		return
	}

	response.Fail(ctx, "导出失败", nil)
}

// 导入Excel数据
func (ctrler CorpCtrler) ImportExcel(ctx *gin.Context) {

	// 获取上传文件
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		log.Println("[Auth] upload file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer file.Close()

	// 临时文件处理
	filePath := fmt.Sprintf("corp_import_%d.xlsx", time.Now().Unix())
	// 创建临时文件
	tempFile, err := os.Create(filePath)
	if err != nil {
		log.Println("[Auth] create temp file fail", err)
		response.Fail(ctx, "上传文件出错", nil)
		return
	}
	defer os.Remove(filePath)
	defer tempFile.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		log.Println("[Auth] copy to temp file fail", err)
		response.Fail(ctx, "文件复制出错", nil)
		return
	}

	f, err := excelize.OpenFile(tempFile.Name())
	if err != nil {
		log.Println("[Auth] excelize open file fail", err)
		response.Fail(ctx, "无法打开文件", nil)
		return
	}

	defer f.Close()

	// 获取单元格的值

	// 获取第一个工作表的所有行
	rows, err := f.GetRows(f.GetSheetList()[0])
	if err != nil {
		response.Fail(ctx, "获取表数据失败", nil)
		return
	}

	// 全部门ID对应名称
	var corpMap = make(map[string]int)
	corpDao := dao.NewCorpDAO(ctx)
	corps, _ := corpDao.FindCorpList(model.Corp{}, 1)
	if len(corps) > 0 {
		for _, v := range corps {
			corpMap[v.CorpName] = int(v.ID)
		}
	}

	var problemCount = 0
	var dumplicate = 0
	failAddCount := 0
	failAddCorp := []model.Corp{}
	successCount := 0
	successCorp := []model.Corp{}
	var corpParentMap = make(map[string]string)

	// 二维表操作
	for idx, row := range rows {
		// 表头跳过
		if idx == 0 {
			continue
		}
		// 固定顺序导入数据
		var corpTem = model.Corp{}
		if len(row[0]) == 0 {
			problemCount = problemCount + 1
			continue
		}
		corpTem.CorpName = row[0]

		// 父节点
		if corpMap[row[1]] > 0 {
			corpTem.ParentId = corpMap[row[1]]
		} else {
			corpTem.ParentId = 0
		}

		corpTem.Level, _ = strconv.Atoi(row[2])

		// 保存
		/* 校验数据 */
		_, res := corpDao.ExistName(corpTem)
		if res {
			// 已存在同名部门
			dumplicate = dumplicate + 1
			continue
		}

		// 添加
		data, res := corpDao.Add(corpTem)
		if !res {
			failAddCount = failAddCount + 1
			failAddCorp = append(failAddCorp, data)
			continue
		}
		successCount = successCount + 1
		successCorp = append(successCorp, data)

		/* 父节点处理 */
		// 记录没有父节点的数据
		if data.ParentId == 0 {
			corpParentMap[data.CorpName] = row[1]
		}
		// 记录新数据名称对应id
		corpMap[data.CorpName] = int(data.ID)
	}

	// 关闭文件
	if err := f.Close(); err != nil {
		fmt.Println(err)
	}

	// 处理父节点
	for idx := range successCorp {
		// 保存时父节点id不存在, 后续数据中有保存
		if successCorp[idx].ParentId == 0 && corpMap[corpParentMap[successCorp[idx].CorpName]] > 0 {
			successCorp[idx].ParentId = corpMap[corpParentMap[successCorp[idx].CorpName]]
			// 更新
			data, res := corpDao.Update(successCorp[idx])
			if res {
				successCorp[idx].ParentId = data.ParentId
			}
		}
	}

	response.Success(ctx, "导入成功",
		gin.H{"successData": successCorp, "successCount": successCount,
			"failAddData": failAddCorp, "failAddCount": failAddCount,
			"problemCount": problemCount, "dumplicate": dumplicate})
}
