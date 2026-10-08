package middleware

import (
	"bytes"
	"dicloud/internal/common/dao"
	"dicloud/models/model"
	"dicloud/pkg/requestctx"
	"dicloud/util/response"
	"encoding/json"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SystemLogOption func(*systemLogOptions)

type systemLogOptions struct {
	allowAnonymous bool
}

func AllowAnonymousLog() SystemLogOption {
	return func(options *systemLogOptions) {
		options.allowAnonymous = true
	}
}

/** 操作日志保存 */
func SystemLogMiddleware(content string, opts ...SystemLogOption) gin.HandlerFunc {
	options := systemLogOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	return func(ctx *gin.Context) {

		requestParams := getRequestParams(ctx)
		ctx.Next()

		/* 获取用户信息 */

		userInfo, hasUserInfo := resolveSystemLogUser(ctx, options.allowAnonymous)
		if !hasUserInfo && !options.allowAnonymous {
			return
		}

		/* 保存操作信息 */
		syslog := model.SystemLog{}
		syslog.UserId = int(userInfo.ID)
		syslog.Username = userInfo.Username
		syslog.TrueName = userInfo.TrueName
		if len(syslog.Username) == 0 && options.allowAnonymous {
			syslog.Username = extractAnonymousLogName(requestParams)
		}
		if len(syslog.TrueName) == 0 && len(syslog.Username) > 0 {
			syslog.TrueName = syslog.Username
		}

		syslog.LogTime = time.Now().Unix()
		syslog.PlatId = 0
		if requestPlatId, ok := requestctx.GetRequestPlatID(ctx); ok {
			syslog.PlatId = requestPlatId
		} else if userInfo.PlatId > 0 {
			syslog.PlatId = userInfo.PlatId
		}
		syslog.IpAddress = getClientIP(ctx)
		if len(syslog.IpAddress) == 0 {
			syslog.IpAddress = "-"
		}

		syslog.LogContent = content
		// fmt.Println("[Syslog Middleware]", content)

		syslog.OperateDetails = buildSystemLogDetails(ctx, requestParams)

		systemDao := dao.NewSystemLogDao(ctx)
		systemDao.Create(syslog)
	}
}

// 获取参数集合
func getRequestParams(ctx *gin.Context) any {
	var bindParams any = map[string]any{}
	if ctx.Request.Method == "POST" {
		contextType := ctx.Request.Header.Get("Content-Type")
		requestBody, err := ctx.GetRawData()
		if err != nil {
			log.Println("[SYS-MID] GetRawData,err: ", err)
			return bindParams
		}
		// 重置请求体，以便后续的中间件和处理程序可以正常读取
		ctx.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		defer func() {
			ctx.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}()
		// 绑定参数
		if strings.Contains(contextType, "application/json") {
			if len(bytes.TrimSpace(requestBody)) == 0 {
				return bindParams
			}
			if err := json.Unmarshal(requestBody, &bindParams); err != nil { // 报错
				log.Println("[SYS-MID] 获取参数异常: ", err)
				return bindParams
			}
		} else {
			formParams := map[string]any{}
			_ = ctx.Request.ParseMultipartForm(32 << 20)
			if len(ctx.Request.PostForm) > 0 {
				for k, v := range ctx.Request.PostForm {
					formParams[k] = v[0]
				}
			}
			bindParams = formParams
		}
	}
	ctx.Set("RequestParams", bindParams)

	return bindParams
}

func buildSystemLogDetails(ctx *gin.Context, requestParams any) string {
	httpStatus := getContextInt(ctx, response.ContextResponseHTTPStatus, ctx.Writer.Status())
	if httpStatus == 0 {
		httpStatus = 200
	}
	responseCode := getContextInt(ctx, response.ContextResponseCode, httpStatus)
	responseMsg := getContextString(ctx, response.ContextResponseMsg)
	result := "success"
	if responseCode != 200 || httpStatus >= 400 || len(ctx.Errors) > 0 {
		result = "fail"
	}

	path := ctx.FullPath()
	if len(path) == 0 && ctx.Request != nil && ctx.Request.URL != nil {
		path = ctx.Request.URL.Path
	}

	details := map[string]any{
		"method":       ctx.Request.Method,
		"path":         path,
		"params":       sanitizeLogValue("", requestParams),
		"result":       result,
		"responseCode": responseCode,
		"responseMsg":  responseMsg,
		"httpStatus":   httpStatus,
	}

	dataTem, err := json.Marshal(details)
	if err != nil {
		log.Println("[SYS-MID] 转换JSON异常", err)
		return ""
	}
	return string(dataTem)
}

func getClientIP(ctx *gin.Context) string {
	// 获取X-Forwarded-For头部信息
	xff := ctx.Request.Header.Get("X-Real-IP,X-Forwarded-For")
	if xff != "" {
		// 可能会有多个IP，例如通过多个代理。通常第一个是客户端的真实IP
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}
	// 直接从RemoteAddr获取，适用于没有代理的情况或配置了正确的X-Forwarded-For
	return ctx.ClientIP()
}

func getContextInt(ctx *gin.Context, key string, fallback int) int {
	value, ok := ctx.Get(key)
	if !ok {
		return fallback
	}

	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return fallback
	}
}

func getContextString(ctx *gin.Context, key string) string {
	value, ok := ctx.Get(key)
	if !ok {
		return ""
	}
	if str, ok := value.(string); ok {
		return str
	}
	return ""
}

func resolveSystemLogUser(ctx *gin.Context, allowAnonymous bool) (model.User, bool) {
	userInfoCache, ok := ctx.Get("userInfo")
	if !ok {
		if !allowAnonymous {
			log.Println("[SYS-MID] userInfo missing, skip system log")
		}
		return model.User{}, false
	}

	switch userInfo := userInfoCache.(type) {
	case model.User:
		return userInfo, true
	case *model.User:
		if userInfo != nil {
			return *userInfo, true
		}
	}

	if !allowAnonymous {
		log.Println("[SYS-MID] userInfo invalid, skip system log")
	}
	return model.User{}, false
}

func extractAnonymousLogName(requestParams any) string {
	requestParamMap, ok := requestParams.(map[string]any)
	if !ok {
		return ""
	}
	for _, key := range []string{"username", "phone", "apiKey", "apikey", "ApiKey", "Apikey"} {
		if value, ok := requestParamMap[key]; ok {
			name := strings.TrimSpace(toLogString(value))
			if len(name) > 0 && name != "******" {
				return name
			}
		}
	}
	return ""
}

func toLogString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []string:
		if len(v) > 0 {
			return v[0]
		}
	default:
		return ""
	}
	return ""
}

func sanitizeLogValue(key string, value any) any {
	if isSensitiveLogKey(key) {
		return "******"
	}

	switch v := value.(type) {
	case map[string]any:
		data := make(map[string]any, len(v))
		for childKey, childValue := range v {
			data[childKey] = sanitizeLogValue(childKey, childValue)
		}
		return data
	case []any:
		data := make([]any, 0, len(v))
		for _, item := range v {
			data = append(data, sanitizeLogValue("", item))
		}
		return data
	default:
		return value
	}
}

func isSensitiveLogKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "password", "oldpassword", "newpassword", "token", "authorization", "code", "codeid", "accesspassword", "controlpassword", "clientsecret", "secret":
		return true
	default:
		return false
	}
}
