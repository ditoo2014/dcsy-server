package requestctx

import "github.com/gin-gonic/gin"

const requestPlatIDKey = "requestPlatId"

func SetRequestPlatID(ctx *gin.Context, platID int) {
	if ctx == nil {
		return
	}
	ctx.Set(requestPlatIDKey, platID)
}

func GetRequestPlatID(ctx *gin.Context) (int, bool) {
	if ctx == nil {
		return 0, false
	}

	value, exists := ctx.Get(requestPlatIDKey)
	if !exists {
		return 0, false
	}

	switch typed := value.(type) {
	case int:
		return typed, true
	case int64:
		return int(typed), true
	case float64:
		return int(typed), true
	default:
		return 0, false
	}
}

func GetRequestPlatIDOrDefault(ctx *gin.Context, defaultPlatID int) int {
	if platID, ok := GetRequestPlatID(ctx); ok {
		return platID
	}
	return defaultPlatID
}
