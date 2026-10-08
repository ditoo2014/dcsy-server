package schedule

import (
	"crypto/sha256"
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/common/model"
	tolinkmodel "dicloud/internal/tolink/model"
	"dicloud/util/opengauss"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"sync"
	"time"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"gorm.io/gorm"
)

const (
	ttdcGeneratedTriggerPrefix = "__TTDC_PUBLIC_RULE_"
	// OpenGauss in Oracle-compatible mode treats an empty string as NULL.
	// A whitespace placeholder satisfies the legacy NOT NULL columns while
	// remaining an empty notification target after downstream TrimSpace calls.
	ttdcNoNotificationValue = " "
	tdcAlarmSyncInterval    = 5 * time.Second
)

var ttdcAlarmSyncState struct {
	sync.Mutex
	last time.Time
}

var normalizedTtdcSingleTriggerDBs sync.Map

type ttdcAlarmRuleSnapshot struct {
	Rule      tolinkmodel.TtdcAlarmRule
	LabelName string
}

type ttdcAddressGroup struct {
	GatewayID       string `gorm:"column:gateway_id"`
	ModuleADR       string `gorm:"column:module_adr"`
	TemplateGroupID uint   `gorm:"column:template_group_id"`
}

type ttdcTenantPoint struct {
	EquipmentID int64  `gorm:"column:equipment_id"`
	SiteID      int64  `gorm:"column:site_id"`
	GatewayID   string `gorm:"column:gateway_id"`
	DataID      string `gorm:"column:data_id"`
	LabelName   string `gorm:"column:label_name"`
}

type ttdcTenantPointIndex map[string][]model.GeneralMarktrigger

// refreshTtdcAlarmRules synchronizes public TTDC rules at a bounded cadence.
// It is called from the existing business scheduler and does not alter the
// ordinary user-created trigger refresh path.
func refreshTtdcAlarmRules() {
	ttdcAlarmSyncState.Lock()
	if !ttdcAlarmSyncState.last.IsZero() && time.Since(ttdcAlarmSyncState.last) < tdcAlarmSyncInterval {
		ttdcAlarmSyncState.Unlock()
		return
	}
	ttdcAlarmSyncState.last = time.Now()
	ttdcAlarmSyncState.Unlock()

	defaultDB := opengauss.GetDefaultGormDB()
	if defaultDB == nil {
		log.Printf("[BUSINESS-TTDC-ALARM] skip public rule sync: default database is nil")
		return
	}
	if !defaultDB.Migrator().HasTable(&tolinkmodel.TtdcAlarmRule{}) {
		log.Printf("[BUSINESS-TTDC-ALARM] skip public rule sync: ttdc_alarm_rule table is absent")
		return
	}
	if !defaultDB.Migrator().HasTable(&tolinkmodel.GatewayTtdcAddressGroupBinding{}) {
		log.Printf("[BUSINESS-TTDC-ALARM] skip public rule sync: gateway_ttdc_address_group_binding table is absent")
		return
	}
	rules, err := loadTtdcAlarmRules(defaultDB)
	if err != nil {
		log.Printf("[BUSINESS-TTDC-ALARM] load public rules failed: %v", err)
		return
	}
	if len(rules) == 0 {
		return
	}
	addressGroups, err := loadTtdcAddressGroups(defaultDB)
	if err != nil {
		log.Printf("[BUSINESS-TTDC-ALARM] load ADR template groups failed: %v", err)
		return
	}

	for _, target := range ttdcAlarmTenantDatabases() {
		if target.db == nil || !target.db.Migrator().HasTable(&businessmodel.TtdcAlarmTriggerBinding{}) {
			log.Printf("[BUSINESS-TTDC-ALARM] skip datasource=%s: binding table is absent", target.name)
			continue
		}
		if err := normalizeTtdcSingleTriggerCycles(target.db); err != nil {
			log.Printf("[BUSINESS-TTDC-ALARM] normalize single-trigger cycles datasource=%s failed: %v", target.name, err)
			continue
		}
		if err := syncTtdcAlarmRulesForTenant(target.db, target.name, rules, addressGroups); err != nil {
			log.Printf("[BUSINESS-TTDC-ALARM] sync datasource=%s failed: %v", target.name, err)
		}
	}
}

