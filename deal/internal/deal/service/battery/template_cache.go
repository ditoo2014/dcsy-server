package battery

import (
	"deal/internal/batterylog"
	"deal/internal/tolink/model"
	"deal/util/opengauss"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

type TemplateSnapshot struct {
	ID                  uint
	GroupID             uint
	GroupCode           string
	GroupName           string
	TopicSegment        string
	TemplateCode        string
	TemplateName        string
	CID2Type            string
	ProtocolCode        string
	MachineCID2         string
	ParserProfile       string
	GroupRuntimeEnabled bool
	GroupAutoCreate     bool
	Version             int
	Points              []model.TtdcTemplatePoint
}

type TemplateGroupSnapshot struct {
	ID                  uint
	GroupCode           string
	GroupName           string
	TopicSegment        string
	RuntimeMatchEnabled bool
	AutoCreateEnabled   bool
	Version             int
	Templates           map[string]TemplateSnapshot
}

var (
	templateCacheMu       sync.RWMutex
	templateCache         = map[string]TemplateSnapshot{}
	groupCache            = map[string]TemplateGroupSnapshot{}
	legacyTemplateMode    bool
	templateRuntimeReady  bool
	templateNegativeCache = map[string]time.Time{}
)

func templateKey(topicSegment, rawCID2 string) string {
	return strings.TrimSpace(topicSegment) + ":" + strings.ToUpper(strings.TrimSpace(rawCID2))
}

func groupTemplateKey(groupID uint, role string) string {
	return fmt.Sprintf("%d:%s", groupID, strings.ToUpper(strings.TrimSpace(role)))
}

func LoadTemplateCache(_ interface{}) error {
	db := opengauss.GetDefaultGormDB()
	if db == nil {
		return errors.New("opengauss gorm db is nil")
	}
	if !templateGroupSchemaReady(db) {
		return loadLegacyTemplateCache(db)
	}
	var groups []model.TtdcTemplateGroup
	if err := db.Preload("Templates", func(q *gorm.DB) *gorm.DB {
		return q.Preload("Points", func(p *gorm.DB) *gorm.DB { return p.Where("enabled = ?", true).Order("sort_no asc, id asc") }).Order("cid2_type asc")
	}).Where("runtime_match_enabled = ?", true).Order("id asc").Find(&groups).Error; err != nil {
		return err
	}
	nextGroups := map[string]TemplateGroupSnapshot{}
	nextTemplates := map[string]TemplateSnapshot{}
	for _, group := range groups {
		roles := map[string]int{}
		snapshot := TemplateGroupSnapshot{ID: group.ID, GroupCode: group.GroupCode, GroupName: group.GroupName, TopicSegment: group.TopicSegment, RuntimeMatchEnabled: group.RuntimeMatchEnabled, AutoCreateEnabled: group.AutoCreateEnabled, Version: group.Version, Templates: map[string]TemplateSnapshot{}}
		for _, item := range group.Templates {
			roles[item.CID2Type]++
			pointSnapshot := append([]model.TtdcTemplatePoint(nil), item.Points...)
			template := TemplateSnapshot{ID: item.ID, GroupID: group.ID, GroupCode: group.GroupCode, GroupName: group.GroupName, TopicSegment: group.TopicSegment, TemplateCode: item.TemplateCode, TemplateName: item.TemplateName, CID2Type: item.CID2Type, ProtocolCode: item.ProtocolCode, MachineCID2: item.MachineCID2, ParserProfile: item.ParserProfile, GroupRuntimeEnabled: group.RuntimeMatchEnabled, GroupAutoCreate: group.AutoCreateEnabled, Version: item.Version, Points: pointSnapshot}
			snapshot.Templates[item.CID2Type] = template
			nextTemplates[groupTemplateKey(group.ID, item.CID2Type)] = template
		}
		if roles[model.TtdcRoleCID42] != 1 || roles[model.TtdcRoleCID42SOC] != 1 || roles[model.TtdcRoleCID44] != 1 {
			batterylog.Printf("[BATTERY-TEMPLATE] group=%s is incomplete; runtime mapping disabled for this group\n", group.GroupCode)
			continue
		}
		if err := validateTemplateGroupStructure(snapshot); err != nil {
			batterylog.Printf("[BATTERY-TEMPLATE] group=%s is invalid; runtime mapping disabled: %v\n", group.GroupCode, err)
			continue
		}
		if _, exists := nextGroups[group.TopicSegment]; exists {
			return fmt.Errorf("conflicting runtime template group topic: %s", group.TopicSegment)
		}
		nextGroups[group.TopicSegment] = snapshot
	}
	templateCacheMu.Lock()
	groupCache, templateCache = nextGroups, nextTemplates
	legacyTemplateMode, templateRuntimeReady = false, true
	templateNegativeCache = map[string]time.Time{}
	templateCacheMu.Unlock()
	batterylog.Printf("[BATTERY-TEMPLATE] template-group cache loaded groups=%d roles=%d\n", len(nextGroups), len(nextTemplates))
	return nil
}

func templateGroupSchemaReady(db *gorm.DB) bool {
	if !db.Migrator().HasTable(&model.TtdcTemplateGroup{}) || !db.Migrator().HasTable(&model.TtdcSchemaVersionRecord{}) {
		return false
	}
	var state model.TtdcSchemaVersionRecord
	err := db.Where("component = ?", model.TtdcSchemaComponent).First(&state).Error
	if err != nil || state.Version != model.TtdcSchemaVersion || !model.TtdcSchemaRuntimeUsable(state.Status) {
		if err == nil {
			batterylog.Printf("[BATTERY-TEMPLATE] template-group schema is not initialized version=%d status=%s; using legacy compatibility\n", state.Version, state.Status)
		}
		return false
	}
	return true
}

func loadLegacyTemplateCache(db *gorm.DB) error {
	if !db.Migrator().HasTable(&model.TtdcTemplate{}) || !db.Migrator().HasTable(&model.TtdcTemplateTopic{}) {
		clearTemplateCache(true, false)
		return nil
	}
	var topics []model.TtdcTemplateTopic
	if err := db.Where("runtime_match_enabled = ?", true).Order("id asc").Find(&topics).Error; err != nil {
		return err
	}
	next := map[string]TemplateSnapshot{}
	for _, topic := range topics {
		var item model.TtdcTemplate
		if err := db.Preload("Points", func(q *gorm.DB) *gorm.DB { return q.Where("enabled = ?", true).Order("sort_no asc, id asc") }).First(&item, topic.TemplateID).Error; err != nil {
			return err
		}
		next[templateKey(topic.TopicSegment, topic.RawCID2)] = TemplateSnapshot{ID: item.ID, TemplateCode: item.TemplateCode, TemplateName: item.TemplateName, CID2Type: item.CID2Type, ProtocolCode: item.ProtocolCode, MachineCID2: item.MachineCID2, ParserProfile: item.ParserProfile, Version: item.Version, Points: append([]model.TtdcTemplatePoint(nil), item.Points...)}
	}
	templateCacheMu.Lock()
	templateCache, groupCache, legacyTemplateMode, templateRuntimeReady = next, map[string]TemplateGroupSnapshot{}, true, false
	templateNegativeCache = map[string]time.Time{}
	templateCacheMu.Unlock()
	batterylog.Printf("[BATTERY-TEMPLATE] legacy template cache loaded mappings=%d\n", len(next))
	return nil
}

func clearTemplateCache(legacy, ready bool) {
	templateCacheMu.Lock()
	templateCache, groupCache, legacyTemplateMode, templateRuntimeReady = map[string]TemplateSnapshot{}, map[string]TemplateGroupSnapshot{}, legacy, ready
	templateCacheMu.Unlock()
}

func MatchTemplate(topicSegment, rawCID2 string) (TemplateSnapshot, bool) {
	templateCacheMu.RLock()
	item, ok := templateCache[templateKey(topicSegment, rawCID2)]
	if !ok {
		if group, groupOK := groupCache[strings.TrimSpace(topicSegment)]; groupOK {
			item, ok = group.Templates[strings.ToUpper(strings.TrimSpace(rawCID2))]
		}
	}
	templateCacheMu.RUnlock()
	return item, ok
}

func MatchTemplateAt(_ interface{}, topicSegment, rawCID2 string) (TemplateSnapshot, bool, error) {
	if item, ok := MatchTemplate(topicSegment, rawCID2); ok {
		return item, true, nil
	}
	templateCacheMu.RLock()
	legacy, ready := legacyTemplateMode, templateRuntimeReady
	templateCacheMu.RUnlock()
	if legacy || !ready {
		return TemplateSnapshot{}, false, nil
	}
	return TemplateSnapshot{}, false, nil
}

func MatchTemplateGroup(topicSegment string) (TemplateGroupSnapshot, bool) {
	templateCacheMu.RLock()
	item, ok := groupCache[strings.TrimSpace(topicSegment)]
	templateCacheMu.RUnlock()
	return item, ok
}
func LegacyTemplateMode() bool {
	templateCacheMu.RLock()
	defer templateCacheMu.RUnlock()
	return legacyTemplateMode
}
func TemplateRuntimeReady() bool {
	templateCacheMu.RLock()
	defer templateCacheMu.RUnlock()
	return templateRuntimeReady
}

func startTemplateCacheRefresher() {
	go func() {
		ticker := time.NewTicker(templateRefreshInterval())
		defer ticker.Stop()
		for range ticker.C {
			if err := LoadTemplateCache(nil); err != nil {
				batterylog.Printf("[BATTERY-TEMPLATE] refresh failed: %v\n", err)
			}
		}
	}()
}

func templateRefreshInterval() time.Duration { return 30 * time.Second }
