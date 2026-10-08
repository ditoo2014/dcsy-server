package schedule

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	commonservice "dicloud/internal/common/service"
	triggerservice "dicloud/internal/trigger/service"
	"dicloud/pkg/cache"
	bloomutil "dicloud/util/bloom"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// 刷新数据库数据, 触发器数据, 通过bloom过滤器过滤, 减少查库操作
func RefreshDatasource() {
	// The route refresh initializes configured renter datasources before the
	// normal bloom refresh enumerates them. Ordinary trigger handling remains
	// unchanged; this index is used only by the battery-specific RocketMQ tag.
	triggerservice.RefreshBatteryTriggerRoutes()

	/* 默认平台数据 - multi renter(多租户) */
	ctx := &gin.Context{}
	doRefresh(ctx)

	/* multi renter(多租户) */
	// 处理所有租户的定时统计
	dbNameDatas := commonservice.LoadingAllPlat()
	for _, dbTem := range dbNameDatas {
		ctxTem := &gin.Context{}
		ctxTem.Set("dbName", dbTem)
		doRefresh(ctxTem)
	}
}

// tigger模块自己加载数据库配置, 其他模块服务管理无效
func doRefresh(ctx *gin.Context) {

	// log.Println("[TRIGGER] schedule refresh triggerconfig data")
	timeTem := time.Now()
	if timeTem.Hour() == 3 {
		// 每天3点, 重新加载数据, 如果数据库有删除数据
		// 执行一次
		dateStr := time.Now().Format("2006-01-02-15")
		keyTem := "business:bloom:trigger:schedule:" + dateStr
		res, _ := cache.GetRedisClient(ctx).SetNX(keyTem, "1", 1*time.Hour).Result()
		if !res {
			return
		}
		log.Println("[TRIGGER] clean bloom cache")
		bloomutil.LocalBloomFilter.ClearAll()
		bloomutil.BusinessGeneralTriggerBloomFilter.ClearAll()
	}

	// 数据库数据
	triggerDao := dao.NewTriggerDao(ctx)
	datas, res := triggerDao.Find(model.TriggerConfig{})
	if !res {
		log.Println("[TRIGGER] schedule refresh triggerconfig fail - empty")
		refreshGeneralTriggerBloom(ctx)
		return
	}

	// 填充过滤器
	for _, v := range datas {
		bloomutil.LocalBloomFilter.Add([]byte(v.Marktrigger))
	}

	// 数据库数据
	generalTriggerDao := dao.NewGeneralTriggerDao(ctx)
	datasTem, res := generalTriggerDao.Find(model.GeneralTriggerConfig{})
	if !res {
		// log.Println("[TRIGGER] schedule refresh generaltriggerconfig fail - empty")
		return
	}

	// 填充过滤器 - 通用触发器数据
	for _, v := range datasTem {
		for _, mtTem := range v.Marktriggers {
			bloomutil.BusinessGeneralTriggerBloomFilter.Add([]byte(mtTem.Marktrigger))
		}
	}

}

func refreshGeneralTriggerBloom(ctx *gin.Context) {
	generalTriggerDao := dao.NewGeneralTriggerDao(ctx)
	datasTem, res := generalTriggerDao.Find(model.GeneralTriggerConfig{})
	if !res {
		return
	}

	for _, v := range datasTem {
		for _, mtTem := range v.Marktriggers {
			bloomutil.BusinessGeneralTriggerBloomFilter.Add([]byte(mtTem.Marktrigger))
		}
	}
}
