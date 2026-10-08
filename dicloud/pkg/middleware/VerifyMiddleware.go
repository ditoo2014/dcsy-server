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

/* 鉴权 该用户在某平台 是否有权限 -- 并绑定平台信息 */
func VerifyMiddleware(permissionUrl string, platId int) gin.HandlerFunc {
	return VerifyAnyMiddleware([]string{permissionUrl}, platId)
}

// VerifyAnyMiddleware allows an endpoint to be shared by several permission paths.
func VerifyAnyMiddleware(permissionUrls []string, platId int) gin.HandlerFunc {
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

		/* 获取数据库用户 */

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

		/* 获取数据库权限 鉴权 */

		if !ensureActiveSession(ctx, claims.UserName, tokenString) {
			return
		}

		if dbUser.Isadmin != 1 {
			permissionSet := make(map[string]struct{}, len(permissionUrls))
			for _, permissionUrl := range permissionUrls {
				if permissionUrl != "" {
					permissionSet[permissionUrl] = struct{}{}
				}
			}
			// 断言(未授权)
			pass := false
			// 查询redis
			result := cache.HMGet(ctx, fmt.Sprintf("user:%s", dbUser.Username), fmt.Sprintf("permission%v", platId))
			if len(result) > 0 {
				if dbUserStr, ok := result[0].(string); ok {
					var pmns model.Permissions
					json.Unmarshal([]byte(dbUserStr), &pmns)
					for _, p := range pmns.Data {
						if _, required := permissionSet[p.Url]; required {
							// 鉴权成功
							pass = true
							break
						} else {
							continue
						}
					}
				}
			}

			if !pass {
				// fmt.Println("search database dbuser: %v", dbUser)
				// data, _ := dbUser.MarshalBinary()
				// fmt.Printf("search database dbuserStr: %v \n", string(data))

				// 查询数据库授权的权限
				var permissionDao = dao.NewPermissionDAO(ctx)
				dataDtos, _ := permissionDao.GetUserPermissions(dbUser, platId)
				// 存入redis
				var permissions = model.Permissions{}
				permissions.Data = dataDtos
				cache.HMSet(ctx, fmt.Sprintf("user:%s", dbUser.Username), map[string]interface{}{
					fmt.Sprintf("permission%v", platId): permissions,
				})

				for i := 0; i < len(dataDtos); i++ {
					if _, required := permissionSet[dataDtos[i].Url]; required {
						// 鉴权成功
						pass = true
						break
					}
				}
				if !pass {
					// 鉴权失败
					response.Fail(ctx, "没有权限, 不能进行操作!", nil)
					ctx.Abort()
					return
				}
			}
		}

		/* 插入用户信息 */

		dbUser.PlatId = platId
		ctx.Set("userInfo", dbUser)

		ctx.Next()
	}
}
