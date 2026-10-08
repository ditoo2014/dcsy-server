package service

import (
	authdao "alarm/internal/auth/dao"
	"alarm/internal/business/dao"
	"alarm/models/model"
	"alarm/pkg/cache"
	normalutil "alarm/util/normal"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	CORP_CACHE_KEY      = "corp:%d"         // corp:id - field:@son/parent - value:@id([]int)
	CORP_CACHE_ONEBYONE = "corp:parent:obo" // corp:parent:obo - field:@id(int) - value:@id(int)
)

func FindAllSonCorp(ctx *gin.Context, corpIds []int) []int {

	var resultCorpIds []int
	// 递归查询
	mapSearch := make(map[int]int)
	if len(corpIds) > 0 {
		for _, idTem := range corpIds {
			// 不重复查询
			if mapSearch[idTem] > 0 {
				continue
			}
			mapSearch[idTem] = idTem

			var corpIdsTem []int
			var existCache = false

			// 查缓存
			result := cache.HMGet(ctx, fmt.Sprintf(CORP_CACHE_KEY, idTem), "son")
			if len(result) > 0 {
				if dbCorpIdsStr, ok := result[0].(string); ok {
					json.Unmarshal([]byte(dbCorpIdsStr), &corpIdsTem)
					if len(corpIdsTem) > 0 {
						// redis校验成功
						existCache = true
						corpIdsTem = append(corpIdsTem, idTem)
						resultCorpIds = append(resultCorpIds, corpIdsTem...)
					}
				}
			}

			// 查数据库
			if !existCache {

				corpIdsTem = append(corpIdsTem, idTem)
				// 子部门查询
				corpDao := authdao.NewCorpDAO(ctx)
				dataTem, res := corpDao.FindAllSon(corpIdsTem)
				if res {
					// 本部门计入
					dataTem = append(dataTem, idTem)
					// 结果数据封装
					resultCorpIds = append(resultCorpIds, dataTem...)
					binData, err := json.Marshal(dataTem)
					if err == nil {
						cache.HMSet(ctx, fmt.Sprintf(CORP_CACHE_KEY, idTem),
							map[string]interface{}{
								"son": string(binData),
							})
						// 一分钟更新一次 -> 三分钟
						// 延长更新时间, 部门基本不会频繁变更
						cache.ExpireKey(ctx, fmt.Sprintf(CORP_CACHE_KEY, idTem), time.Second*60*3)
					}
				}
			}
		}
	}

	// 去重
	resultCorpIds = normalutil.Unique(resultCorpIds)
	return resultCorpIds
}

// 查询所有父节点, 展示数据
func FindAllParentCorp(ctx *gin.Context, corpIds []int) []int {

	var resultCorpIds []int
	// 递归查询
	mapSearch := make(map[int]int)
	if len(corpIds) > 0 {
		for _, idTem := range corpIds {
			// 不重复查询
			if mapSearch[idTem] > 0 {
				continue
			}
			mapSearch[idTem] = idTem

			var corpIdsTem []int
			var existCache = false

			// 查缓存
			result := cache.HMGet(ctx, fmt.Sprintf(CORP_CACHE_KEY, idTem), "parent")
			if len(result) > 0 {
				if dbCorpIdsStr, ok := result[0].(string); ok {
					json.Unmarshal([]byte(dbCorpIdsStr), &corpIdsTem)
					if len(corpIdsTem) > 0 {
						// redis校验成功
						existCache = true
						corpIdsTem = append(corpIdsTem, idTem)
						resultCorpIds = append(resultCorpIds, corpIdsTem...)
					}
				}
			}

			// 查数据库
			if !existCache {

				corpIdsTem = append(corpIdsTem, idTem)
				// 子部门查询
				corpDao := authdao.NewCorpDAO(ctx)
				dataTem, res := corpDao.FindAllParent(corpIdsTem)
				if res {
					// 本部门计入
					dataTem = append(dataTem, idTem)
					// 结果数据封装
					resultCorpIds = append(resultCorpIds, dataTem...)
					binData, err := json.Marshal(dataTem)
					if err == nil {
						cache.HMSet(ctx, fmt.Sprintf(CORP_CACHE_KEY, idTem),
							map[string]interface{}{
								"parent": string(binData),
							})
						// 一分钟更新一次
						cache.ExpireKey(ctx, fmt.Sprintf(CORP_CACHE_KEY, idTem), time.Second*60)
					}
				}
			}
		}
	}

	// 去重
	resultCorpIds = normalutil.Unique(resultCorpIds)
	return resultCorpIds
}

