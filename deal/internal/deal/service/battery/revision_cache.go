package battery

import (
	"deal/internal/batterylog"
	"deal/internal/tolink/model"
	"deal/util/opengauss"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

var configRevisionState = struct {
	sync.Mutex
	cursor  uint64
	enabled bool
}{}

func currentConfigRevisionID(ctx *gin.Context) (uint64, bool, error) {
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return 0, false, errors.New("opengauss gorm db is nil")
	}
	if !db.Migrator().HasTable(&model.TtdcConfigRevision{}) {
		return 0, false, nil
	}
	var revision model.TtdcConfigRevision
	err := db.Order("id desc").Limit(1).First(&revision).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, true, nil
	}
	if err != nil {
		return 0, true, fmt.Errorf("query TTDC config revision cursor: %w", err)
	}
	return revision.ID, true, nil
}

func setConfigRevisionCursor(cursor uint64, enabled bool) {
	configRevisionState.Lock()
	configRevisionState.cursor = cursor
	configRevisionState.enabled = enabled
	configRevisionState.Unlock()
}

func PollConfigRevisions(ctx *gin.Context) error {
	configAccessMu.Lock()
	defer configAccessMu.Unlock()
	return pollConfigRevisionsLocked(ctx)
}

func pollConfigRevisionsLocked(ctx *gin.Context) error {
	configRevisionState.Lock()
	defer configRevisionState.Unlock()

	if !configRevisionState.enabled {
		cursor, enabled, err := currentConfigRevisionID(ctx)
		if err != nil || !enabled {
			return err
		}
		if err := refreshAllConfigCachesLocked(ctx); err != nil {
			return err
		}
		configRevisionState.cursor = cursor
		configRevisionState.enabled = true
		batterylog.Printf("[BATTERY-CONFIG] revision polling enabled cursor=%d\n", cursor)
		return nil
	}

	db := opengauss.GetDefaultGormDB()
	for {
		var revisions []model.TtdcConfigRevision
		if err := db.Where("id > ?", configRevisionState.cursor).Order("id asc").Limit(2000).Find(&revisions).Error; err != nil {
			return fmt.Errorf("poll TTDC config revisions: %w", err)
		}
		if len(revisions) == 0 {
			return nil
		}

		gatewayKeys := map[string]bool{}
		moduleKeys := map[string]bool{}
		addressKeys := map[string]bool{}
		templateChanged := false
		for _, revision := range revisions {
			switch revision.ConfigType {
			case "gateway":
				gatewayKeys[revision.TargetKey] = true
			case "module":
				moduleKeys[revision.TargetKey] = true
			case "template":
				templateChanged = true
			case "template_group":
				templateChanged = true
			case "address_group":
				addressKeys[revision.TargetKey] = true
			}
		}
		for _, gatewayID := range sortedRevisionKeys(gatewayKeys) {
			if err := RefreshGateway(ctx, gatewayID); err != nil {
				return fmt.Errorf("refresh gateway %s: %w", gatewayID, err)
			}
		}
		for _, gatewayID := range sortedRevisionKeys(moduleKeys) {
			if err := RefreshGatewayModules(ctx, gatewayID); err != nil {
				return fmt.Errorf("refresh gateway modules %s: %w", gatewayID, err)
			}
		}
		if templateChanged {
			if err := LoadTemplateCache(ctx); err != nil {
				return fmt.Errorf("refresh TTDC templates: %w", err)
			}
		}
		for _, key := range sortedRevisionKeys(addressKeys) {
			if err := RefreshAddressBinding(key); err != nil {
				return fmt.Errorf("refresh TTDC ADR binding %s: %w", key, err)
			}
		}

		configRevisionState.cursor = revisions[len(revisions)-1].ID
		batterylog.Printf("[BATTERY-CONFIG] revisions applied through id=%d gateways=%d modules=%d addresses=%d templatesChanged=%t\n", configRevisionState.cursor, len(gatewayKeys), len(moduleKeys), len(addressKeys), templateChanged)
		if len(revisions) < 2000 {
			return nil
		}
	}
}

func sortedRevisionKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		if key != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func RefreshAllConfigCaches(ctx *gin.Context) error {
	configAccessMu.Lock()
	defer configAccessMu.Unlock()
	return refreshAllConfigCachesLocked(ctx)
}

type configCacheBackup struct {
	gateways             map[string]GatewayRef
	modules              map[string]moduleSnapshot
	addresses            map[string]addressGroupSnapshot
	templates            map[string]TemplateSnapshot
	groups               map[string]TemplateGroupSnapshot
	templateNegative     map[string]time.Time
	legacyTemplateMode   bool
	templateRuntimeReady bool
}

func captureConfigCacheBackup() configCacheBackup {
	gatewayCacheState.RLock()
	gateways := gatewayCacheState.values
	gatewayCacheState.RUnlock()
	moduleCacheMu.RLock()
	modules := moduleCache
	moduleCacheMu.RUnlock()
	addressGroupCacheState.RLock()
	addresses := addressGroupCacheState.values
	addressGroupCacheState.RUnlock()
	templateCacheMu.RLock()
	templates, groups := templateCache, groupCache
	templateNegative := templateNegativeCache
	legacy, ready := legacyTemplateMode, templateRuntimeReady
	templateCacheMu.RUnlock()
	return configCacheBackup{gateways: gateways, modules: modules, addresses: addresses, templates: templates, groups: groups, templateNegative: templateNegative, legacyTemplateMode: legacy, templateRuntimeReady: ready}
}

func restoreConfigCacheBackup(backup configCacheBackup) {
	gatewayCacheState.Lock()
	gatewayCacheState.values = backup.gateways
	gatewayCacheState.Unlock()
	moduleCacheMu.Lock()
	moduleCache = backup.modules
	moduleCacheMu.Unlock()
	addressGroupCacheState.Lock()
	addressGroupCacheState.values = backup.addresses
	addressGroupCacheState.Unlock()
	templateCacheMu.Lock()
	templateCache, groupCache = backup.templates, backup.groups
	templateNegativeCache = backup.templateNegative
	legacyTemplateMode, templateRuntimeReady = backup.legacyTemplateMode, backup.templateRuntimeReady
	templateCacheMu.Unlock()
}

func refreshAllConfigCachesLocked(ctx *gin.Context) error {
	backup := captureConfigCacheBackup()
	if err := LoadGatewayCache(ctx); err != nil {
		return err
	}
	if err := LoadModuleCache(ctx); err != nil {
		restoreConfigCacheBackup(backup)
		return err
	}
	if err := LoadAddressGroupCache(); err != nil {
		restoreConfigCacheBackup(backup)
		return err
	}
	if err := LoadTemplateCache(ctx); err != nil {
		restoreConfigCacheBackup(backup)
		return err
	}
	batterylog.Printf("[BATTERY-CONFIG] full cache calibration complete gateways=%d\n", ModuleCacheSize())
	return nil
}
