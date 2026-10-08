package battery

import (
	"deal/internal/batterylog"
	"deal/internal/tolink/model"
	"deal/util/opengauss"
	"errors"
	"fmt"
	"hash/fnv"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"golang.org/x/sync/singleflight"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var gatewayIDPattern = regexp.MustCompile(`^[0-9]{15}$`)

type GatewayRef struct {
	GatewayID, Name, Model string
	RefreshAfter           time.Time
}

type PreparedBatteryChannels struct {
	GatewayID       string
	TopicSegment    string
	RawCID2         string
	TemplateGroupID uint
	CurrentModule   model.GatewayModuleTtdc
	SocModule       model.GatewayModuleTtdc
	ModulesByRole   map[string]model.GatewayModuleTtdc
}

// PreparedModule remains as a compatibility type for the legacy parser tests.
type PreparedModule struct {
	GatewayID, TopicSegment, RawCID2, ProtocolCode, ParserProfile string
	TemplateID                                                    uint
	Module                                                        model.GatewayModuleTtdc
}

type addressGroupSnapshot struct {
	GatewayID, ADR, GroupCode, TopicSegment, SOI, VER, CID1, Status string
	TemplateGroupID                                                 uint
}

var (
	gatewayEnsureGroup   singleflight.Group
	gatewayRefreshGroup  singleflight.Group
	addressEnsureGroup   singleflight.Group
	gatewayCreateLimiter = make(chan struct{}, 32)
	moduleCreateLimiter  = make(chan struct{}, 16)
	gatewayCacheState    = struct {
		sync.RWMutex
		values map[string]GatewayRef
	}{values: map[string]GatewayRef{}}
	addressGroupCacheState = struct {
		sync.RWMutex
		values map[string]addressGroupSnapshot
	}{values: map[string]addressGroupSnapshot{}}
)

func ValidateGatewayID(value string) bool {
	return gatewayIDPattern.MatchString(strings.TrimSpace(value))
}
func addressKey(gatewayID, adr string) string {
	return strings.TrimSpace(gatewayID) + ":" + strings.ToUpper(strings.TrimSpace(adr))
}

func LoadGatewayCache(_ interface{}) error {
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return errors.New("opengauss gorm db is nil")
	}
	next := map[string]GatewayRef{}
	lastID := uint(0)
	for {
		var rows []model.Gateway
		if err := db.Select("id", "gateway_id", "name", "model").Where("id > ?", lastID).Order("id asc").Limit(2000).Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) == 0 {
			break
		}
		for _, row := range rows {
			id := strings.TrimSpace(row.GatewayId)
			if id != "" {
				next[id] = newGatewayRef(id, row.Name, row.Model)
			}
		}
		lastID = rows[len(rows)-1].ID
	}
	gatewayCacheState.Lock()
	gatewayCacheState.values = next
	gatewayCacheState.Unlock()
	batterylog.Printf("[BATTERY-CONFIG] gateway cache loaded gateways=%d\n", len(next))
	return nil
}

func LoadAddressGroupCache() error {
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return errors.New("opengauss gorm db is nil")
	}
	if !db.Migrator().HasTable(&model.GatewayTtdcAddressGroupBinding{}) {
		addressGroupCacheState.Lock()
		addressGroupCacheState.values = map[string]addressGroupSnapshot{}
		addressGroupCacheState.Unlock()
		return nil
	}
	var rows []model.GatewayTtdcAddressGroupBinding
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	next := make(map[string]addressGroupSnapshot, len(rows))
	for _, row := range rows {
		next[addressKey(row.GatewayID, row.ModuleADR)] = addressSnapshot(row)
	}
	addressGroupCacheState.Lock()
	addressGroupCacheState.values = next
	addressGroupCacheState.Unlock()
	batterylog.Printf("[BATTERY-CONFIG] ADR group cache loaded bindings=%d\n", len(next))
	return nil
}