// normalizeTtdcSingleTriggerCycles repairs runtime TTDC triggers created by
// earlier versions that still carried a cycle value. The in-memory guard
// keeps this compatibility update to one indexed UPDATE per datasource and
// never changes ordinary user-created triggers.
func normalizeTtdcSingleTriggerCycles(db *gorm.DB) error {
	if db == nil {
		return errors.New("TTDC trigger database is nil")
	}
	if _, loaded := normalizedTtdcSingleTriggerDBs.LoadOrStore(db, struct{}{}); loaded {
		return nil
	}
	result := db.Model(&model.GeneralTriggerConfig{}).
		Where("trigger_name LIKE ?", ttdcGeneratedTriggerPrefix+"%").
		Where("cycle <> ?", 0).
		Update("cycle", 0)
	if result.Error != nil {
		normalizedTtdcSingleTriggerDBs.Delete(db)
		return result.Error
	}
	if result.RowsAffected > 0 {
		log.Printf("[BUSINESS-TTDC-ALARM] normalized datasource TTDC triggers to single mode count=%d", result.RowsAffected)
	}
	return nil
}

type ttdcAlarmTenantDatabase struct {
	name string
	db   *gorm.DB
}

func ttdcAlarmTenantDatabases() []ttdcAlarmTenantDatabase {
	result := []ttdcAlarmTenantDatabase{{name: "default", db: opengauss.GetDefaultGormDB()}}
	for _, name := range opengauss.ConfiguredRenterDBNames() {
		if db := opengauss.GetRenterDB(name); db != nil {
			result = append(result, ttdcAlarmTenantDatabase{name: name, db: db})
		}
	}
	return result
}

func loadTtdcAlarmRules(db *gorm.DB) ([]ttdcAlarmRuleSnapshot, error) {
	var rules []tolinkmodel.TtdcAlarmRule
	query := db.Model(&tolinkmodel.TtdcAlarmRule{}).
		Preload("Logics", func(query *gorm.DB) *gorm.DB { return query.Order("sort_no asc, id asc") }).
		Joins("JOIN ttdc_template_group g ON g.id = ttdc_alarm_rule.template_group_id").
		Joins("JOIN ttdc_template t ON t.id = ttdc_alarm_rule.template_id").
		Joins("JOIN ttdc_template_point p ON p.id = ttdc_alarm_rule.template_point_id").
		// The referenced template point supplies the public label name. The
		// template group remains a management/reference relationship, not a
		// runtime battery-chemistry restriction.
		Where("ttdc_alarm_rule.enabled = ? AND g.deleted_at IS NULL AND t.enabled = ? AND t.deleted_at IS NULL AND p.enabled = ? AND p.deleted_at IS NULL AND p.data_type = ?", true, true, true, 1).
		Order("ttdc_alarm_rule.id asc")
	if err := query.Find(&rules).Error; err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, nil
	}

	// Resolve the template point's business label separately. Loading the
	// normal rule model first keeps GORM's Logics preload intact while allowing
	// the label to be used only by the public-rule matcher.
	type ttdcRuleLabel struct {
		RuleID    uint   `gorm:"column:rule_id"`
		LabelName string `gorm:"column:label_name"`
	}
	ruleIDs := make([]uint, 0, len(rules))
	for _, rule := range rules {
		ruleIDs = append(ruleIDs, rule.ID)
	}
	var labels []ttdcRuleLabel
	if err := db.Table("ttdc_alarm_rule AS r").
		Select("r.id AS rule_id, p.name AS label_name").
		Joins("JOIN ttdc_template_point AS p ON p.id = r.template_point_id").
		Where("r.id IN ?", ruleIDs).
		Find(&labels).Error; err != nil {
		return nil, err
	}
	labelByRuleID := make(map[uint]string, len(labels))
	for _, label := range labels {
		labelByRuleID[label.RuleID] = label.LabelName
	}

	result := make([]ttdcAlarmRuleSnapshot, 0, len(rules))
	for _, row := range rules {
		if row.TriggerModel != 0 {
			log.Printf("[BUSINESS-TTDC-ALARM] skip legacy non-numeric rule id=%d trigger_model=%d", row.ID, row.TriggerModel)
			continue
		}
		if row.TemplateGroupID == 0 || row.BaseDataID == "" || row.RuleCode == "" {
			log.Printf("[BUSINESS-TTDC-ALARM] skip invalid public rule id=%d", row.ID)
			continue
		}
		labelName := strings.TrimSpace(labelByRuleID[row.ID])
		if labelName == "" {
			log.Printf("[BUSINESS-TTDC-ALARM] skip public rule id=%d: template point label is empty", row.ID)
			continue
		}
		result = append(result, ttdcAlarmRuleSnapshot{Rule: row, LabelName: labelName})
	}
	return result, nil
}

