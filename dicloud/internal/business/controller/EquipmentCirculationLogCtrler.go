package controller

import (
	authdao "dicloud/internal/auth/dao"
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/common/service"
	usermodel "dicloud/models/model"
	"dicloud/pkg/cache"
	"dicloud/util/response"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

type IEquipmentCirculationLogCtrler interface {
	Create(ctx *gin.Context)
	Find(ctx *gin.Context)
}

type EquipmentCirculationLogCtrler struct {
}

// 记录设备流转记录
func (eclCtrler EquipmentCirculationLogCtrler) Create(ctx *gin.Context) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	userInfo := userInfoCache.(usermodel.User)

	/* 绑定数据 */

	var iEcl = model.EquipmentCirculationLog{}
	ctx.Bind(&iEcl)

	/* 鉴权 */
	if iEcl.OperateType == 0 {
		log.Println("[EquipmentCirculationLogCtrler] unknown operate")
		response.Fail(ctx, "非法操作", nil)
		return
	}
	permissionUrl := fmt.Sprintf("equipmentcirlog:add:%d", iEcl.OperateType)

	pass := equipmentCirculationLogAuthentication(ctx, permissionUrl)
	if !pass {
		// 鉴权失败
		response.Fail(ctx, "没有权限, 不能进行操作!", nil)
		return
	}

	/* 校验数据 */
	if iEcl.EquipmentId == 0 {
		log.Println("[EquipmentCirculationCtrler] equipment id is null")
		response.Fail(ctx, "无效数据", nil)
		return
	}

	equipmentDao := dao.NewEquipmentDAO(ctx)
	_, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(iEcl.EquipmentId)})
	if !res {
		log.Println("[EquipmentCirculationCtrler] equipment id find fail")
		response.Fail(ctx, "无效数据", nil)
		return
	}

	// 填充数据
	if iEcl.OperateDetails == nil {
		iEcl.OperateDetails = []byte(`{}`)
	}
	iEcl.OperatorUserId = int(userInfo.ID)
	iEcl.CreateTime = time.Now().Unix()

	// 创建
	equipmentCirLogDao := dao.NewEquipmentCirculationLogDao(ctx)
	_, res = equipmentCirLogDao.Create(iEcl)
	if !res {
		log.Println("[EquipmentCirculationCtrler] create fail")
		response.Fail(ctx, "记录失败", nil)
		return
	}

	response.Success(ctx, "记录成功", nil)
}