func addressSnapshot(row model.GatewayTtdcAddressGroupBinding) addressGroupSnapshot {
	return addressGroupSnapshot{GatewayID: row.GatewayID, ADR: row.ModuleADR, TemplateGroupID: row.TemplateGroupID, GroupCode: row.GroupCode, TopicSegment: row.TopicSegment, SOI: row.ModuleSOI, VER: row.ModuleVER, CID1: row.ModuleCID1, Status: row.BindingStatus}
}
func cachedGateway(id string) (GatewayRef, bool) {
	gatewayCacheState.RLock()
	v, ok := gatewayCacheState.values[id]
	gatewayCacheState.RUnlock()
	return v, ok
}
func cacheGateway(v GatewayRef) {
	v.RefreshAfter = gatewayRefreshAfter(v.GatewayID)
	gatewayCacheState.Lock()
	gatewayCacheState.values[v.GatewayID] = v
	gatewayCacheState.Unlock()
}
func newGatewayRef(id, name, gatewayModel string) GatewayRef {
	return GatewayRef{GatewayID: id, Name: strings.TrimSpace(name), Model: strings.TrimSpace(gatewayModel), RefreshAfter: gatewayRefreshAfter(id)}
}
func gatewayRefreshAfter(id string) time.Time {
	h := fnv.New32a()
	_, _ = h.Write([]byte(id))
	return time.Now().Add(30*time.Minute + time.Duration(h.Sum32()%300)*time.Second)
}

func RefreshGateway(_ interface{}, gatewayID string) error {
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return errors.New("opengauss gorm db is nil")
	}
	var gateway model.Gateway
	err := db.Select("gateway_id", "name", "model").Where("gateway_id = ?", gatewayID).First(&gateway).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		gatewayCacheState.Lock()
		delete(gatewayCacheState.values, gatewayID)
		gatewayCacheState.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	cacheGateway(newGatewayRef(gatewayID, gateway.Name, gateway.Model))
	return nil
}

func refreshGatewayInBackground(gatewayID string) {
	go func() {
		configAccessMu.RLock()
		defer configAccessMu.RUnlock()
		_, err, _ := gatewayRefreshGroup.Do(gatewayID, func() (interface{}, error) {
			return nil, RefreshGateway(nil, gatewayID)
		})
		if err == nil {
			return
		}
		// Back off after a failed refresh so a busy gateway cannot query the
		// database again for every packet while the database is unavailable.
		gatewayCacheState.Lock()
		if cached, ok := gatewayCacheState.values[gatewayID]; ok {
			cached.RefreshAfter = time.Now().Add(time.Minute)
			gatewayCacheState.values[gatewayID] = cached
		}
		gatewayCacheState.Unlock()
		batterylog.Printf("[BATTERY-CONFIG] gateway soft-TTL refresh failed gatewayID=%s err=%v\n", gatewayID, err)
	}()
}