func loadTtdcAddressGroups(db *gorm.DB) (map[string]uint, error) {
	var rows []ttdcAddressGroup
	if err := db.Model(&tolinkmodel.GatewayTtdcAddressGroupBinding{}).
		Where("binding_status = ?", "active").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]uint, len(rows))
	for _, row := range rows {
		gatewayID := strings.TrimSpace(row.GatewayID)
		adr, valid := tolinkmodel.NormalizeTtdcADR(row.ModuleADR)
		if gatewayID == "" || !valid || row.TemplateGroupID == 0 {
			continue
		}
		result[gatewayID+"\x00"+adr] = row.TemplateGroupID
	}
	return result, nil
}

func syncTtdcAlarmRulesForTenant(db *gorm.DB, name string, rules []ttdcAlarmRuleSnapshot, addressGroups map[string]uint) error {
	points, err := loadTenantTtdcPoints(db)
	if err != nil {
		return err
	}
	pointIndex := indexTtdcTenantPoints(points, addressGroups)
	desiredRules := make(map[uint]uint, len(rules))
	for _, snapshot := range rules {
		desiredRules[snapshot.Rule.ID] = snapshot.Rule.TemplateGroupID
		marks := resolveTtdcRuleMarks(snapshot, pointIndex)
		if len(marks) == 0 {
			log.Printf("[BUSINESS-TTDC-ALARM] datasource=%s rule=%d label=%q matched no TTDC data-pool points", name, snapshot.Rule.ID, snapshot.LabelName)
		}
		if err := syncOneTtdcAlarmRule(db, snapshot.Rule, marks); err != nil {
			log.Printf("[BUSINESS-TTDC-ALARM] datasource=%s rule=%d sync failed: %v", name, snapshot.Rule.ID, err)
		}
	}
	return removeDeletedTtdcAlarmRules(db, desiredRules)
}

func loadTenantTtdcPoints(db *gorm.DB) ([]ttdcTenantPoint, error) {
	var rows []ttdcTenantPoint
	// Public TTDC alarm rules only bind to battery gateways. The data pool
	// itself remains manually maintained, but the resolved point list must not
	// leak onto non-TTDC devices that happen to share the same stream schema.
	query := db.Table("equipment AS e").
		Select("e.id AS equipment_id, e.site_id, l.gateway_id, l.data_id, COALESCE(NULLIF(TRIM(d.label_name), ''), NULLIF(TRIM(d.name), ''), l.name) AS label_name").
		Joins("JOIN stream AS s ON s.api_key = e.api_key").
		Joins("JOIN stream_label_unit AS l ON l.stream_id = s.id").
		// Keep the original general-trigger association semantics: the public
		// rule selects by business label, while label_id is preferred to resolve
		// one concrete stream point. Name matching remains the compatibility
		// fallback used by the legacy trigger synchronizer.
		Joins("JOIN data_details AS d ON d.equipment_id = e.id AND (d.label_id = l.label_id OR TRIM(d.label_name) = TRIM(l.name) OR TRIM(d.name) = TRIM(l.name))").
		Joins("JOIN gateway AS g ON g.gateway_id = l.gateway_id AND g.model = ?", tolinkmodel.TTDC).
		Where("e.api_key IS NOT NULL AND l.gateway_id IS NOT NULL AND l.data_id IS NOT NULL")
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func resolveTtdcRuleMarks(snapshot ttdcAlarmRuleSnapshot, pointIndex ttdcTenantPointIndex) []model.GeneralMarktrigger {
	labelKey := normalizeTtdcLabelName(snapshot.LabelName)
	marks := append([]model.GeneralMarktrigger(nil), pointIndex[labelKey]...)
	sort.Slice(marks, func(i, j int) bool {
		if marks[i].EquipmentId != marks[j].EquipmentId {
			return marks[i].EquipmentId < marks[j].EquipmentId
		}
		return marks[i].Marktrigger < marks[j].Marktrigger
	})
	return marks
}

// normalizeTtdcLabelName follows the data-pool label convention used by the
// TTDC batch importer: surrounding whitespace and formatting whitespace do
// not change a label, and ASCII case differences are ignored.
func normalizeTtdcLabelName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "（", "(")
	value = strings.ReplaceAll(value, "）", ")")
	value = strings.Join(strings.Fields(value), "")
	return strings.ToUpper(value)
}

