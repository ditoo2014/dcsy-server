package response

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xuri/excelize/v2"
)

const (
	ContextResponseHTTPStatus = "responseHTTPStatus"
	ContextResponseCode       = "responseCode"
	ContextResponseMsg        = "responseMsg"
)

func Response(ctx *gin.Context, httpStatus int, code int, ret gin.H, msg string) {
	ctx.Set(ContextResponseHTTPStatus, httpStatus)
	ctx.Set(ContextResponseCode, code)
	ctx.Set(ContextResponseMsg, msg)
	ctx.JSON(httpStatus, gin.H{"code": code, "return": ret, "msg": msg})
}

func Success(ctx *gin.Context, msg string, ret gin.H) {
	Response(ctx, http.StatusOK, 200, ret, msg)
}

func Fail(ctx *gin.Context, msg string, ret gin.H) {
	Response(ctx, http.StatusOK, 400, ret, msg)
}

func ResponseCode(ctx *gin.Context, httpStatus int, ret gin.H, msg string) {
	ctx.Set(ContextResponseHTTPStatus, httpStatus)
	ctx.Set(ContextResponseCode, httpStatus)
	ctx.Set(ContextResponseMsg, msg)
	ctx.JSON(httpStatus, gin.H{"code": httpStatus, "return": ret, "msg": msg})
}

func ResponseFileStream(ctx *gin.Context, fileName string, file *excelize.File) {
	ctx.Writer.Header().Add("Content-Type", "application/octet-stream")
	ctx.Writer.Header().Add("Content-Disposition", "attachment;filename="+fileName)
	ctx.Writer.Header().Add("Content-Transfer-Encoding", "binary")
	_ = file.Write(ctx.Writer)

	// 其他办法
	// xlsxData, err := file.WriteToBuffer()
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	// ctx.Data(http.StatusOK, "application/octet-stream", xlsxData.Bytes())
}

func ResponseFile(ctx *gin.Context, fileName string, file *excelize.File) {
	// ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"; filename*=UTF-8''%s", fileName, fileName))
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	ctx.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	ctx.File(fileName)

	// 保存文件到内存中，以便可以直接作为HTTP响应返回
	// xlsxData, err := file.WriteToBuffer()
	// if err != nil {
	// 	ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	// 	return
	// }
	// ctx.Data(http.StatusOK, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", xlsxData.Bytes())

	// 其他办法
	var buffer bytes.Buffer
	_ = file.Write(&buffer)
	content := bytes.NewReader(buffer.Bytes())
	http.ServeContent(ctx.Writer, ctx.Request, fileName, time.Now(), content)
}