func EnsureGateway(_ *gin.Context, gatewayID string) (GatewayRef, error) {
	gatewayID = strings.TrimSpace(gatewayID)
	if !ValidateGatewayID(gatewayID) {
		return GatewayRef{}, permanentf("gateway ID must be exactly 15 decimal digits: %s", gatewayID)
	}
	if cached, ok := cachedGateway(gatewayID); ok {
		if cached.Model != model.TTDC {
			return GatewayRef{}, permanentf("gateway %s model conflict: %s", gatewayID, cached.Model)
		}
		if !cached.RefreshAfter.IsZero() && time.Now().After(cached.RefreshAfter) {
			refreshGatewayInBackground(gatewayID)
		}
		return cached, nil
	}
	value, err, _ := gatewayEnsureGroup.Do(gatewayID, func() (interface{}, error) {
		if cached, ok := cachedGateway(gatewayID); ok {
			return cached, nil
		}
		db := opengauss.GetDefaultGormDB()
		if db == nil {
			return GatewayRef{}, errors.New("opengauss gorm db is nil")
		}
		var gateway model.Gateway
		dbErr := db.Where("gateway_id = ?", gatewayID).First(&gateway).Error
		if dbErr == nil {
			if gateway.Model != model.TTDC {
				return GatewayRef{}, permanentf("gateway %s model conflict: %s", gatewayID, gateway.Model)
			}
			ref := newGatewayRef(gatewayID, gateway.Name, gateway.Model)
			cacheGateway(ref)
			return ref, nil
		}
		if !errors.Is(dbErr, gorm.ErrRecordNotFound) {
			return GatewayRef{}, dbErr
		}
		if !viper.GetBool("battery.gatewayAutoCreateEnabled") {
			return GatewayRef{}, permanentf("gateway %s is missing and automatic creation is disabled", gatewayID)
		}
		gatewayCreateLimiter <- struct{}{}
		defer func() { <-gatewayCreateLimiter }()
		gateway = model.Gateway{Name: gatewayID, GatewayId: gatewayID, Model: model.TTDC, Card: "-", CreateUserId: 0, CorpId: 0}
		dbErr = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(&gateway).Error; err != nil {
				return err
			}
			return tx.Create(&model.TtdcConfigRevision{ConfigType: "gateway", TargetKey: gatewayID, Operation: "create"}).Error
		})
		if dbErr != nil {
			if reload := db.Where("gateway_id = ?", gatewayID).First(&gateway).Error; reload != nil {
				return GatewayRef{}, dbErr
			}
			if gateway.Model != model.TTDC {
				return GatewayRef{}, permanentf("gateway %s model conflict: %s", gatewayID, gateway.Model)
			}
		} else {
			batterylog.Printf("[BATTERY-AUTOCONFIG] gateway created gatewayID=%s model=%s\n", gatewayID, model.TTDC)
		}
		ref := newGatewayRef(gatewayID, gateway.Name, gateway.Model)
		cacheGateway(ref)
		return ref, nil
	})
	if err != nil {
		return GatewayRef{}, err
	}
	return value.(GatewayRef), nil
}