func splitTtdcDataID(dataID string) (string, string, bool) {
	separator := strings.IndexByte(dataID, '_')
	if separator <= 0 || separator == len(dataID)-1 {
		return "", "", false
	}
	adr, valid := tolinkmodel.NormalizeTtdcADR(dataID[:separator])
	if !valid {
		return "", "", false
	}
	return adr, strings.ToUpper(strings.TrimSpace(dataID[separator+1:])), true
}

func syncOneTtdcAlarmRule(db *gorm.DB, rule tolinkmodel.TtdcAlarmRule, marks []model.GeneralMarktrigger) error {
	if !db.Migrator().HasTable(&model.GeneralTriggerConfig{}) {
		return errors.New("通用触发器表不存在")
	}
	var binding businessmodel.TtdcAlarmTriggerBinding
	bindingErr := db.Where("alarm_rule_id = ? AND template_group_id = ?", rule.ID, rule.TemplateGroupID).First(&binding).Error
	if bindingErr != nil && !errors.Is(bindingErr, gorm.ErrRecordNotFound) {
		return bindingErr
	}
	markHash := hashTtdcMarks(marks)
	if errors.Is(bindingErr, gorm.ErrRecordNotFound) {
		return createGeneratedTtdcTrigger(db, rule, marks, markHash)
	}
	if binding.GeneralTriggerID == 0 {
		return recreateGeneratedTtdcTrigger(db, rule, marks, markHash, binding)
	}
	if binding.RuleVersion == rule.Version && binding.MarkHash == markHash && binding.SyncStatus == "ready" && binding.SyncError == "" {
		return repairGeneratedTtdcMarkAlarmState(db, binding.GeneralTriggerID, rule)
	}
	if binding.RuleVersion != rule.Version {
		if err := updateGeneratedTtdcTrigger(db, rule, binding.GeneralTriggerID, marks); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return recreateGeneratedTtdcTrigger(db, rule, marks, markHash, binding)
			}
			_ = db.Model(&binding).Updates(map[string]interface{}{"sync_status": "failed", "sync_error": err.Error()}).Error
			return err
		}
	} else if err := updateGeneratedTtdcMarks(db, binding.GeneralTriggerID, marks); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return recreateGeneratedTtdcTrigger(db, rule, marks, markHash, binding)
		}
		_ = db.Model(&binding).Updates(map[string]interface{}{"sync_status": "failed", "sync_error": err.Error()}).Error
		return err
	}
	if err := db.Model(&binding).Updates(map[string]interface{}{"rule_version": rule.Version, "mark_hash": markHash, "sync_status": "ready", "sync_error": ""}).Error; err != nil {
		return err
	}
	return repairGeneratedTtdcMarkAlarmState(db, binding.GeneralTriggerID, rule)
}

