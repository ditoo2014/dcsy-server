package controller

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/spf13/viper"
)

const defaultMaintainImageLimit = int64(5 << 20)

// 只接收服务生成的随机名及旧版时间戳名，禁止路径分隔符、盘符和隐藏文件。
var maintainImageNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+\.(?i:png|jpe?g|gif)$`)

func maintainUploadDirectory() string {
	directory := strings.TrimSpace(viper.GetString("uploads.directory"))
	if directory == "" {
		directory = "./uploads"
	}
	return directory
}

func maintainImageLimit() int64 {
	megabytes := viper.GetInt64("uploads.max_size_mb")
	if megabytes <= 0 || megabytes > 100 {
		return defaultMaintainImageLimit
	}
	return megabytes << 20
}

// validateMaintainImage 检查实际图像格式、文件大小和像素上限，不只依赖扩展名。
func validateMaintainImage(data []byte, filename string, maximum int64) (string, error) {
	if len(data) == 0 || int64(len(data)) > maximum {
		return "", errors.New("图片为空或超过允许大小")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return "", errors.New("图片内容无效或分辨率过大")
	}
	extension := strings.ToLower(filepath.Ext(filename))
	switch format {
	case "jpeg":
		if extension != ".jpg" && extension != ".jpeg" {
			return "", errors.New("图片扩展名与内容不一致")
		}
		return ".jpg", nil
	case "png", "gif":
		if extension != "."+format {
			return "", errors.New("图片扩展名与内容不一致")
		}
		return extension, nil
	default:
		return "", errors.New("只支持 PNG、JPEG、GIF 图片")
	}
}

// Upload 上传图片并返回同源地址，保留原前端使用的 data.url 响应格式。
func Upload(ctx *gin.Context) {
	maximum := maintainImageLimit()
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, maximum+(1<<20))
	file, err := ctx.FormFile("image")
	if ctx.Request.MultipartForm != nil {
		defer ctx.Request.MultipartForm.RemoveAll()
	}
	if err != nil || file.Size > maximum {
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": "图片未上传或超过允许大小"})
		return
	}
	source, err := file.Open()
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无法读取上传图片"})
		return
	}
	defer source.Close()
	data, err := io.ReadAll(io.LimitReader(source, maximum+1))
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": "读取图片失败"})
		return
	}
	extension, err := validateMaintainImage(data, file.Filename, maximum)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": err.Error()})
		return
	}
	directory := maintainUploadDirectory()
	if err := os.MkdirAll(directory, 0750); err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无法创建图片保存目录"})
		return
	}
	filename := uuid.NewString() + extension
	// 排他创建避免覆盖任何已有图片；保存失败时只清理本次新建的随机文件。
	destination := filepath.Join(directory, filename)
	output, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0640)
	if err != nil {
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": "无法保存图片"})
		return
	}
	_, writeErr := output.Write(data)
	closeErr := output.Close()
	if writeErr != nil || closeErr != nil {
		_ = os.Remove(destination)
		ctx.JSON(http.StatusOK, gin.H{"code": 400, "msg": "保存图片失败"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"code": 200, "msg": "上传成功", "data": gin.H{"url": "/go/images/file/" + filename}})
}

// openMaintainImage 使用白名单文件名与真实路径校验，禁止目录穿越和目录外符号链接。
func openMaintainImage(filename string) (*os.File, error) {
	if !maintainImageNamePattern.MatchString(filename) {
		return nil, errors.New("图片文件名无效")
	}
	root, err := filepath.Abs(maintainUploadDirectory())
	if err != nil {
		return nil, err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, err
	}
	target, err := filepath.EvalSymlinks(filepath.Join(root, filename))
	if err != nil {
		return nil, err
	}
	relative, err := filepath.Rel(root, target)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, errors.New("图片不在上传目录内")
	}
	file, err := os.Open(target)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > maintainImageLimit() {
		file.Close()
		return nil, errors.New("图片文件无效或超过允许大小")
	}
	return file, nil
}

// ReadUploadedImage 提供安全的同源图片读取，浏览器无需使用旧服务器域名。
func ReadUploadedImage(ctx *gin.Context) {
	file, err := openMaintainImage(ctx.Param("filename"))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "图片不存在"})
		return
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		ctx.Status(http.StatusNotFound)
		return
	}
	ctx.Header("X-Content-Type-Options", "nosniff")
	ctx.Header("Cache-Control", "private, max-age=3600")
	extension := strings.ToLower(filepath.Ext(ctx.Param("filename")))
	contentType := "image/jpeg"
	if extension == ".png" {
		contentType = "image/png"
	} else if extension == ".gif" {
		contentType = "image/gif"
	}
	ctx.Header("Content-Type", contentType)
	http.ServeContent(ctx.Writer, ctx.Request, ctx.Param("filename"), info.ModTime(), file)
}

// readMaintainImage 从兼容的新旧 URL 提取文件名；只读取本地文件，不进行网络请求。
func readMaintainImage(imageURL string) ([]byte, string, error) {
	parsed, err := url.Parse(imageURL)
	if err != nil {
		return nil, "", err
	}
	if strings.Contains(parsed.Path, "\\") {
		return nil, "", errors.New("图片路径无效")
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == ".." {
			return nil, "", errors.New("图片路径无效")
		}
	}
	filename := path.Base(parsed.Path)
	file, err := openMaintainImage(filename)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maintainImageLimit()+1))
	if err != nil {
		return nil, "", err
	}
	extension, err := validateMaintainImage(data, filename, maintainImageLimit())
	if err != nil {
		return nil, "", fmt.Errorf("运维图片无效: %w", err)
	}
	return data, extension, nil
}
