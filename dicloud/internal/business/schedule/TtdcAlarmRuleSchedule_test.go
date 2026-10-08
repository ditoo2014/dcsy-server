package schedule

import (
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/common/model"
	tolinkmodel "dicloud/internal/tolink/model"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestIsDesiredTtdcAlarmBindingRequiresCurrentTemplateGroup(t *testing.T) {
	desired := map[uint]uint{7: 12}

	if !isDesiredTtdcAlarmBinding(businessmodel.TtdcAlarmTriggerBinding{
		AlarmRuleID: 7, TemplateGroupID: 12,
	}, desired) {
		t.Fatal("current rule/template-group binding should be retained")
	}
	if isDesiredTtdcAlarmBinding(businessmodel.TtdcAlarmTriggerBinding{
		AlarmRuleID: 7, TemplateGroupID: 9,
	}, desired) {
		t.Fatal("binding from the previous template group should be removed")
	}
	if isDesiredTtdcAlarmBinding(businessmodel.TtdcAlarmTriggerBinding{
		AlarmRuleID: 8, TemplateGroupID: 12,
	}, desired) {
		t.Fatal("deleted or disabled rule binding should be removed")
	}
}

func TestGeneratedTtdcTriggerAlwaysUsesNumericRulesAndNoRecipients(t *testing.T) {
	trigger := generatedTtdcTrigger(tolinkmodel.TtdcAlarmRule{
		ID: 9, RuleCode: "CELL_LOW", AlarmType: "1", AlarmStage: 2,
		TriggerModel: 1, AlarmChannel: "4", AlarmUsers: "1001", Cycle: 30,
		Logics: []tolinkmodel.TtdcAlarmRuleLogic{{Logic: "<", TriggerValue: 2.8, MatchValue: "legacy"}},
	})

	if trigger.TriggerModel != 0 {
		t.Fatalf("TTDC public trigger model = %d, want numeric mode 0", trigger.TriggerModel)
	}
	if trigger.EquipmentIds == nil {
		t.Fatal("TTDC public trigger equipment_ids must be an empty array, not NULL")
	}
	if len(trigger.EquipmentIds) != 0 {
		t.Fatalf("TTDC public trigger equipment_ids = %#v, want empty array", trigger.EquipmentIds)
	}
	if trigger.Cycle != 0 {
		t.Fatalf("TTDC public trigger cycle = %d, want single-trigger mode 0", trigger.Cycle)
	}
	if trigger.AlarmUser.AlarmChannel == "" || trigger.AlarmUser.AlarmUsers == "" {
		t.Fatalf("TTDC public trigger notification placeholders must satisfy NOT NULL columns: %#v", trigger.AlarmUser)
	}
	if strings.TrimSpace(trigger.AlarmUser.AlarmChannel) != "" || strings.TrimSpace(trigger.AlarmUser.AlarmUsers) != "" {
		t.Fatalf("TTDC public trigger should not resolve notification channels or recipients: %#v", trigger.AlarmUser)
	}
	if len(trigger.StrMatchs) != 0 || len(trigger.Logics) != 1 {
		t.Fatalf("TTDC public trigger children = logics:%d strings:%d, want 1 numeric logic and no string logic", len(trigger.Logics), len(trigger.StrMatchs))
	}
	if trigger.Logics[0] != (model.GeneralRealtimeLogic{Logic: "<", TriggerValue: 2.8}) {
		t.Fatalf("unexpected generated numeric logic: %#v", trigger.Logics[0])
	}
}

func TestIndexTtdcTenantPointsUsesDataPoolLabelsAcrossTemplateGroups(t *testing.T) {
	points := []ttdcTenantPoint{
		{EquipmentID: 101, SiteID: 11, GatewayID: "gateway-a", DataID: "01_alg_soc", LabelName: "SOC"},
		{EquipmentID: 102, SiteID: 12, GatewayID: "gateway-b", DataID: "02_alg_soc", LabelName: " soc "},
		{EquipmentID: 103, SiteID: 13, GatewayID: "gateway-c", DataID: "03_alg_soh", LabelName: "SOH"},
		{EquipmentID: 104, SiteID: 14, GatewayID: "gateway-d", DataID: "04_alg_soc", LabelName: "SOC"},
	}
	activeChannels := map[string]uint{
		"gateway-a\x0001": 1,
		"gateway-b\x0002": 2,
		// gateway-d is intentionally absent: an inactive/unbound channel must
		// not receive a public trigger mark.
	}

	index := indexTtdcTenantPoints(points, activeChannels)
	marks := index[normalizeTtdcLabelName("SOC")]
	if len(marks) != 2 {
		t.Fatalf("SOC marks = %d, want 2", len(marks))
	}
	if marks[0].EquipmentId != 101 || marks[1].EquipmentId != 102 {
		t.Fatalf("unexpected SOC marks: %#v", marks)
	}
}

func TestResolveTtdcRuleMarksUsesTemplatePointLabel(t *testing.T) {
	index := ttdcTenantPointIndex{
		"SOC":      {{EquipmentId: 1, Marktrigger: "gateway 01_alg_soc"}},
		"BASE_SOC": {{EquipmentId: 2, Marktrigger: "gateway 01_legacy"}},
	}
	snapshot := ttdcAlarmRuleSnapshot{
		Rule:      tolinkmodel.TtdcAlarmRule{TemplateGroupID: 99, BaseDataID: "BASE_SOC"},
		LabelName: "SOC",
	}

	marks := resolveTtdcRuleMarks(snapshot, index)
	if len(marks) != 1 || marks[0].EquipmentId != 1 {
		t.Fatalf("resolved marks = %#v, want SOC label marks", marks)
	}
}

func TestNormalizeTtdcLabelNameMatchesImporterFormatting(t *testing.T) {
	if got := normalizeTtdcLabelName(" 电池组（SOC） "); got != "电池组(SOC)" {
		t.Fatalf("normalized label = %q, want %q", got, "电池组(SOC)")
	}
}

func TestRepairGeneratedTtdcMarkAlarmStateRestoresLostRecords(t *testing.T) {
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.GeneralMarktrigger{}, &model.AlarmDataRecord{}); err != nil {
		t.Fatalf("migrate test tables: %v", err)
	}

	marks := []model.GeneralMarktrigger{
		{TriggerId: 9, EquipmentId: 101, Marktrigger: "gateway 01_alarm", Record: 100},
		{TriggerId: 9, EquipmentId: 101, Marktrigger: "gateway 02_alarm"},
		{TriggerId: 9, EquipmentId: 102, Marktrigger: "gateway 01_alarm"},
	}
	if err := db.Create(&marks).Error; err != nil {
		t.Fatalf("create marks: %v", err)
	}
	records := []model.AlarmDataRecord{
		{EquipmentId: 101, AlarmCode: "欠压告警", AlarmStatus: 1, BeginTime: 100},
		{EquipmentId: 101, AlarmCode: "欠压告警", AlarmStatus: 1, BeginTime: 200},
		{EquipmentId: 102, AlarmCode: "欠压告警", AlarmStatus: 1, BeginTime: 300},
		{EquipmentId: 102, AlarmCode: "其他告警", AlarmStatus: 1, BeginTime: 400},
	}
	if err := db.Create(&records).Error; err != nil {
		t.Fatalf("create alarm records: %v", err)
	}

	if err := repairGeneratedTtdcMarkAlarmState(db, 9, tolinkmodel.TtdcAlarmRule{RuleCode: "欠压告警"}); err != nil {
		t.Fatalf("repair mark state: %v", err)
	}

	var repaired []model.GeneralMarktrigger
	if err := db.Where("trigger_id = ?", 9).Order("equipment_id ASC, id ASC").Find(&repaired).Error; err != nil {
		t.Fatalf("find repaired marks: %v", err)
	}
	if len(repaired) != 3 || repaired[0].Record != 100 || repaired[1].Record != 200 || repaired[2].Record != 300 {
		t.Fatalf("unexpected repaired mark state: %#v", repaired)
	}
}
