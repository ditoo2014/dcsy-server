package battery

import (
	"deal/internal/batterylog"
	"deal/internal/tolink/model"
	"deal/util/opengauss"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type moduleSnapshot struct {
	Modules []model.GatewayModuleTtdc
}

var (
	batteryEnabled atomic.Bool
	configAccessMu sync.RWMutex

	moduleCacheMu           sync.RWMutex
	moduleCache             = make(map[string]moduleSnapshot)
	initializationMu        sync.Mutex
	initializationRetryOnce sync.Once
	cacheRefresherOnce      sync.Once
)

func Init() {
	batterylog.Printf("[BATTERY-CONFIG] enabled=%t debugLog=%t aiEnabled=%t opengauss=%s:%s/%s\n", viper.GetBool("battery.enabled"), viper.GetBool("battery.debugLog"), viper.GetBool("battery.ai.enabled"), viper.GetString("opengauss.host"), viper.GetString("opengauss.port"), viper.GetString("opengauss.dbname"))
	if !Configured() {
		batterylog.Println("[BATTERY] disabled by config")
		return
	}
	if err := initialize(); err != nil {
		batterylog.Printf("[BATTERY] initialization failed, retrying in background: %v\n", err)
		startInitializationRetry()
		return
	}
	batterylog.Printf("[BATTERY] module cache loaded, gateway count=%d\n", ModuleCacheSize())
}

func initialize() error {
	initializationMu.Lock()
	defer initializationMu.Unlock()
	if batteryEnabled.Load() {
		return nil
	}

	db := opengauss.GetOpenGaussDB()
	reconnected := db == nil || db.Ping() != nil
	if reconnected {
		if db != nil {
			_ = db.Close()
		}
		if err := opengauss.InitOpenGaussDB(); err != nil {
			return fmt.Errorf("initialize OpenGauss: %w", err)
		}
	}
	if db := opengauss.GetDefaultGormDB(); reconnected || db == nil {
		if db = opengauss.InitGormDB(); db == nil {
			return fmt.Errorf("initialize OpenGauss GORM")
		}
	}
	revisionID, revisionEnabled, err := currentConfigRevisionID(&gin.Context{})
	if err != nil {
		return err
	}
	if err := LoadModuleCache(&gin.Context{}); err != nil {
		return err
	}
	if err := LoadGatewayCache(&gin.Context{}); err != nil {
		return err
	}
	if err := LoadAddressGroupCache(); err != nil {
		return err
	}
	if err := LoadTemplateCache(&gin.Context{}); err != nil {
		return err
	}
	setConfigRevisionCursor(revisionID, revisionEnabled)
	if revisionEnabled {
		if err := PollConfigRevisions(&gin.Context{}); err != nil {
			return err
		}
	}

	batteryEnabled.Store(true)
	cacheRefresherOnce.Do(startConfigCacheRefresher)
	return nil
}

func Enabled() bool {
	return batteryEnabled.Load()
}

func Configured() bool {
	return viper.GetBool("battery.enabled")
}

func startInitializationRetry() {
	initializationRetryOnce.Do(func() {
		seconds := viper.GetInt("battery.initRetrySeconds")
		if seconds <= 0 {
			seconds = 30
		}
		go func() {
			ticker := time.NewTicker(time.Duration(seconds) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				if err := initialize(); err != nil {
					batterylog.Printf("[BATTERY] initialization retry failed: %v\n", err)
					continue
				}
				batterylog.Printf("[BATTERY] initialization recovered, gateway count=%d\n", ModuleCacheSize())
				return
			}
		}()
	})
}

func LoadModuleCache(ctx *gin.Context) error {
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return fmt.Errorf("opengauss gorm db is nil")
	}

	next := make(map[string]moduleSnapshot)
	totalModules := 0
	totalDataPoints := 0
	lastID := uint(0)
	hasBindings := db.Migrator().HasTable(&model.GatewayTtdcTemplateBinding{})
	for {
		var modules []model.GatewayModuleTtdc
		if err := db.Preload("TtdcConfig").Preload("ModuleDatas").Where("id > ?", lastID).Order("id asc").Limit(500).Find(&modules).Error; err != nil {
			return fmt.Errorf("query gateway_module_ttdc failed: %w", err)
		}
		if len(modules) == 0 {
			break
		}
		if hasBindings {
			if err := attachModuleBindings(db, modules); err != nil {
				return err
			}
		}
		for _, module := range modules {
			gatewayID := strings.TrimSpace(module.GatewayId)
			if gatewayID == "" {
				continue
			}
			snapshot := next[gatewayID]
			snapshot.Modules = append(snapshot.Modules, module)
			next[gatewayID] = snapshot
			totalModules++
			totalDataPoints += len(module.ModuleDatas)
			debugLogf("[BATTERY-CONFIG] gateway=%s moduleID=%d name=%q SOI=%q VER=%q ADR=%q CID1=%q CID2=%q templateID=%d profile=%q dataPoints=%d\n", gatewayID, module.ID, module.Name, module.TtdcConfig.ModuleSoi, module.TtdcConfig.ModuleVer, module.TtdcConfig.ModuleAdr, module.TtdcConfig.ModuleCid1, module.TtdcConfig.ModuleCid2, module.TemplateID, module.ParserProfile, len(module.ModuleDatas))
		}
		lastID = modules[len(modules)-1].ID
	}

	moduleCacheMu.Lock()
	moduleCache = next
	moduleCacheMu.Unlock()
	batterylog.Printf("[BATTERY-CONFIG] module cache loaded gateways=%d modules=%d dataPoints=%d\n", len(next), totalModules, totalDataPoints)
	return nil
}