func EnsureModules(_ *gin.Context, gatewayID, topicSegment, requestFrame, rawCID2 string, _ int, group TemplateGroupSnapshot) (PreparedBatteryChannels, error) {
	if len(requestFrame) < 9 {
		return PreparedBatteryChannels{}, permanentf("request frame is too short")
	}
	adr, validADR := model.NormalizeTtdcADR(requestFrame[3:5])
	if !validADR {
		return PreparedBatteryChannels{}, permanentf("invalid TTDC ADR: %s", adr)
	}
	if group.ID == 0 || group.TopicSegment != topicSegment || !group.RuntimeMatchEnabled {
		return PreparedBatteryChannels{}, permanentf("template group does not match topic segment")
	}
	currentTemplate, ok := group.Templates[strings.ToUpper(rawCID2)]
	if !ok || rawCID2 == model.TtdcRoleCID42SOC {
		return PreparedBatteryChannels{}, permanentf("template group does not support CID2 %s", rawCID2)
	}
	if err := validateTemplateStructure(currentTemplate); err != nil {
		return PreparedBatteryChannels{}, permanentf("template %s: %v", currentTemplate.TemplateCode, err)
	}
	identity := addressKey(gatewayID, adr)
	if prepared, ok, err := preparedFromCaches(gatewayID, topicSegment, requestFrame, rawCID2, group); ok || err != nil {
		return prepared, err
	}
	value, err, _ := addressEnsureGroup.Do(identity, func() (interface{}, error) {
		if prepared, ok, err := preparedFromCaches(gatewayID, topicSegment, requestFrame, rawCID2, group); ok || err != nil {
			return prepared, err
		}
		db := opengauss.GetDefaultGormDB()
		if db == nil {
			return PreparedBatteryChannels{}, errors.New("opengauss gorm db is nil")
		}
		if !db.Migrator().HasTable(&model.GatewayTtdcAddressGroupBinding{}) {
			return PreparedBatteryChannels{}, permanentf("TTDC template-group schema is not initialized")
		}
		var address model.GatewayTtdcAddressGroupBinding
		queryErr := db.Where("gateway_id = ? AND module_adr = ?", gatewayID, adr).First(&address).Error
		if queryErr == nil {
			snapshot := addressSnapshot(address)
			addressGroupCacheState.Lock()
			addressGroupCacheState.values[identity] = snapshot
			addressGroupCacheState.Unlock()
			// An inactive row can be left by an older release after the gateway was
			// deleted. Its ownership is checked and replaced under the transaction
			// lock below after confirming that no channel binding remains.
			if address.BindingStatus == "active" {
				if err := validateAddressOwnership(snapshot, group, requestFrame); err != nil {
					return PreparedBatteryChannels{}, err
				}
			}
		}
		if queryErr != nil && !errors.Is(queryErr, gorm.ErrRecordNotFound) {
			return PreparedBatteryChannels{}, queryErr
		}
		if queryErr == nil && address.BindingStatus == "active" {
			if err := RefreshGatewayModules(nil, gatewayID); err != nil {
				return PreparedBatteryChannels{}, err
			}
			if prepared, ok, err := preparedFromCaches(gatewayID, topicSegment, requestFrame, rawCID2, group); ok || err != nil {
				return prepared, err
			}
		}
		if !viper.GetBool("battery.channelAutoCreateEnabled") || !group.AutoCreateEnabled {
			return PreparedBatteryChannels{}, permanentf("ADR %s channels are incomplete and automatic creation is disabled", adr)
		}
		moduleCreateLimiter <- struct{}{}
		defer func() { <-moduleCreateLimiter }()
		if err := createMissingGroupChannels(db, gatewayID, requestFrame, group); err != nil {
			return PreparedBatteryChannels{}, err
		}
		if err := RefreshGatewayModules(nil, gatewayID); err != nil {
			return PreparedBatteryChannels{}, err
		}
		if err := refreshAddressBinding(gatewayID, adr); err != nil {
			return PreparedBatteryChannels{}, err
		}
		prepared, ok, prepareErr := preparedFromCaches(gatewayID, topicSegment, requestFrame, rawCID2, group)
		if prepareErr != nil {
			return PreparedBatteryChannels{}, prepareErr
		}
		if !ok {
			return PreparedBatteryChannels{}, errors.New("created TTDC channels are not visible in cache")
		}
		return prepared, nil
	})
	if err != nil {
		return PreparedBatteryChannels{}, err
	}
	return value.(PreparedBatteryChannels), nil
}