// repairGeneratedTtdcMarkAlarmState reconnects active alarm records to marks
// whose runtime state was lost when a generated trigger or binding changed.
// The next normal point value can then emit the regular recovery event.
func repairGeneratedTtdcMarkAlarmState(db *gorm.DB, triggerID uint, rule tolinkmodel.TtdcAlarmRule) error {
	if db == nil || triggerID == 0 || strings.TrimSpace(rule.RuleCode) == "" {
		return nil
	}

	var marks []model.GeneralMarktrigger
	if err := db.Where("trigger_id = ?", triggerID).
		Order("equipment_id ASC, id ASC").Find(&marks).Error; err != nil {
		return err
	}
	if len(marks) == 0 {
		return nil
	}

	equipmentIDs := make([]int, 0, len(marks))
	equipmentSeen := make(map[int]struct{}, len(marks))
	trackedBegins := make(map[int]map[int64]struct{})
	zeroMarks := make(map[int][]model.GeneralMarktrigger)
	for _, mark := range marks {
		if _, exists := equipmentSeen[mark.EquipmentId]; !exists {
			equipmentSeen[mark.EquipmentId] = struct{}{}
			equipmentIDs = append(equipmentIDs, mark.EquipmentId)
		}
		if mark.Record > 0 {
			if trackedBegins[mark.EquipmentId] == nil {
				trackedBegins[mark.EquipmentId] = make(map[int64]struct{})
			}
			trackedBegins[mark.EquipmentId][mark.Record] = struct{}{}
			continue
		}
		zeroMarks[mark.EquipmentId] = append(zeroMarks[mark.EquipmentId], mark)
	}
	if len(zeroMarks) == 0 {
		return nil
	}

	var activeRecords []model.AlarmDataRecord
	if err := db.Model(&model.AlarmDataRecord{}).
		Select("id, equipment_id, begin_time").
		Where("equipment_id IN ? AND alarm_code = ? AND alarm_status = ?", equipmentIDs, rule.RuleCode, 1).
		Order("equipment_id ASC, begin_time ASC, id ASC").Find(&activeRecords).Error; err != nil {
		return err
	}

	repaired := 0
	for _, record := range activeRecords {
		if record.BeginTime <= 0 || len(zeroMarks[record.EquipmentId]) == 0 {
			continue
		}
		if _, tracked := trackedBegins[record.EquipmentId][record.BeginTime]; tracked {
			continue
		}

		mark := zeroMarks[record.EquipmentId][0]
		zeroMarks[record.EquipmentId] = zeroMarks[record.EquipmentId][1:]
		result := db.Model(&model.GeneralMarktrigger{}).
			Where("id = ? AND trigger_id = ? AND record = ?", mark.ID, triggerID, 0).
			Update("record", record.BeginTime)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected > 0 {
			repaired++
		}
	}
	if repaired > 0 {
		log.Printf("[BUSINESS-TTDC-ALARM] restored active mark state trigger=%d count=%d", triggerID, repaired)
	}
	return nil
}

func createGeneratedTtdcTrigger(db *gorm.DB, rule tolinkmodel.TtdcAlarmRule, marks []model.GeneralMarktrigger, markHash string) error {
	trigger := generatedTtdcTrigger(rule)
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&trigger).Error; err != nil {
			return err
		}
		if err := createTtdcTriggerChildren(tx, &trigger); err != nil {
			return err
		}
		if err := replaceGeneratedTtdcMarks(tx, trigger.ID, marks); err != nil {
			return err
		}
		binding := businessmodel.TtdcAlarmTriggerBinding{AlarmRuleID: rule.ID, TemplateGroupID: rule.TemplateGroupID, GeneralTriggerID: trigger.ID, RuleVersion: rule.Version, MarkHash: markHash, SyncStatus: "ready"}
		return tx.Create(&binding).Error
	})
	if err != nil {
		return err
	}
	log.Printf("[BUSINESS-TTDC-ALARM] generated trigger created rule=%d trigger=%d marks=%d", rule.ID, trigger.ID, len(marks))
	return nil
}

// indexTtdcTenantPoints turns the O(rules * points) association scan into one
// point pass plus O(rules) lookups. Public rules are keyed by the data-pool
// label name, so one rule can cover every active TTDC channel that exposes the
// same business label, regardless of its template group or battery topic.
func indexTtdcTenantPoints(points []ttdcTenantPoint, addressGroups map[string]uint) ttdcTenantPointIndex {
	index := make(ttdcTenantPointIndex)
	seen := make(map[string]struct{})
	for _, point := range points {
		gatewayID := strings.TrimSpace(point.GatewayID)
		dataID := strings.TrimSpace(point.DataID)
		adr, _, ok := splitTtdcDataID(dataID)
		if !ok {
			continue
		}
		// ADR/template binding still identifies an active TTDC channel. It is
		// deliberately not used as the public-rule matching key.
		if _, bound := addressGroups[gatewayID+"\x00"+adr]; !bound || point.EquipmentID == 0 {
			continue
		}
		labelKey := normalizeTtdcLabelName(point.LabelName)
		if labelKey == "" {
			continue
		}
		identity := fmt.Sprintf("%d\x00%s\x00%s", point.EquipmentID, gatewayID, dataID)
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		index[labelKey] = append(index[labelKey], model.GeneralMarktrigger{
			SiteId: int(point.SiteID), EquipmentId: int(point.EquipmentID), Marktrigger: gatewayID + " " + dataID,
		})
	}
	return index
}