// 查询设备流转记录
func (eclCtrler EquipmentCirculationLogCtrler) Find(ctx *gin.Context) {

	/* 获取并校验用户信息，避免未登录请求触发类型断言异常。 */
	userInfoCache, exists := ctx.Get("userInfo")
	userInfo, validUser := userInfoCache.(usermodel.User)
	if !exists || !validUser || userInfo.ID == 0 {
		response.Fail(ctx, "用户信息无效", nil)
		return
	}

	/* 绑定数据 */

	var iEcl = model.EquipmentCirculationLog{}
	if err := ctx.ShouldBind(&iEcl); err != nil {
		response.Fail(ctx, "请求参数错误", nil)
		return
	}
	// 流转记录必须绑定具体设备，禁止使用设备编号0退化为查询全部记录。
	if iEcl.EquipmentId <= 0 {
		response.Fail(ctx, "设备ID无效", nil)
		return
	}

	equipmentDao := dao.NewEquipmentDAO(ctx)
	equipmentCirLogDao := dao.NewEquipmentCirculationLogDao(ctx)
	if userInfo.Isadmin == 1 {
		// 管理员也必须先确认目标设备存在，查询范围始终限定在该设备。
		dataTem, res := equipmentDao.FindEquipmentById(model.Equipment{ID: uint(iEcl.EquipmentId)})
		if !res || dataTem.ID == 0 {
			response.Success(ctx, "设备不存在", nil)
			return
		}

		dataList, res := equipmentCirLogDao.Find(iEcl)

		if !res {
			log.Println("[EquipmentCirculationCtrler] find fail")
			response.Fail(ctx, "", nil)
			return
		}

		response.Success(ctx, "success", gin.H{"data": dataList})
	} else {

		// 查询用户部门。没有部门的普通用户没有设备可见范围。
		var corpIds []int
		for _, v := range userInfo.CorpIds {
			// 未分配部门标记不是组织根节点，不能从 0 展开到全部部门。
			if v > 0 {
				corpIds = append(corpIds, int(v))
			}
		}
		// 子部门查询
		corpIds = service.FindAllSonCorp(ctx, corpIds)
		var hasCorpScope bool
		corpIds, hasCorpScope = circulationLogCorpScope(corpIds)
		if !hasCorpScope {
			response.Success(ctx, "设备不存在", nil)
			return
		}

		dataTem, res := equipmentDao.FindEquipmentByIdAndCorp(model.Equipment{ID: uint(iEcl.EquipmentId)}, corpIds)
		if !res || dataTem.ID == 0 {
			response.Success(ctx, "设备不存在", nil)
			return
		}

		dataList, res := equipmentCirLogDao.Find(iEcl)
		if !res {
			log.Println("[EquipmentCirculationCtrler] find fail")
			response.Fail(ctx, "", nil)
			return
		}

		response.Success(ctx, "success", gin.H{"data": dataList})
	}

}

// circulationLogCorpScope 生成流转记录查询的部门范围。
// 普通用户必须至少拥有一个真实部门，且与设备列表保持一致，允许查看未分配部门（部门编号0）的设备。
func circulationLogCorpScope(corpIds []int) ([]int, bool) {
	seen := make(map[int]struct{}, len(corpIds)+1)
	scope := make([]int, 0, len(corpIds)+1)
	for _, corpId := range corpIds {
		if corpId <= 0 {
			continue
		}
		if _, exists := seen[corpId]; exists {
			continue
		}
		seen[corpId] = struct{}{}
		scope = append(scope, corpId)
	}
	if len(scope) == 0 {
		return nil, false
	}
	seen[0] = struct{}{}
	return append(scope, 0), true
}

func equipmentCirculationLogAuthentication(ctx *gin.Context, permissionUrl string) (pass bool) {

	/* 获取用户信息 */

	userInfoCache, _ := ctx.Get("userInfo")
	dbUser := userInfoCache.(usermodel.User)

	/* 获取数据库权限 鉴权 */

	pass = true
	if dbUser.Isadmin != 1 {
		// 断言(未授权)
		pass = false
		// 查询redis
		result := cache.HMGet(ctx, fmt.Sprintf("user:%s", dbUser.Username), fmt.Sprintf("permission%v", 1))
		if len(result) > 0 {
			if dbUserStr, ok := result[0].(string); ok {
				var pmns usermodel.Permissions
				json.Unmarshal([]byte(dbUserStr), &pmns)
				for _, p := range pmns.Data {
					if p.Url == permissionUrl {
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
			// 查询数据库授权的权限
			var permissionDao = authdao.NewPermissionDAO(ctx)
			dataDtos, _ := permissionDao.GetUserPermissions(dbUser, 1)
			// 存入redis
			var permissions = usermodel.Permissions{}
			permissions.Data = dataDtos
			cache.HMSet(ctx, fmt.Sprintf("user:%s", dbUser.Username), map[string]interface{}{
				fmt.Sprintf("permission%v", 1): permissions,
			})

			for i := 0; i < len(dataDtos); i++ {
				if dataDtos[i].Url == permissionUrl {
					// 鉴权成功
					pass = true
					break
				}
			}
			if !pass {
				// 鉴权失败
				response.Fail(ctx, "没有权限, 不能进行操作!", nil)
				return
			}
		}
	}

	return
}