func validateTemplateStructure(item TemplateSnapshot) error {
	role := strings.ToUpper(strings.TrimSpace(item.CID2Type))
	switch role {
	case model.TtdcRoleCID42SOC:
		if !model.TtdcAlgorithmProfileCompatible(item.ParserProfile) {
			return errors.New("42_SOC template must use a derived algorithm profile")
		}
		seen := make(map[string]bool, len(item.Points))
		keys := make([]string, 0, len(item.Points))
		for _, point := range item.Points {
			key, expectedType, supported := model.TtdcAlgorithmResultKey(point.BaseDataID, point.AlgorithmKey)
			if seen[key] {
				return fmt.Errorf("algorithm result ID %s is duplicated", key)
			}
			if !supported || expectedType != point.DataType {
				return fmt.Errorf("algorithm result ID %s is unsupported or has the wrong data type", point.BaseDataID)
			}
			seen[key] = true
			keys = append(keys, key)
			if point.Address != nil || point.ByteCount != nil || point.ConvertType != nil || point.BitIndex != nil {
				return fmt.Errorf("algorithm point %s contains raw transport fields", point.BaseDataID)
			}
		}
		_, err := model.TtdcCID42SOCPointLimit(keys)
		return err
	case model.TtdcRoleCID42:
		if !model.TtdcProfileCompatibleWithRole(role, item.ParserProfile) {
			return errors.New("CID42 template must use a raw-fields profile")
		}
	case model.TtdcRoleCID44:
		if !model.TtdcProfileCompatibleWithRole(role, item.ParserProfile) {
			return errors.New("CID44 template must use a raw-fields profile")
		}
	default:
		return fmt.Errorf("unsupported template role %s", item.CID2Type)
	}

	for _, point := range item.Points {
		if point.Address == nil || point.ByteCount == nil || point.ConvertType == nil {
			return fmt.Errorf("raw point %s is missing transport fields", point.BaseDataID)
		}
		if point.DataType != 1 {
			return fmt.Errorf("raw point %s must use numeric data type", point.BaseDataID)
		}
		if *point.Address < 0 || *point.Address%2 != 0 || *point.ByteCount <= 0 || *point.ConvertType < 1 || *point.ConvertType > 3 {
			return fmt.Errorf("raw point %s has invalid transport fields", point.BaseDataID)
		}
		if role == model.TtdcRoleCID42 && point.BitIndex != nil {
			return fmt.Errorf("CID42 point %s cannot have a bit index", point.BaseDataID)
		}
		if *point.ConvertType == 2 && point.BitIndex == nil {
			return fmt.Errorf("binary point %s is missing a bit index", point.BaseDataID)
		}
		if *point.ConvertType != 2 && point.BitIndex != nil {
			return fmt.Errorf("decimal point %s cannot have a bit index", point.BaseDataID)
		}
		if point.BitIndex != nil && (*point.BitIndex < 0 || *point.BitIndex >= *point.ByteCount*8) {
			return fmt.Errorf("point %s bit index exceeds byte range", point.BaseDataID)
		}
		if point.AlgorithmKey != nil {
			return fmt.Errorf("raw point %s cannot have an algorithm key", point.BaseDataID)
		}
	}
	if role == model.TtdcRoleCID42 {
		baseDataIDs := make([]string, 0, len(item.Points))
		for _, point := range item.Points {
			baseDataIDs = append(baseDataIDs, point.BaseDataID)
		}
		if _, _, err := model.TtdcCID42PointLimits(baseDataIDs); err != nil {
			return err
		}
	}
	return nil
}

func validateTemplateGroupStructure(group TemplateGroupSnapshot) error {
	cid42CellLimit, cid42SOCCellLimit := 0, 0
	for _, role := range []string{model.TtdcRoleCID42, model.TtdcRoleCID42SOC, model.TtdcRoleCID44} {
		item, ok := group.Templates[role]
		if !ok {
			return fmt.Errorf("template group is missing role %s", role)
		}
		if err := validateTemplateStructure(item); err != nil {
			return fmt.Errorf("template %s: %v", item.TemplateCode, err)
		}
		switch role {
		case model.TtdcRoleCID42:
			baseDataIDs := make([]string, 0, len(item.Points))
			for _, point := range item.Points {
				baseDataIDs = append(baseDataIDs, point.BaseDataID)
			}
			var err error
			cid42CellLimit, _, err = model.TtdcCID42PointLimits(baseDataIDs)
			if err != nil {
				return fmt.Errorf("template %s: %v", item.TemplateCode, err)
			}
		case model.TtdcRoleCID42SOC:
			keys := make([]string, 0, len(item.Points))
			for _, point := range item.Points {
				key, _, _ := model.TtdcAlgorithmResultKey(point.BaseDataID, point.AlgorithmKey)
				keys = append(keys, key)
			}
			var err error
			cid42SOCCellLimit, err = model.TtdcCID42SOCPointLimit(keys)
			if err != nil {
				return fmt.Errorf("template %s: %v", item.TemplateCode, err)
			}
		}
	}
	if cid42CellLimit != cid42SOCCellLimit {
		return fmt.Errorf("CID42 has %d cell-voltage points but 42_SOC has %d per-cell SOC points", cid42CellLimit, cid42SOCCellLimit)
	}
	return nil
}

