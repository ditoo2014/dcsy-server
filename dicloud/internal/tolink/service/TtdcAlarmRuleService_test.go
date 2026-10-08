package service

import "testing"

func TestNormalizeTtdcAlarmRuleRequestForcesPublicNumericDefaults(t *testing.T) {
	item, logics, err := normalizeTtdcAlarmRuleRequest(TtdcAlarmRuleRequest{
		TemplateGroupID: 1, TemplateID: 2, TemplatePointID: 3,
		BaseDataID: "cell_voltage", RuleName: "单体电压低", RuleCode: "CELL_LOW",
		AlarmType: "1", AlarmStage: 2, AlarmChannel: "4", AlarmUsers: "1001",
		TriggerModel: 1, SortNo: 99, Formula: "x * 10", Cycle: 30,
		Logics: []TtdcAlarmRuleLogicRequest{{Logic: "<", TriggerValue: 2.8, MatchValue: "legacy", SortNo: 99}},
	})
	if err != nil {
		t.Fatalf("normalize public alarm rule: %v", err)
	}
	if item.TriggerModel != 0 || item.AlarmChannel != "" || item.AlarmUsers != "" || item.SortNo != 0 || item.Cycle != 0 {
		t.Fatalf("public rule defaults were not forced: %#v", item)
	}
	if len(logics) != 1 || logics[0].Logic != "<" || logics[0].TriggerValue != 2.8 || logics[0].MatchValue != "" || logics[0].SortNo != 1 {
		t.Fatalf("unexpected normalized numeric logic: %#v", logics)
	}
}

func TestNormalizeTtdcAlarmRuleRequestRejectsNonNumericOperator(t *testing.T) {
	_, _, err := normalizeTtdcAlarmRuleRequest(TtdcAlarmRuleRequest{
		TemplateGroupID: 1, TemplateID: 2, TemplatePointID: 3,
		BaseDataID: "cell_voltage", RuleName: "单体电压低", RuleCode: "CELL_LOW", AlarmType: "0",
		Logics: []TtdcAlarmRuleLogicRequest{{Logic: "match", TriggerValue: 1}},
	})
	if err == nil {
		t.Fatal("expected non-numeric operator to be rejected")
	}
}