func attachModuleBindings(db *gorm.DB, modules []model.GatewayModuleTtdc) error {
	moduleIDs := make([]uint, 0, len(modules))
	for _, module := range modules {
		moduleIDs = append(moduleIDs, module.ID)
	}
	if len(moduleIDs) == 0 {
		return nil
	}
	var bindings []model.GatewayTtdcTemplateBinding
	if err := db.Where("module_id IN ?", moduleIDs).Find(&bindings).Error; err != nil {
		return fmt.Errorf("query TTDC template bindings failed: %w", err)
	}
	bindingByModule := make(map[uint]model.GatewayTtdcTemplateBinding, len(bindings))
	for _, binding := range bindings {
		bindingByModule[binding.ModuleID] = binding
	}
	for i := range modules {
		if binding, ok := bindingByModule[modules[i].ID]; ok {
			modules[i].TemplateID = binding.TemplateID
			modules[i].TemplateVersion = binding.TemplateVersion
			modules[i].ProtocolCode = binding.ProtocolCode
			modules[i].ParserProfile = binding.ParserProfile
		}
	}
	return nil
}

func RefreshGatewayModules(_ interface{}, gatewayID string) error {
	db := opengauss.GetDefaultGormDB()
	var modules []model.GatewayModuleTtdc
	if err := db.Preload("TtdcConfig").Preload("ModuleDatas").Where("gateway_id = ?", gatewayID).Order("id asc").Find(&modules).Error; err != nil {
		return err
	}
	if db.Migrator().HasTable(&model.GatewayTtdcTemplateBinding{}) {
		if err := attachModuleBindings(db, modules); err != nil {
			return err
		}
	}
	moduleCacheMu.Lock()
	if len(modules) == 0 {
		delete(moduleCache, gatewayID)
	} else {
		moduleCache[gatewayID] = moduleSnapshot{Modules: modules}
	}
	moduleCacheMu.Unlock()
	return nil
}

func GetModulesByGatewayID(gatewayID string) ([]model.GatewayModuleTtdc, bool) {
	key := strings.TrimSpace(gatewayID)
	if key == "" {
		return nil, false
	}
	moduleCacheMu.RLock()
	snapshot, ok := moduleCache[key]
	moduleCacheMu.RUnlock()
	if !ok {
		return nil, false
	}
	modules := make([]model.GatewayModuleTtdc, len(snapshot.Modules))
	copy(modules, snapshot.Modules)
	return modules, true
}

func ModuleCacheSize() int {
	moduleCacheMu.RLock()
	defer moduleCacheMu.RUnlock()
	return len(moduleCache)
}

func upsertModuleCache(module model.GatewayModuleTtdc) {
	gatewayID := strings.TrimSpace(module.GatewayId)
	if gatewayID == "" {
		return
	}
	moduleCacheMu.Lock()
	snapshot := moduleCache[gatewayID]
	replaced := false
	for i := range snapshot.Modules {
		if snapshot.Modules[i].ID == module.ID {
			snapshot.Modules[i] = module
			replaced = true
			break
		}
	}
	if !replaced {
		snapshot.Modules = append(snapshot.Modules, module)
	}
	moduleCache[gatewayID] = snapshot
	moduleCacheMu.Unlock()
}

func startConfigCacheRefresher() {
	pollInterval := viper.GetDuration("battery.revisionPollInterval")
	if pollInterval <= 0 {
		seconds := viper.GetInt("battery.revisionPollSeconds")
		if seconds <= 0 {
			seconds = 5
		}
		pollInterval = time.Duration(seconds) * time.Second
	}
	fullRefreshInterval := viper.GetDuration("battery.fullCacheRefreshInterval")
	if fullRefreshInterval <= 0 {
		minutes := viper.GetInt("battery.fullCacheRefreshMinutes")
		if minutes <= 0 {
			minutes = 60
		}
		fullRefreshInterval = time.Duration(minutes) * time.Minute
	}
	go func() {
		pollTicker := time.NewTicker(pollInterval)
		fullTicker := time.NewTicker(fullRefreshInterval)
		defer pollTicker.Stop()
		defer fullTicker.Stop()
		for {
			select {
			case <-pollTicker.C:
				if err := PollConfigRevisions(&gin.Context{}); err != nil {
					batterylog.Printf("[BATTERY-CONFIG] revision poll failed: %v\n", err)
				}
			case <-fullTicker.C:
				if err := RefreshAllConfigCaches(&gin.Context{}); err != nil {
					batterylog.Printf("[BATTERY-CONFIG] full cache calibration failed: %v\n", err)
				}
			}
		}
	}()
}