// recreateGeneratedTtdcTrigger repairs a stale binding whose generated
// general-trigger row was removed. The binding identity is retained so a
// deleted runtime instance cannot create duplicate bindings on every poll.
func recreateGeneratedTtdcTrigger(db *gorm.DB, rule tolinkmodel.TtdcAlarmRule, marks []model.GeneralMarktrigger, markHash string, binding businessmodel.TtdcAlarmTriggerBinding) error {
	trigger := generatedTtdcTrigger(rule)
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&trigger).Error; err != nil {
			return err
		}
		if err := createTtdcTriggerChildren(tx, &trigger); err != nil {
			return err
		}
		if err := replaceGeneratedTtdcMarks(tx, trigger.ID, marks); err != nil {
			return err
		}
		return tx.Model(&binding).Updates(map[string]interface{}{
			"general_trigger_id": trigger.ID,
			"rule_version":       rule.Version,
			"mark_hash":          markHash,
			"sync_status":        "ready",
			"sync_error":         "",
		}).Error
	})
}

func updateGeneratedTtdcMarks(db *gorm.DB, triggerID uint, marks []model.GeneralMarktrigger) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&model.GeneralTriggerConfig{}, triggerID).Error; err != nil {
			return err
		}
		return replaceGeneratedTtdcMarks(tx, triggerID, marks)
	})
}

func updateGeneratedTtdcTrigger(db *gorm.DB, rule tolinkmodel.TtdcAlarmRule, triggerID uint, marks []model.GeneralMarktrigger) error {
	trigger := generatedTtdcTrigger(rule)
	trigger.ID = triggerID
	return db.Transaction(func(tx *gorm.DB) error {
		var current model.GeneralTriggerConfig
		if err := tx.First(&current, triggerID).Error; err != nil {
			return err
		}
		if err := tx.Model(&current).Updates(map[string]interface{}{
			"trigger_name": trigger.TriggerName, "label_name": trigger.LabelName, "trigger_type": trigger.TriggerType,
			"trigger_model": trigger.TriggerModel, "code": trigger.Code, "cycle": trigger.Cycle,
			"sustain_time": trigger.SustainTime, "formula": trigger.Formula,
		}).Error; err != nil {
			return err
		}
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralRealtimeLogic{}).Error; err != nil {
			return err
		}
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralRealtimeStrMatch{}).Error; err != nil {
			return err
		}
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralAlarmUser{}).Error; err != nil {
			return err
		}
		if err := createTtdcTriggerChildren(tx, &trigger); err != nil {
			return err
		}
		return replaceGeneratedTtdcMarks(tx, triggerID, marks)
	})
}

func generatedTtdcTrigger(rule tolinkmodel.TtdcAlarmRule) model.GeneralTriggerConfig {
	trigger := model.GeneralTriggerConfig{
		// The legacy general-trigger table declares equipment_ids NOT NULL.
		// TTDC public rules are associated with concrete equipment through
		// general_marktrigger, so use an empty PostgreSQL array instead of NULL.
		EquipmentIds: pq.Int64Array{},
		TriggerName:  ttdcGeneratedTriggerPrefix + fmt.Sprint(rule.ID),
		LabelName:    ttdcGeneratedTriggerPrefix + fmt.Sprint(rule.ID),
		TriggerType:  0, TriggerModel: 0, Code: rule.RuleCode,
		// Public TTDC alarm rules are always single-trigger rules. Keep the
		// database field readable for compatibility, but never propagate a
		// legacy cycle value to the tenant runtime trigger.
		Cycle: 0, SustainTime: rule.SustainTime, Formula: defaultTtdcAlarmFormula(rule.Formula),
		AlarmUser: model.GeneralAlarmUser{
			AlarmCode: rule.RuleCode, AlarmType: rule.AlarmType, AlarmStage: rule.AlarmStage,
			AlarmChannel: ttdcNoNotificationValue, AlarmUsers: ttdcNoNotificationValue,
		},
	}
	for _, logic := range rule.Logics {
		trigger.Logics = append(trigger.Logics, model.GeneralRealtimeLogic{Logic: logic.Logic, TriggerValue: logic.TriggerValue})
	}
	return trigger
}

