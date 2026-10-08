package schedule

import (
	"dicloud/internal/business/dao"
	"dicloud/internal/business/model"
	"dicloud/internal/business/service"
	commonservice "dicloud/internal/common/service"
	"dicloud/pkg/cache"
	normalutil "dicloud/util/normal"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// 定时统计平台设备数量
func StatisticPlatEquipmentSchedule() {

	/* 默认平台数据 - multi renter(多租户) */
	ctx := &gin.Context{}
	statisticPlatEquipment(ctx, "")
	statisticPlatEquipmentSeq(ctx, "")
	statisticPlatEquipmentCurrent(ctx, "")

	/* multi renter(多租户) */
	// 处理所有租户的定时统计
	dbNameDatas := commonservice.LoadingAllPlat()
	if len(dbNameDatas) > 0 {
		for _, dbTem := range dbNameDatas {
			ctxTem := &gin.Context{}
			ctxTem.Set("dbName", dbTem)

			statisticPlatEquipment(ctxTem, dbTem)
			statisticPlatEquipmentSeq(ctxTem, dbTem)
			statisticPlatEquipmentCurrent(ctxTem, dbTem)
		}
	}
}

// 每小时, 统计设备数量, 设备健康度
func statisticPlatEquipment(ctx *gin.Context, dbTem string) {

	// log.Println("[StatisticPlatEquipment] renter " + dbTem + " is start.")
	// 每小时执行一次
	dateStr := time.Now().Format("2006-01-02-15")
	keyTem := "statistic:equipment:schedule:" + dbTem + ":" + dateStr
	res, _ := cache.GetRedisClient(ctx).SetNX(keyTem, "1", 1*time.Hour).Result()
	if !res {
		return
	}

	// 每小时进入一次 - 统计设备数量
	log.Println("[BUSINESS] StatisticPlatEquipmentSchedule in")
	formatStatisticData(ctx)
}

// 每天一次, 回算最近7个完整自然日的基础数据。
func statisticPlatEquipmentSeq(ctx *gin.Context, dbTem string) {

	now := time.Now()
	if !shouldRunDailyStatistic(now) {
		// 每天 2:30 以后首次获得任务锁时执行，避免服务在 2:30 短暂不可用后整天漏算。
		return
	}

	dateStr := now.Format("2006-01-02")
	keyTem := "statistic:equipment:seqdata:schedule:" + dbTem + ":" + dateStr
	res, _ := cache.GetRedisClient(ctx).SetNX(keyTem, "1", 26*time.Hour).Result()
	if !res {
		return
	}

	// 统计最近7个完整自然日，今天的数据由小时任务持续补齐。
	year, month, day := now.Date()
	currentTime := time.Date(year, month, day, 0, 0, 0, 0, now.Location())
	var endTime = currentTime.Unix() - 1
	var startTime = endTime - 7*24*60*60
	// 回算最近7天的累计值和健康度
	log.Println("[BUSINESS] StatisticPlatEquipmentSeq in")

	equipmentDao := dao.NewEquipmentDAO(ctx)
	offset, limit := normalutil.GetPageOffsetLimit(1, 9999)
	equipmentList, _, _ := equipmentDao.FindEquipmentAll(model.Equipment{}, offset, limit)

	if !service.StatisticDurationData(ctx, equipmentList, startTime, endTime) {
		cache.GetRedisClient(ctx).Del(keyTem)
		log.Printf("[BUSINESS] StatisticPlatEquipmentSeq failed, daily lock released renter=%s", dbTem)
	}
}

// 每小时刷新当天的累计充放电量、循环次数和健康度。
func statisticPlatEquipmentCurrent(ctx *gin.Context, dbTem string) {
	now := time.Now()
	dateStr := now.Format("2006-01-02-15")
	keyTem := "statistic:equipment:current:schedule:" + dbTem + ":" + dateStr
	res, _ := cache.GetRedisClient(ctx).SetNX(keyTem, "1", 2*time.Hour).Result()
	if !res {
		return
	}

	equipmentDao := dao.NewEquipmentDAO(ctx)
	offset, limit := normalutil.GetPageOffsetLimit(1, 9999)
	// 累计充放电量按所有已登记电池统计，与每日历史回算保持相同范围。
	equipmentList, _, findRes := equipmentDao.FindEquipmentAll(model.Equipment{}, offset, limit)
	if !findRes || len(equipmentList) == 0 {
		return
	}

	go func() {
		if !service.StatisticCurrentDayData(ctx, equipmentList, now.Unix()) {
			cache.GetRedisClient(ctx).Del(keyTem)
			log.Printf("[BUSINESS] StatisticPlatEquipmentCurrent failed, hourly lock released renter=%s", dbTem)
		}
	}()
}

func shouldRunDailyStatistic(now time.Time) bool {
	hour, minute, _ := now.Clock()
	return hour > 2 || (hour == 2 && minute >= 30)
}

func formatStatisticData(ctx *gin.Context) {
	equipmentDao := dao.NewEquipmentDAO(ctx)

	equipmentTem := model.Equipment{
		PositionStatus: 3,
	}
	pageSize := 9999
	offset, limit := normalutil.GetPageOffsetLimit(1, pageSize)
	equipmentList, count, res := equipmentDao.FindEquipmentAll(equipmentTem, offset, limit)
	if res {
		// 注: 不一样的线程, RocketMQ的FIFO队列, 相同的ShardingKey可能会分到不同队列
		// 注: 提前的方法细分, 会决定不同的<统计代码>分流, 即时相同的EquipmentId相遇, Code也不同; 互不影响
		// 注: 如果Code相同, 尽量使用同一个线程, 使用ShardingKey分流, 保证FIFO
		go service.SendEquipmentNumberRocketMsgToStatistic(ctx, equipmentList)
		go service.SendEquipmentHealthRocketMsgToStatistic(ctx, equipmentList)
	} else {
		return
	}

	// 页码计算
	rangeNum := count / int64(pageSize)
	if (count % int64(pageSize)) != 0 {
		rangeNum = rangeNum + 1
	}
	// 分页个数
	for i := 1; i < int(rangeNum); i++ {
		// i=1: 从第2页查, 直到足够页数
		offset, limit := normalutil.GetPageOffsetLimit(i+1, pageSize)
		equipmentList, _, res := equipmentDao.FindEquipmentAll(equipmentTem, offset, limit)
		if res {
			go service.SendEquipmentNumberRocketMsgToStatistic(ctx, equipmentList)
			go service.SendEquipmentHealthRocketMsgToStatistic(ctx, equipmentList)
		} else {
			return
		}
	}
}
