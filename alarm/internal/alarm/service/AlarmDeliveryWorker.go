package service

import (
	"fmt"
	"log"
	"runtime/debug"
	"sort"
	"time"

	alarmdao "alarm/internal/alarm/dao"
	alarmmodel "alarm/internal/alarm/model"
	businessmodel "alarm/internal/business/model"
	commonmodel "alarm/internal/common/model"
	usermodel "alarm/models/model"
	"alarm/util/gormutil"
	"alarm/util/opengauss"

	"github.com/gin-gonic/gin"
)

func ValidateAlarmReliabilityDatabases() error {
	if err := alarmdao.ValidateReliabilitySchema(opengauss.OpengaussGormDB); err != nil {
		return err
	}
	for _, dbName := range opengauss.ConfiguredRenterDBNames() {
		db := opengauss.GetRenterDB(dbName)
		if db == nil {
			return fmt.Errorf("initialize alarm renter database %s failed", dbName)
		}
		if err := alarmdao.ValidateReliabilitySchema(db); err != nil {
			return fmt.Errorf("validate alarm renter database %s: %w", dbName, err)
		}
	}
	return nil
}

// AutoMigrateAlarmDatabases 在消费者启动前创建本程序实际使用的业务表。
func AutoMigrateAlarmDatabases() error {
	scopes := []*gin.Context{{}}
	for _, name := range opengauss.ConfiguredRenterDBNames() {
		ctx := &gin.Context{}
		ctx.Set("dbName", name)
		scopes = append(scopes, ctx)
	}
	for _, ctx := range scopes {
		db := gormutil.GetGormDB(ctx, "")
		if db == nil {
			return fmt.Errorf("alarm database is nil")
		}
		if err := alarmdao.AutoMigrateReliabilitySchema(db); err != nil {
			return err
		}
		// 告警程序会直接查询设备、用户、组织和短信凭据，首次部署时一并自动建表。
		if err := db.AutoMigrate(
			&commonmodel.AlarmDataRecord{},
			&alarmmodel.SmsSecret{},
			&businessmodel.Equipment{},
			&usermodel.User{},
			&usermodel.Corp{},
		); err != nil {
			return err
		}
	}
	return nil
}

func ProcessAllAlarmDeliveryScopes() {
	for _, ctx := range alarmDeliveryScopeContexts() {
		ProcessAlarmDeliveries(ctx)
	}
}

func CleanupAlarmReliabilityRecords() {
	before := time.Now().AddDate(0, 0, -90)
	for _, ctx := range alarmDeliveryScopeContexts() {
		dao := alarmdao.NewDeliveryDAO(ctx)
		if err := dao.DeleteFinishedBefore(before); err != nil {
			log.Println("[ALARM] cleanup notification delivery fail:", err)
		}
		if err := dao.DeleteInboxBefore(before); err != nil {
			log.Println("[ALARM] cleanup event inbox fail:", err)
		}
	}
}

func RunAlarmDeliveryWorker() {
	runAlarmWorkerSafely("delivery", ProcessAllAlarmDeliveryScopes)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		runAlarmWorkerSafely("delivery", ProcessAllAlarmDeliveryScopes)
	}
}

func RunAlarmReliabilityCleanupWorker() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		runAlarmWorkerSafely("cleanup", CleanupAlarmReliabilityRecords)
	}
}

func runAlarmWorkerSafely(name string, work func()) {
	defer func() {
		if recovered := recover(); recovered != nil {
			log.Printf("[ALARM] %s worker panic, err=%v\n%s", name, recovered, debug.Stack())
		}
	}()
	work()
}

func alarmDeliveryScopeContexts() []*gin.Context {
	contexts := []*gin.Context{{}}
	nameSet := make(map[string]struct{})
	for _, name := range append(opengauss.ConfiguredRenterDBNames(), opengauss.RenterDBNames()...) {
		if name != "" {
			nameSet[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if opengauss.GetRenterDB(name) == nil {
			log.Printf("[ALARM] skip unavailable renter delivery database, dbName=%s", name)
			continue
		}
		ctx := &gin.Context{}
		ctx.Set("dbName", name)
		contexts = append(contexts, ctx)
	}
	return contexts
}
