package middleware

import (
	"github.com/gin-gonic/gin"
)

// 切换数据源
func SwitchDBMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		/* 获取数据库信息 */
		dbName := ctx.Query("renter")

		if len(dbName) != 0 {
			ctx.Set("dbName", dbName)
		}

		ctx.Next()
	}
}