func preparedFromCaches(gatewayID, topicSegment, request, rawCID2 string, group TemplateGroupSnapshot) (PreparedBatteryChannels, bool, error) {
	key := addressKey(gatewayID, request[3:5])
	addressGroupCacheState.RLock()
	address, hasAddress := addressGroupCacheState.values[key]
	addressGroupCacheState.RUnlock()
	if !hasAddress {
		return PreparedBatteryChannels{}, false, nil
	}
	if address.Status != "active" {
		return PreparedBatteryChannels{}, false, nil
	}
	if err := validateAddressOwnership(address, group, request); err != nil {
		return PreparedBatteryChannels{}, false, err
	}
	modules, ok := GetModulesByGatewayID(gatewayID)
	if !ok {
		return PreparedBatteryChannels{}, false, nil
	}
	byRole := map[string]model.GatewayModuleTtdc{}
	for _, module := range modules {
		if strings.EqualFold(module.TtdcConfig.ModuleAdr, request[3:5]) && frameHeaderMatches(module, request) {
			for role, t := range group.Templates {
				if module.TemplateID == t.ID {
					if _, duplicate := byRole[role]; duplicate {
						return PreparedBatteryChannels{}, false, permanentf("duplicate channel role %s for gateway %s ADR %s", role, gatewayID, request[3:5])
					}
					byRole[role] = module
				}
			}
		}
	}
	for _, role := range []string{model.TtdcRoleCID42, model.TtdcRoleCID42SOC, model.TtdcRoleCID44} {
		if _, exists := byRole[role]; !exists {
			return PreparedBatteryChannels{}, false, nil
		}
	}
	return PreparedBatteryChannels{GatewayID: gatewayID, TopicSegment: topicSegment, RawCID2: rawCID2, TemplateGroupID: group.ID, CurrentModule: byRole[rawCID2], SocModule: byRole[model.TtdcRoleCID42SOC], ModulesByRole: byRole}, true, nil
}

func validateAddressOwnership(address addressGroupSnapshot, group TemplateGroupSnapshot, request string) error {
	if address.TemplateGroupID != group.ID || address.TopicSegment != group.TopicSegment {
		return permanentf("gateway %s ADR %s is owned by another template group", address.GatewayID, address.ADR)
	}
	if address.SOI != request[0:1] || address.VER != request[1:3] || address.CID1 != request[5:7] {
		return permanentf("gateway %s ADR %s protocol header conflicts with historical binding", address.GatewayID, address.ADR)
	}
	return nil
}