func FindAllCorpEquipmentId(ctx *gin.Context, corpIds []int) []int {

	var resultEquipmentIds []int
	// 递归查询
	mapSearch := make(map[int]int)
	if len(corpIds) > 0 {
		for _, corpIdTem := range corpIds {
			// 不重复查询
			if mapSearch[corpIdTem] > 0 {
				continue
			}
			mapSearch[corpIdTem] = corpIdTem

			var corpEquipmentIdsTem []int
			var existCache = false

			// 查缓存
			result := cache.HMGet(ctx, fmt.Sprintf(CORP_CACHE_KEY, corpIdTem), "equipmentIds")
			if len(result) > 0 {
				if dbCorpIdsStr, ok := result[0].(string); ok {
					json.Unmarshal([]byte(dbCorpIdsStr), &corpEquipmentIdsTem)
					if len(corpEquipmentIdsTem) > 0 {
						// redis校验成功
						existCache = true
						resultEquipmentIds = append(resultEquipmentIds, corpEquipmentIdsTem...)
					}
				}
			}

			// 查数据库
			if !existCache {

				corpIdArr := []int{corpIdTem}
				// 部门设备查询
				equipmentDao := dao.NewEquipmentDAO(ctx)
				dataTem, res := equipmentDao.FindAllEquipmentIds(corpIdArr)
				if res {
					// 结果数据封装
					resultEquipmentIds = append(resultEquipmentIds, dataTem...)
					binData, err := json.Marshal(dataTem)
					if err == nil {
						cache.HMSet(ctx, fmt.Sprintf(CORP_CACHE_KEY, corpIdTem),
							map[string]interface{}{
								"equipmentIds": string(binData),
							})
						// 一分钟更新一次 -> 三分钟
						// 延长更新时间, 部门基本不会频繁变更
						cache.ExpireKey(ctx, fmt.Sprintf(CORP_CACHE_KEY, corpIdTem), time.Second*60*3)
					}
				}
			}
		}
	}

	// 去重
	resultEquipmentIds = normalutil.Unique(resultEquipmentIds)
	return resultEquipmentIds
}

func FindParentCorpOneByOne(ctx *gin.Context, corpId int) int {

	var parentCorp int
	var existparent = false

	// 查缓存
	result := cache.HMGet(ctx, CORP_CACHE_ONEBYONE, fmt.Sprint(corpId))
	if len(result) > 0 {
		if dbCorpIdsStr, ok := result[0].(string); ok {
			json.Unmarshal([]byte(dbCorpIdsStr), &parentCorp)
			if parentCorp > 0 {
				// redis获取成功
				existparent = true
				return parentCorp
			}
		}
	}

	// 查数据库
	if !existparent {

		// 子部门查询
		corpDao := authdao.NewCorpDAO(ctx)
		dataTem, res := corpDao.Find(model.Corp{ID: uint(corpId)})
		if res {
			binData, err := json.Marshal(dataTem.ParentId)
			if err == nil {
				cache.HMSet(ctx, CORP_CACHE_ONEBYONE,
					map[string]interface{}{
						fmt.Sprint(corpId): string(binData),
					})
				// 一分钟更新一次
				cache.ExpireKey(ctx, CORP_CACHE_ONEBYONE, time.Second*60)
			}
		}
	}

	// 默认使用最高级节点, 的父节点0
	return parentCorp
}