func createTtdcTriggerChildren(tx *gorm.DB, trigger *model.GeneralTriggerConfig) error {
	for index := range trigger.Logics {
		trigger.Logics[index].TriggerId = trigger.ID
		if err := tx.Create(&trigger.Logics[index]).Error; err != nil {
			return err
		}
	}
	for index := range trigger.StrMatchs {
		trigger.StrMatchs[index].TriggerId = trigger.ID
		if err := tx.Create(&trigger.StrMatchs[index]).Error; err != nil {
			return err
		}
	}
	trigger.AlarmUser.TriggerId = trigger.ID
	return tx.Create(&trigger.AlarmUser).Error
}

func replaceGeneratedTtdcMarks(tx *gorm.DB, triggerID uint, desired []model.GeneralMarktrigger) error {
	var current []model.GeneralMarktrigger
	if err := tx.Where("trigger_id = ?", triggerID).Find(&current).Error; err != nil {
		return err
	}
	byIdentity := make(map[string]model.GeneralMarktrigger, len(current))
	for _, mark := range current {
		byIdentity[fmt.Sprintf("%d\x00%s", mark.EquipmentId, mark.Marktrigger)] = mark
	}
	keep := make(map[uint]struct{}, len(desired))
	for _, next := range desired {
		identity := fmt.Sprintf("%d\x00%s", next.EquipmentId, next.Marktrigger)
		if previous, exists := byIdentity[identity]; exists {
			if err := tx.Model(&previous).Updates(map[string]interface{}{"site_id": next.SiteId, "equipment_id": next.EquipmentId, "marktrigger": next.Marktrigger}).Error; err != nil {
				return err
			}
			keep[previous.ID] = struct{}{}
			continue
		}
		next.TriggerId = triggerID
		if err := tx.Create(&next).Error; err != nil {
			return err
		}
		keep[next.ID] = struct{}{}
	}
	for _, previous := range current {
		if _, exists := keep[previous.ID]; !exists {
			if err := tx.Delete(&previous).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func hashTtdcMarks(marks []model.GeneralMarktrigger) string {
	identities := make([]string, 0, len(marks))
	for _, mark := range marks {
		identities = append(identities, fmt.Sprintf("%d\x00%d\x00%s", mark.EquipmentId, mark.SiteId, mark.Marktrigger))
	}
	sort.Strings(identities)
	hash := sha256.New()
	for _, identity := range identities {
		_, _ = hash.Write([]byte(identity))
		_, _ = hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func removeDeletedTtdcAlarmRules(db *gorm.DB, desired map[uint]uint) error {
	var bindings []businessmodel.TtdcAlarmTriggerBinding
	if err := db.Find(&bindings).Error; err != nil {
		return err
	}
	for _, binding := range bindings {
		if isDesiredTtdcAlarmBinding(binding, desired) {
			continue
		}
		if binding.GeneralTriggerID > 0 {
			if err := deleteGeneratedTtdcTrigger(db, binding.GeneralTriggerID); err != nil {
				return err
			}
		}
		if err := db.Delete(&binding).Error; err != nil {
			return err
		}
	}
	return nil
}

// A rule can be moved to another template group during editing. The binding
// identity includes the group, so the old group's generated trigger must be
// removed after the new binding is synchronized.
func isDesiredTtdcAlarmBinding(binding businessmodel.TtdcAlarmTriggerBinding, desired map[uint]uint) bool {
	templateGroupID, exists := desired[binding.AlarmRuleID]
	return exists && templateGroupID == binding.TemplateGroupID
}

func deleteGeneratedTtdcTrigger(db *gorm.DB, triggerID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralMarktrigger{}).Error; err != nil {
			return err
		}
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralRealtimeLogic{}).Error; err != nil {
			return err
		}
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralRealtimeStrMatch{}).Error; err != nil {
			return err
		}
		if err := tx.Where("trigger_id = ?", triggerID).Delete(&model.GeneralAlarmUser{}).Error; err != nil {
			return err
		}
		return tx.Delete(&model.GeneralTriggerConfig{}, triggerID).Error
	})
}

func defaultTtdcAlarmFormula(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "x"
	}
	return value
}