func createMissingGroupChannels(db *gorm.DB, gatewayID, request string, group TemplateGroupSnapshot) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Serialize battery channel creation for the same gateway across deal
		// instances. This keeps configuration-only installations safe without
		// depending on the optional legacy-data finalize step.
		var lockedGateway model.Gateway
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Where("gateway_id = ?", gatewayID).First(&lockedGateway).Error; err != nil {
			return err
		}
		var lockedGroup model.TtdcTemplateGroup
		// A shared lock keeps management updates out while allowing independent
		// gateways using the same battery group to create channels concurrently.
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).First(&lockedGroup, group.ID).Error; err != nil {
			return err
		}
		if !lockedGroup.RuntimeMatchEnabled || !lockedGroup.AutoCreateEnabled || lockedGroup.TopicSegment != group.TopicSegment {
			return permanentf("template group changed while creating channels")
		}
		if lockedGroup.Version != group.Version {
			return fmt.Errorf("template group cache is stale while creating channels")
		}
		var lockedTemplates []model.TtdcTemplate
		if err := tx.Clauses(clause.Locking{Strength: "SHARE"}).Where("group_id = ?", group.ID).Find(&lockedTemplates).Error; err != nil {
			return err
		}
		if err := verifyCurrentTemplateVersions(group, lockedTemplates); err != nil {
			return err
		}
		adr := strings.ToUpper(request[3:5])
		var address model.GatewayTtdcAddressGroupBinding
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("gateway_id = ? AND module_adr = ?", gatewayID, adr).First(&address).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			address = model.GatewayTtdcAddressGroupBinding{GatewayID: gatewayID, ModuleADR: adr, TemplateGroupID: group.ID, GroupCode: group.GroupCode, TopicSegment: group.TopicSegment, ModuleSOI: request[0:1], ModuleVER: request[1:3], ModuleCID1: request[5:7], BindSource: "auto", BindingStatus: "active"}
			if err := tx.Create(&address).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		} else {
			snapshot := addressSnapshot(address)
			if address.BindingStatus == "inactive" {
				var residual int64
				if err := tx.Model(&model.GatewayTtdcTemplateBinding{}).Where("gateway_id = ? AND module_adr = ?", gatewayID, adr).Count(&residual).Error; err != nil {
					return err
				}
				if residual > 0 {
					return permanentf("inactive ADR %s has residual channels", adr)
				}
				// Reclaim inactive ownership created by an older deletion path. This is
				// safe only when no actual channel still references the ADR.
				address.TemplateGroupID = group.ID
				address.GroupCode = group.GroupCode
				address.TopicSegment = group.TopicSegment
				address.ModuleSOI = request[0:1]
				address.ModuleVER = request[1:3]
				address.ModuleCID1 = request[5:7]
				address.BindSource = "auto"
			} else if err := validateAddressOwnership(snapshot, group, request); err != nil {
				return err
			}
		}
		var existing []model.GatewayTtdcTemplateBinding
		if err := tx.Where("gateway_id = ? AND module_adr = ?", gatewayID, adr).Find(&existing).Error; err != nil {
			return err
		}
		byRole := map[string]model.GatewayTtdcTemplateBinding{}
		for _, binding := range existing {
			byRole[strings.ToUpper(strings.TrimSpace(binding.NormalizedCID2))] = binding
		}
		for _, role := range []string{model.TtdcRoleCID42, model.TtdcRoleCID42SOC, model.TtdcRoleCID44} {
			template, ok := group.Templates[role]
			if !ok {
				return permanentf("template group is missing role %s", role)
			}
			if binding, exists := byRole[role]; exists {
				if binding.TemplateGroupID != group.ID || binding.TemplateID != template.ID {
					return permanentf("existing channel role %s uses another template", role)
				}
				continue
			}
			if err := createRoleChannel(tx, gatewayID, request, group, template); err != nil {
				return err
			}
		}
		if address.BindingStatus == "inactive" {
			if err := tx.Model(&address).Updates(map[string]interface{}{
				"template_group_id": group.ID,
				"group_code":        group.GroupCode,
				"topic_segment":     group.TopicSegment,
				"module_soi":        request[0:1],
				"module_ver":        request[1:3],
				"module_cid1":       request[5:7],
				"bind_source":       "auto",
				"binding_status":    "active",
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Create(&model.TtdcConfigRevision{ConfigType: "address_group", TargetKey: addressKey(gatewayID, adr), Operation: "update"}).Error; err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "module", TargetKey: gatewayID, Operation: "update"}).Error
	})
}

func verifyCurrentTemplateVersions(group TemplateGroupSnapshot, rows []model.TtdcTemplate) error {
	current := make(map[string]model.TtdcTemplate, len(rows))
	for _, row := range rows {
		if _, duplicate := current[row.CID2Type]; duplicate {
			return permanentf("template group has duplicate role %s", row.CID2Type)
		}
		current[row.CID2Type] = row
	}
	for _, role := range []string{model.TtdcRoleCID42, model.TtdcRoleCID42SOC, model.TtdcRoleCID44} {
		snapshot, ok := group.Templates[role]
		row, currentOK := current[role]
		if !ok || !currentOK || row.ID != snapshot.ID || row.Version != snapshot.Version {
			return fmt.Errorf("template role %s cache is stale while creating channels", role)
		}
	}
	return nil
}

