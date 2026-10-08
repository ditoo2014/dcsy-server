package service

import (
	"dicloud/util/opengauss"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"gorm.io/gorm"
)

type batteryTriggerRouteState struct {
	sync.RWMutex
	byDatasource map[string]map[string]struct{}
	byPoint      map[string][]string
}

var batteryTriggerRoutes = batteryTriggerRouteState{
	byDatasource: make(map[string]map[string]struct{}),
	byPoint:      make(map[string][]string),
}

type batteryTriggerDatasource struct {
	name string
	db   *gorm.DB
}

// RefreshBatteryTriggerRoutes builds the technical database route for TTDC
// points from the trigger bindings already materialized in each datasource.
// Rules still bind to equipment_id; the datasource name only tells
// trigger/alarm where that equipment is stored.
func RefreshBatteryTriggerRoutes() {
	targets := []batteryTriggerDatasource{{db: opengauss.GetDefaultGormDB()}}
	configured := opengauss.ConfiguredRenterDBNames()
	for _, name := range configured {
		targets = append(targets, batteryTriggerDatasource{name: name, db: opengauss.GetRenterDB(name)})
	}

	active := make(map[string]struct{}, len(targets))
	changed := false
	for _, target := range targets {
		active[target.name] = struct{}{}
		marks, err := loadBatteryTriggerMarks(target.db)
		if err != nil {
			log.Printf("[TRIGGER-BATTERY] refresh routes datasource=%s failed: %v", batteryDatasourceLabel(target.name), err)
			continue
		}
		if replaceBatteryDatasourceRoutes(target.name, marks) {
			changed = true
		}
	}
	if removeInactiveBatteryDatasources(active) {
		changed = true
	}
	if changed {
		log.Printf("[TRIGGER-BATTERY] routes refreshed datasources=%d points=%d", len(active), batteryTriggerRouteCount())
	}
}

func loadBatteryTriggerMarks(db *gorm.DB) (map[string]struct{}, error) {
	if db == nil {
		return nil, fmt.Errorf("database is unavailable")
	}
	result := make(map[string]struct{})
	if db.Migrator().HasTable("general_marktrigger") {
		if err := appendBatteryTriggerMarks(db.Table("general_marktrigger"), result); err != nil {
			return nil, err
		}
	}
	// Battery packets used trigger_dor before the dedicated tag existed. Keep
	// tenant-side legacy trigger_config bindings routable as well.
	if db.Migrator().HasTable("trigger_config") {
		if err := appendBatteryTriggerMarks(db.Table("trigger_config"), result); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func appendBatteryTriggerMarks(query *gorm.DB, result map[string]struct{}) error {
	var values []string
	if err := query.Distinct("marktrigger").
		Where("marktrigger IS NOT NULL").
		Pluck("marktrigger", &values).Error; err != nil {
		return err
	}
	for _, value := range values {
		if mark := strings.TrimSpace(value); mark != "" {
			result[mark] = struct{}{}
		}
	}
	return nil
}

func replaceBatteryDatasourceRoutes(dbName string, marks map[string]struct{}) bool {
	batteryTriggerRoutes.Lock()
	defer batteryTriggerRoutes.Unlock()
	if sameBatteryMarkSet(batteryTriggerRoutes.byDatasource[dbName], marks) {
		return false
	}
	batteryTriggerRoutes.byDatasource[dbName] = marks
	rebuildBatteryPointRoutesLocked()
	return true
}

func removeInactiveBatteryDatasources(active map[string]struct{}) bool {
	batteryTriggerRoutes.Lock()
	defer batteryTriggerRoutes.Unlock()
	changed := false
	for name := range batteryTriggerRoutes.byDatasource {
		if _, ok := active[name]; ok {
			continue
		}
		delete(batteryTriggerRoutes.byDatasource, name)
		changed = true
	}
	if changed {
		rebuildBatteryPointRoutesLocked()
	}
	return changed
}

func rebuildBatteryPointRoutesLocked() {
	byPoint := make(map[string][]string)
	for dbName, marks := range batteryTriggerRoutes.byDatasource {
		for mark := range marks {
			byPoint[mark] = append(byPoint[mark], dbName)
		}
	}
	for mark := range byPoint {
		sort.Strings(byPoint[mark])
	}
	batteryTriggerRoutes.byPoint = byPoint
}

func batteryTriggerDBNames(gatewayID, dataID string) []string {
	mark := strings.TrimSpace(gatewayID) + " " + strings.TrimSpace(dataID)
	batteryTriggerRoutes.RLock()
	names := append([]string(nil), batteryTriggerRoutes.byPoint[mark]...)
	batteryTriggerRoutes.RUnlock()
	return names
}

func batteryTriggerRouteCount() int {
	batteryTriggerRoutes.RLock()
	count := len(batteryTriggerRoutes.byPoint)
	batteryTriggerRoutes.RUnlock()
	return count
}

func sameBatteryMarkSet(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, ok := right[value]; !ok {
			return false
		}
	}
	return true
}

func batteryDatasourceLabel(name string) string {
	if strings.TrimSpace(name) == "" {
		return "default"
	}
	return name
}

// HandleBatteryTrigger routes one parsed battery point to every datasource
// that owns an equipment binding for that public TTDC rule. Legacy messages
// carrying dbName remain accepted during rolling deployment.
func HandleBatteryTrigger(message *primitive.MessageExt) {
	if message == nil {
		return
	}
	gatewayID, dataID, value, timestamp, legacyDBName, ok := parseTriggerPointText(string(message.Body))
	if !ok {
		log.Printf("[TRIGGER-BATTERY] invalid point message msgId=%s", message.MsgId)
		return
	}

	dbNames := batteryTriggerDBNames(gatewayID, dataID)
	if strings.TrimSpace(legacyDBName) != "" {
		dbNames = appendUniqueBatteryDBName(dbNames, legacyDBName)
	}
	if len(dbNames) == 0 {
		// Preserve the pre-route behavior for the default database while the
		// business synchronizer is still creating the first generated mark.
		dbNames = []string{""}
	}

	for _, dbName := range dbNames {
		routed := *message
		routed.Body = []byte(formatBatteryTriggerPointText(gatewayID, dataID, value, timestamp, dbName))
		HandleDorTrigger(&routed)
		HandleTypeStrSwitchTrigger(&routed)
	}
}

func formatBatteryTriggerPointText(gatewayID, dataID, value string, timestamp int64, dbName string) string {
	body := fmt.Sprintf("%s %s %s %d", gatewayID, dataID, value, timestamp)
	if name := strings.TrimSpace(dbName); name != "" {
		body += " " + name
	}
	return body
}

func appendUniqueBatteryDBName(names []string, value string) []string {
	value = strings.TrimSpace(value)
	for _, name := range names {
		if name == value {
			return names
		}
	}
	return append(names, value)
}
