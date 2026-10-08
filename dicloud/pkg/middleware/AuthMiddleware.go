package middleware

import (
	"encoding/json"
	"fmt"
	"strings"

	"dicloud/internal/auth/dao"
	"dicloud/pkg/cache"
	"dicloud/pkg/common"
	"dicloud/util/response"

	"dicloud/models/model"

	"github.com/gin-gonic/gin"
)

// 登录信息校验
func AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var err error

		/* 获取authorization header */

		tokenString := ctx.GetHeader("authorization")

		/* 获取validate token formate */

		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			// 前端无法处理ResponseCode, 修改
			// response.ResponseCode(ctx, 401, nil, "auth header格式错误")
			response.Fail(ctx, "auth header格式错误", nil)
			ctx.Abort()
			return
		}

		tokenString = tokenString[7:]

		token, claims, err := common.ParseToken(tokenString)
		if err != nil || !token.Valid {
			// response.ResponseCode(ctx, 401, nil, "token过期或未启用")
			response.Fail(ctx, "token过期或未启用", nil)
			ctx.Abort()
			return
		}

		/* 获取数据库 */

		var dbUser = model.User{}
		tokenres := false

		// 查询redis, 用户信息
		result := cache.HMGet(ctx, fmt.Sprintf("user:%s", claims.UserName), "info")
		if len(result) > 0 {
			if dbUserStr, ok := result[0].(string); ok {
				json.Unmarshal([]byte(dbUserStr), &dbUser)
				if tokenString == dbUser.Token {
					// redis校验成功
					tokenres = true
				}
			}
		}

		// 查询数据库
		if !tokenres {
			var userDao = dao.NewUserDAO(ctx)
			dbUser, tokenres = userDao.ExistNameAndToken(claims.UserName, tokenString)
			if !tokenres {
				// response.ResponseCode(ctx, 401, nil, "token验证失败")
				response.Fail(ctx, "token验证失败", nil)
				ctx.Abort()
				return
			}
			cache.HMSet(ctx, fmt.Sprintf("user:%s", claims.UserName), map[string]interface{}{
				"info": dbUser,
			})
		}

		/* 插入用户信息 */

		if !ensureActiveSession(ctx, claims.UserName, tokenString) {
			return
		}

		ctx.Set("userInfo", dbUser)

		ctx.Next()
	}
}

// 登录校验同时绑定用户平台信息
func AuthWithPlatMiddleware(platId int) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		var err error

		/* 获取authorization header */

		tokenString := ctx.GetHeader("authorization")

		/* 获取validate token formate */

		if tokenString == "" || !strings.HasPrefix(tokenString, "Bearer ") {
			// 前端无法处理ResponseCode, 修改
			// response.ResponseCode(ctx, 401, nil, "auth header格式错误")
			response.Fail(ctx, "auth header格式错误", nil)
			ctx.Abort()
			return
		}

		tokenString = tokenString[7:]

		token, claims, err := common.ParseToken(tokenString)
		if err != nil || !token.Valid {
			// response.ResponseCode(ctx, 401, nil, "token过期或未启用")
			response.Fail(ctx, "token过期或未启用", nil)
			ctx.Abort()
			return
		}

		/* 获取数据库 */

		var dbUser = model.User{}
		tokenres := false

		// 查询redis, 用户信息
		result := cache.HMGet(ctx, fmt.Sprintf("user:%s", claims.UserName), "info")
		if len(result) > 0 {
			if dbUserStr, ok := result[0].(string); ok {
				json.Unmarshal([]byte(dbUserStr), &dbUser)
				if tokenString == dbUser.Token {
					// redis校验成功
					tokenres = true
				}
			}
		}

		// 查询数据库
		if !tokenres {
			var userDao = dao.NewUserDAO(ctx)
			dbUser, tokenres = userDao.ExistNameAndToken(claims.UserName, tokenString)
			if !tokenres {
				// response.ResponseCode(ctx, 401, nil, "token验证失败")
				response.Fail(ctx, "token验证失败", nil)
				ctx.Abort()
				return
			}

			dbUser.Password = ""
			cache.HMSet(ctx, fmt.Sprintf("user:%s", claims.UserName), map[string]interface{}{
				"info": dbUser,
			})
		}

		/* 插入用户信息 */

		if !ensureActiveSession(ctx, claims.UserName, tokenString) {
			return
		}

		dbUser.PlatId = platId
		ctx.Set("userInfo", dbUser)

		ctx.Next()
	}
}