func createRoleChannel(tx *gorm.DB, gatewayID, request string, group TemplateGroupSnapshot, template TemplateSnapshot) error {
	adr := strings.ToUpper(request[3:5])
	module := model.GatewayModuleTtdc{GatewayId: gatewayID, Name: adr + "_" + template.TemplateName, Title: adr + "_" + template.TemplateName, Type: 1, Transport: "-"}
	if err := tx.Create(&module).Error; err != nil {
		return err
	}
	module.TtdcConfig = model.ModuleTtdcConfig{ModuleId: int(module.ID), ModuleSoi: request[0:1], ModuleVer: request[1:3], ModuleAdr: adr, ModuleCid1: request[5:7], ModuleCid2: template.MachineCID2, ModuleLength: "0000", ModuleInfo: "", ModuleChksum: "0000", ModuleEoi: ""}
	if err := tx.Create(&module.TtdcConfig).Error; err != nil {
		return err
	}
	for _, point := range template.Points {
		dataID := adr + "_" + point.BaseDataID
		name := adr + "_" + point.Name
		transport := strings.TrimSpace(point.Transport)
		if transport == "" && point.Address != nil && point.ByteCount != nil && point.ConvertType != nil {
			transport = fmt.Sprintf("%d %d %d", *point.Address, *point.ByteCount, *point.ConvertType)
			if point.BitIndex != nil {
				transport += fmt.Sprintf(" %d", *point.BitIndex)
			}
		}
		row := model.ModuleTtdcData{ModuleId: int(module.ID), Name: name, DataId: dataID, DataType: point.DataType, Transport: transport, Formula: point.Formula, Unformula: point.Unformula, AlgorithmKey: point.AlgorithmKey}
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.GatewayData{GatewayId: gatewayID, DataId: dataID, DataType: point.DataType, Name: name}).Error; err != nil {
			return err
		}
	}
	binding := model.GatewayTtdcTemplateBinding{ModuleID: module.ID, GatewayID: gatewayID, TemplateGroupID: group.ID, TemplateID: template.ID, TemplateCode: template.TemplateCode, TemplateVersion: template.Version, ProtocolCode: template.ProtocolCode, ParserProfile: template.ParserProfile, ModuleSOI: request[0:1], ModuleVER: request[1:3], ModuleADR: adr, ModuleCID1: request[5:7], NormalizedCID2: template.CID2Type, CreateSource: "auto"}
	if err := tx.Create(&binding).Error; err != nil {
		return err
	}
	batterylog.Printf("[BATTERY-AUTOCONFIG] channel created gatewayID=%s ADR=%s role=%s template=%s moduleID=%d\n", gatewayID, adr, template.CID2Type, template.TemplateCode, module.ID)
	return nil
}

func refreshAddressBinding(gatewayID, adr string) error {
	return RefreshAddressBinding(addressKey(gatewayID, adr))
}

func splitAddressKey(key string) (string, string, bool) {
	separator := strings.LastIndexByte(strings.TrimSpace(key), ':')
	if separator <= 0 || separator == len(strings.TrimSpace(key))-1 {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	gatewayID := key[:separator]
	adr, validADR := model.NormalizeTtdcADR(key[separator+1:])
	if !ValidateGatewayID(gatewayID) || !validADR {
		return "", "", false
	}
	return gatewayID, adr, true
}

// RefreshAddressBinding applies one revision without reloading every ADR.
func RefreshAddressBinding(key string) error {
	gatewayID, adr, ok := splitAddressKey(key)
	if !ok {
		return fmt.Errorf("invalid ADR cache key: %s", key)
	}
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return errors.New("opengauss gorm db is nil")
	}
	var row model.GatewayTtdcAddressGroupBinding
	err := db.Where("gateway_id = ? AND module_adr = ?", gatewayID, adr).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		addressGroupCacheState.Lock()
		delete(addressGroupCacheState.values, addressKey(gatewayID, adr))
		addressGroupCacheState.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	addressGroupCacheState.Lock()
	addressGroupCacheState.values[addressKey(gatewayID, adr)] = addressSnapshot(row)
	addressGroupCacheState.Unlock()
	return nil
}

func frameHeaderMatches(module model.GatewayModuleTtdc, request string) bool {
	cfg := module.TtdcConfig
	return len(request) >= 7 && strings.TrimSpace(cfg.ModuleSoi) == request[0:1] && strings.TrimSpace(cfg.ModuleVer) == request[1:3] && strings.EqualFold(strings.TrimSpace(cfg.ModuleAdr), request[3:5]) && strings.TrimSpace(cfg.ModuleCid1) == request[5:7]
}
