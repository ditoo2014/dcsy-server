package service

import (
	"dicloud/internal/tolink/model"
	"errors"
	"fmt"
	"strings"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTemplateErrorMessageUsesChineseResponse(t *testing.T) {
	const businessMessage = "模板数据点配置不完整"
	if got := templateErrorMessage(errors.New(businessMessage)); got != businessMessage {
		t.Fatalf("Chinese business error = %q, want %q", got, businessMessage)
	}
	if got := templateErrorMessage(errors.New("pq: syntax error")); got != "模板操作失败，请查看服务日志" {
		t.Fatalf("internal error response = %q", got)
	}
}

func newTtdcTemplateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err = db.AutoMigrate(
		&model.TtdcTemplateGroup{},
		&model.TtdcTemplate{},
		&model.TtdcTemplateTopic{},
		&model.TtdcTemplatePoint{},
		&model.GatewayTtdcAddressGroupBinding{},
		&model.GatewayTtdcTemplateBinding{},
		&model.TtdcConfigRevision{},
		&model.TtdcSchemaVersionRecord{},
		&model.GatewayModuleTtdc{},
		&model.ModuleTtdcData{},
		&model.GatewayData{},
	); err != nil {
		t.Fatal(err)
	}
	if err = db.Create(&model.TtdcSchemaVersionRecord{
		Component: model.TtdcSchemaComponent,
		Version:   model.TtdcSchemaVersion,
		Status:    model.TtdcSchemaConfig,
	}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestPropagateTemplatePointAdditionsOnlyAddsMissingRows(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	_, item := createTtdcTemplateFixture(t, db)
	module := model.GatewayModuleTtdc{GatewayId: "860000000000001", Name: "01_42", Title: "01_42"}
	if err := db.Create(&module).Error; err != nil {
		t.Fatal(err)
	}
	binding := model.GatewayTtdcTemplateBinding{
		ModuleID: module.ID, GatewayID: module.GatewayId, TemplateGroupID: item.GroupID,
		TemplateID: item.ID, TemplateCode: item.TemplateCode, TemplateVersion: 1,
		ProtocolCode: item.ProtocolCode, ParserProfile: item.ParserProfile,
		ModuleSOI: "~", ModuleVER: "20", ModuleADR: "01", ModuleCID1: "4A",
		NormalizedCID2: model.TtdcRoleCID42, CreateSource: "auto",
	}
	if err := db.Create(&binding).Error; err != nil {
		t.Fatal(err)
	}
	existingModulePoint := model.ModuleTtdcData{ModuleId: int(module.ID), DataId: "01_existing", Name: "现场自定义名称", DataType: 2, Transport: "custom", Formula: "x+1"}
	existingGatewayPoint := model.GatewayData{GatewayId: module.GatewayId, DataId: "01_existing", Name: "现场目录名称", DataType: 2}
	if err := db.Create(&existingModulePoint).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&existingGatewayPoint).Error; err != nil {
		t.Fatal(err)
	}
	address, byteCount, convertType := 10, 2, 1
	added := model.TtdcTemplatePoint{
		TemplateID: item.ID, GroupID: item.GroupID, BaseDataID: "new_point", Name: "新增点",
		DataType: 1, Address: &address, ByteCount: &byteCount, ConvertType: &convertType,
		Formula: "x/10", Unformula: "x*10", Enabled: true,
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return propagateTemplatePointAdditions(tx, item.ID, []model.TtdcTemplatePoint{added})
	}); err != nil {
		t.Fatal(err)
	}
	// Replaying the same propagation must remain idempotent.
	if err := db.Transaction(func(tx *gorm.DB) error {
		return propagateTemplatePointAdditions(tx, item.ID, []model.TtdcTemplatePoint{added})
	}); err != nil {
		t.Fatal(err)
	}

	var existingAfter model.ModuleTtdcData
	if err := db.Where("module_id = ? AND data_id = ?", module.ID, "01_existing").First(&existingAfter).Error; err != nil {
		t.Fatal(err)
	}
	if existingAfter.Name != "现场自定义名称" || existingAfter.DataType != 2 || existingAfter.Transport != "custom" || existingAfter.Formula != "x+1" {
		t.Fatalf("existing channel point was modified: %+v", existingAfter)
	}
	var moduleCount, gatewayCount int64
	if err := db.Model(&model.ModuleTtdcData{}).Where("module_id = ? AND data_id = ?", module.ID, "01_new_point").Count(&moduleCount).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&model.GatewayData{}).Where("gateway_id = ? AND data_id = ?", module.GatewayId, "01_new_point").Count(&gatewayCount).Error; err != nil {
		t.Fatal(err)
	}
	if moduleCount != 1 || gatewayCount != 1 {
		t.Fatalf("new point counts module=%d gateway=%d", moduleCount, gatewayCount)
	}
	var saved model.ModuleTtdcData
	if err := db.Where("module_id = ? AND data_id = ?", module.ID, "01_new_point").First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Transport != "10 2 1" || saved.Formula != "x/10" {
		t.Fatalf("new point snapshot = %+v", saved)
	}
}

func TestAddedEnabledTemplatePointsExcludesExistingAndReenabledRows(t *testing.T) {
	previous := []model.TtdcTemplatePoint{
		{ID: 1, BaseDataID: "existing", Enabled: true},
		{ID: 2, BaseDataID: "disabled", Enabled: false},
	}
	current := []model.TtdcTemplatePoint{
		{ID: 1, BaseDataID: "existing", Enabled: true},
		{ID: 2, BaseDataID: "disabled", Enabled: true},
		{ID: 3, BaseDataID: "new_disabled", Enabled: false},
		{ID: 4, BaseDataID: "new_enabled", Enabled: true},
	}
	added := addedEnabledTemplatePoints(previous, current)
	if len(added) != 1 || added[0].BaseDataID != "new_enabled" {
		t.Fatalf("added points = %+v", added)
	}
}

func TestRequireTtdcSchemaAllowsRuntimeInConfiguringState(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	if err := requireTtdcSchema(db, true); err != nil {
		t.Fatalf("configuring schema was rejected for runtime use: %v", err)
	}
	if err := db.Model(&model.TtdcSchemaVersionRecord{}).
		Where("component = ?", model.TtdcSchemaComponent).
		Update("status", model.TtdcSchemaPreparing).Error; err != nil {
		t.Fatal(err)
	}
	if err := requireTtdcSchema(db, true); err == nil {
		t.Fatal("preparing schema was accepted for runtime use")
	}
}

func createTtdcTemplateFixture(t *testing.T, db *gorm.DB) (model.TtdcTemplateGroup, model.TtdcTemplate) {
	t.Helper()
	group := model.TtdcTemplateGroup{GroupCode: "TEST_GROUP", GroupName: "测试模板组", TopicSegment: "TestBattery", Enabled: true}
	if err := db.Create(&group).Error; err != nil {
		t.Fatal(err)
	}
	item := model.TtdcTemplate{
		GroupID: group.ID, TemplateCode: "TEST_TEMPLATE", TemplateName: "测试模板",
		CID2Type: model.TtdcRoleCID42, ProtocolCode: "TEST_PROTOCOL", MachineCID2: model.TtdcRoleCID42,
		ParserProfile: model.TtdcProfileRawFields, Enabled: true,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	return group, item
}

func TestTemplateGroupDetailPreloadsTemplatePoints(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	group, item := createTtdcTemplateFixture(t, db)
	active := model.TtdcTemplatePoint{
		GroupID: item.GroupID, TemplateID: item.ID, SortNo: 1,
		BaseDataID: "active_point", Name: "有效点", DataType: 1, Enabled: true,
	}
	deleted := model.TtdcTemplatePoint{
		GroupID: item.GroupID, TemplateID: item.ID, SortNo: 2,
		BaseDataID: "deleted_point", Name: "已删除点", DataType: 1, Enabled: true,
	}
	if err := db.Create(&active).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&deleted).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&deleted).Error; err != nil {
		t.Fatal(err)
	}

	detail, err := (&TtdcTemplateGroupService{db: db}).Detail(group.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Templates) != 1 {
		t.Fatalf("template count = %d, want 1", len(detail.Templates))
	}
	if len(detail.Templates[0].Points) != 1 || detail.Templates[0].Points[0].BaseDataID != "active_point" {
		t.Fatalf("preloaded points = %+v", detail.Templates[0].Points)
	}
}

func TestTemplateDeletePhysicallyRemovesDefinition(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	_, item := createTtdcTemplateFixture(t, db)
	point := model.TtdcTemplatePoint{GroupID: item.GroupID, TemplateID: item.ID, SortNo: 1, BaseDataID: "data_flag", Name: "数据标识", DataType: 1, Enabled: true}
	topic := model.TtdcTemplateTopic{TemplateID: item.ID, TopicSegment: "LegacyBattery", RawCID2: "42"}
	if err := db.Create(&point).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&TtdcTemplateService{db: db}).Delete(item.ID); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]interface{}{
		"template": &model.TtdcTemplate{}, "point": &model.TtdcTemplatePoint{}, "topic": &model.TtdcTemplateTopic{},
	} {
		var count int64
		if err := db.Unscoped().Model(target).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows remain after physical deletion: %d", name, count)
		}
	}
}

func TestTemplateDeleteWorksWithoutLegacyTopicTable(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	_, item := createTtdcTemplateFixture(t, db)
	point := model.TtdcTemplatePoint{GroupID: item.GroupID, TemplateID: item.ID, SortNo: 1, BaseDataID: "data_flag", Name: "数据标识", DataType: 1, Enabled: true}
	if err := db.Create(&point).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Migrator().DropTable(&model.TtdcTemplateTopic{}); err != nil {
		t.Fatal(err)
	}

	if err := (&TtdcTemplateService{db: db}).Delete(item.ID); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]interface{}{
		"template": &model.TtdcTemplate{}, "point": &model.TtdcTemplatePoint{},
	} {
		var count int64
		if err := db.Unscoped().Model(target).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows remain after deletion without legacy topic table: %d", name, count)
		}
	}
}

func TestReconcileTemplatePointsPhysicallyRemovesMissingPoints(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	_, item := createTtdcTemplateFixture(t, db)
	points := []model.TtdcTemplatePoint{
		{GroupID: item.GroupID, TemplateID: item.ID, SortNo: 1, BaseDataID: "point_1", Name: "点1", DataType: 1, Enabled: true},
		{GroupID: item.GroupID, TemplateID: item.ID, SortNo: 2, BaseDataID: "point_2", Name: "点2", DataType: 1, Enabled: true},
	}
	if err := db.Create(&points).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return reconcileTemplatePoints(tx, item.ID, points[:1], true, false)
	}); err != nil {
		t.Fatal(err)
	}
	var removed int64
	if err := db.Unscoped().Model(&model.TtdcTemplatePoint{}).Where("id = ?", points[1].ID).Count(&removed).Error; err != nil {
		t.Fatal(err)
	}
	if removed != 0 {
		t.Fatal("removed template point was only soft deleted")
	}
}

func TestReconcileTemplatePointsOnlyAllowsUnboundBaseIDChange(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	_, item := createTtdcTemplateFixture(t, db)
	point := model.TtdcTemplatePoint{GroupID: item.GroupID, TemplateID: item.ID, SortNo: 1, BaseDataID: "custom_count", Name: "旧用途", DataType: 1, Enabled: true}
	if err := db.Create(&point).Error; err != nil {
		t.Fatal(err)
	}
	updated := point
	updated.BaseDataID = "point_reassigned"
	updated.Name = "普通数据点"
	if err := db.Transaction(func(tx *gorm.DB) error {
		return reconcileTemplatePoints(tx, item.ID, []model.TtdcTemplatePoint{updated}, false, false)
	}); err == nil || !strings.Contains(err.Error(), "实际通道引用") {
		t.Fatalf("bound point ID change error = %v", err)
	}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return reconcileTemplatePoints(tx, item.ID, []model.TtdcTemplatePoint{updated}, true, false)
	}); err != nil {
		t.Fatal(err)
	}
	var saved model.TtdcTemplatePoint
	if err := db.First(&saved, point.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.BaseDataID != "point_reassigned" || saved.Name != "普通数据点" {
		t.Fatalf("saved point = %+v", saved)
	}
}

func TestReconcileTemplatePointsAllowsBoundCID42ProtocolRepair(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	_, item := createTtdcTemplateFixture(t, db)
	point := model.TtdcTemplatePoint{GroupID: item.GroupID, TemplateID: item.ID, SortNo: 1, BaseDataID: "temperature_count", Name: "单体电池1电压", DataType: 1, Enabled: true}
	if err := db.Create(&point).Error; err != nil {
		t.Fatal(err)
	}
	updated := point
	updated.BaseDataID = "cell01_voltage"
	if err := db.Transaction(func(tx *gorm.DB) error {
		return reconcileTemplatePoints(tx, item.ID, []model.TtdcTemplatePoint{updated}, false, true)
	}); err != nil {
		t.Fatalf("bound CID42 protocol repair was rejected: %v", err)
	}
	var saved model.TtdcTemplatePoint
	if err := db.First(&saved, point.ID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.BaseDataID != "cell01_voltage" {
		t.Fatalf("repaired baseDataID=%q", saved.BaseDataID)
	}
}

func TestTemplateGroupDeletePurgesLegacySoftDeletedDefinitions(t *testing.T) {
	db := newTtdcTemplateTestDB(t)
	group, item := createTtdcTemplateFixture(t, db)
	point := model.TtdcTemplatePoint{GroupID: group.ID, TemplateID: item.ID, SortNo: 1, BaseDataID: "legacy_point", Name: "旧点", DataType: 1, Enabled: true}
	topic := model.TtdcTemplateTopic{TemplateID: item.ID, TopicSegment: "LegacyBattery", RawCID2: "42"}
	if err := db.Create(&point).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&topic).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&point).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Delete(&item).Error; err != nil {
		t.Fatal(err)
	}

	if err := (&TtdcTemplateGroupService{db: db}).Delete(group.ID); err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]interface{}{
		"group": &model.TtdcTemplateGroup{}, "template": &model.TtdcTemplate{},
		"point": &model.TtdcTemplatePoint{}, "topic": &model.TtdcTemplateTopic{},
	} {
		var count int64
		if err := db.Unscoped().Model(target).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows remain after template-group deletion: %d", name, count)
		}
	}
}

func TestNormalizeTemplateRequestRejectsSocAutoCreate(t *testing.T) {
	enabled := true
	_, err := normalizeTemplateRequest(TtdcTemplateRequest{
		TemplateCode:      "SOC_TEST",
		TemplateName:      "SOC test",
		CID2Type:          "42_SOC",
		ProtocolCode:      "CID42_SOC_TEST",
		MachineCID2:       "42_SOC",
		ParserProfile:     "algorithm_only",
		AutoCreateEnabled: &enabled,
		Points: []TtdcTemplatePointRequest{{
			SortNo: 1, BaseDataID: "alg_soc", Name: "SOC", DataType: 1, Enabled: true,
		}},
	})
	if err == nil {
		t.Fatal("normalizeTemplateRequest accepted automatic creation for 42_SOC")
	}
}

func TestNormalizeTemplateRequestGeneratesInternalFields(t *testing.T) {
	enabled := true
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateCode: "OPERATOR_CODE", ProtocolCode: "OPERATOR_PROTOCOL", ParserProfile: model.TtdcProfileCID42Sodium,
		MachineCID2: "42_钠电", TemplateName: "固态电池42", CID2Type: model.TtdcRoleCID42, Enabled: &enabled,
		Points: []TtdcTemplatePointRequest{
			{SortNo: 1, BaseDataID: "wrong_flag", Name: "数据标识", DataType: 1, Address: 0, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 2, BaseDataID: "wrong_position", Name: "电池位置", DataType: 1, Address: 2, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 3, BaseDataID: "wrong_count", Name: "电芯数量", DataType: 1, Address: 4, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 4, BaseDataID: "wrong_voltage", Name: "电芯1电压", DataType: 1, Address: 6, ByteCount: 2, ConvertType: 1, Enabled: true},
			{SortNo: 5, BaseDataID: "wrong_n", Name: "温度数量", DataType: 1, Address: 70, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 6, BaseDataID: "wrong_temperature", Name: "电芯1温度", DataType: 1, Address: 72, ByteCount: 2, ConvertType: 1, Enabled: true},
			{SortNo: 7, BaseDataID: "wrong_p", Name: "自定义遥测量数量", DataType: 1, Address: 132, ByteCount: 1, ConvertType: 3, Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("generic template request rejected: %v", err)
	}
	if item.TemplateCode == "" || item.ProtocolCode == "" {
		t.Fatal("internal template identities were not generated")
	}
	if item.TemplateCode == "OPERATOR_CODE" || item.ProtocolCode == "OPERATOR_PROTOCOL" {
		t.Fatal("operator-entered internal identities were accepted")
	}
	if item.MachineCID2 != model.TtdcRoleCID42 || item.ParserProfile != model.TtdcProfileRawFields {
		t.Fatalf("derived fields = cid2 %q profile %q", item.MachineCID2, item.ParserProfile)
	}
	want := []string{"data_flag", "pack_position", "cell_count", "cell01_voltage", "temperature_count", "cell01_temperature", "custom_count"}
	for index, point := range item.Points {
		if point.BaseDataID != want[index] {
			t.Fatalf("point %d baseDataID=%q, want %q", index+1, point.BaseDataID, want[index])
		}
	}
}

func TestNormalizeTemplateRequestUsesCID42AddressLayout(t *testing.T) {
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "钠电池42", CID2Type: model.TtdcRoleCID42,
		Points: []TtdcTemplatePointRequest{
			{SortNo: 1, BaseDataID: "cell_count", Name: "单体数量", DataType: 1, Address: 4, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 2, BaseDataID: "cell01_voltage", Name: "单体1电压", DataType: 1, Address: 6, ByteCount: 2, ConvertType: 1, Enabled: true},
			{SortNo: 3, BaseDataID: "temperature_count", Name: "温度数量", DataType: 1, Address: 70, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 4, BaseDataID: "cell01_temperature", Name: "温度1", DataType: 1, Address: 72, ByteCount: 2, ConvertType: 1, Enabled: true},
			{SortNo: 5, BaseDataID: "custom_count", Name: "自定义数量", DataType: 1, Address: 132, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 6, BaseDataID: "environment_temperature", Name: "环境温度", DataType: 1, Address: 104, ByteCount: 2, ConvertType: 1, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cell_count", "cell01_voltage", "temperature_count", "cell01_temperature", "custom_count", "environment_temperature"}
	for index, point := range item.Points {
		if point.BaseDataID != want[index] {
			t.Fatalf("point %d baseDataID=%q, want %q", index+1, point.BaseDataID, want[index])
		}
	}
}

func TestNormalizeTemplateRequestGeneratesOpaqueCID42PointForNonProtocolAddress(t *testing.T) {
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "42普通点", CID2Type: model.TtdcRoleCID42,
		Points: []TtdcTemplatePointRequest{
			{SortNo: 1, Name: "温度数量N", DataType: 1, Address: 70, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 2, Name: "自定义遥测量数量P", DataType: 1, Address: 132, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 3, Name: "扩展遥测量", DataType: 1, Address: 140, ByteCount: 2, ConvertType: 1, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Points[0].BaseDataID != "temperature_count" || item.Points[1].BaseDataID != "custom_count" {
		t.Fatalf("protocol point IDs = %q %q", item.Points[0].BaseDataID, item.Points[1].BaseDataID)
	}
	if !strings.HasPrefix(item.Points[2].BaseDataID, "point_") {
		t.Fatalf("ordinary point ID = %q, want opaque ID", item.Points[2].BaseDataID)
	}
}

func TestNormalizeTemplateRequestReportsDuplicatePointInChinese(t *testing.T) {
	_, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "重复点", CID2Type: model.TtdcRoleCID42,
		Points: []TtdcTemplatePointRequest{
			{SortNo: 1, BaseDataID: "custom_count", Name: "旧P", DataType: 1, Address: 100, ByteCount: 1, ConvertType: 3, Enabled: true},
			{SortNo: 2, BaseDataID: "custom_count", Name: "新P", DataType: 1, Address: 132, ByteCount: 1, ConvertType: 3, Enabled: true},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "多个P") {
		t.Fatalf("duplicate error = %v", err)
	}
}

func TestNormalizeTemplateRequestGeneratesOpaqueCID44PointIDs(t *testing.T) {
	bit := 0
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "锂硫电池44", CID2Type: model.TtdcRoleCID44,
		Points: []TtdcTemplatePointRequest{
			{SortNo: 1, Name: "告警1", DataType: 1, Address: 0, ByteCount: 1, ConvertType: 2, BitIndex: &bit, Enabled: true},
			{SortNo: 2, Name: "告警2", DataType: 1, Address: 0, ByteCount: 1, ConvertType: 2, BitIndex: &bit, Enabled: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Points[0].BaseDataID == "" || item.Points[1].BaseDataID == "" || item.Points[0].BaseDataID == item.Points[1].BaseDataID {
		t.Fatalf("generated IDs are invalid: %q %q", item.Points[0].BaseDataID, item.Points[1].BaseDataID)
	}
}

func TestNormalizeTemplateRequestRequiresSocPointID(t *testing.T) {
	_, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "钠电池SOC", CID2Type: model.TtdcRoleCID42SOC,
		Points: []TtdcTemplatePointRequest{{SortNo: 1, Name: "SOC", DataType: 1, AlgorithmKey: "alg_soc", Enabled: true}},
	})
	if err == nil {
		t.Fatal("42_SOC point without an operator ID was accepted")
	}
}

func TestNormalizeTemplateRequestAllowsIncompleteCID42Draft(t *testing.T) {
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "42草稿", CID2Type: model.TtdcRoleCID42,
		Points: []TtdcTemplatePointRequest{{SortNo: 1, Name: "DATA_FLAG", DataType: 1, Address: 0, ByteCount: 1, ConvertType: 3, Enabled: true}},
	})
	if err != nil {
		t.Fatalf("incomplete CID42 draft was rejected: %v", err)
	}
	if len(item.Points) != 1 || item.Points[0].BaseDataID != "data_flag" {
		t.Fatalf("unexpected CID42 draft points: %+v", item.Points)
	}
}

func TestNormalizeTemplateRequestAllowsIncompleteCID42SOCDraft(t *testing.T) {
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "42_SOC草稿", CID2Type: model.TtdcRoleCID42SOC,
		Points: []TtdcTemplatePointRequest{{SortNo: 1, BaseDataID: "alg_soc", Name: "SOC", DataType: 1, Enabled: true}},
	})
	if err != nil {
		t.Fatalf("incomplete 42_SOC draft was rejected: %v", err)
	}
	if len(item.Points) != 1 || item.Points[0].BaseDataID != "alg_soc" {
		t.Fatalf("unexpected 42_SOC draft points: %+v", item.Points)
	}
}

func TestValidateTemplateForActivationRejectsInvalidRawPoint(t *testing.T) {
	address, byteCount, convertType := 0, 1, 2
	item := model.TtdcTemplate{
		CID2Type:      model.TtdcRoleCID44,
		ParserProfile: model.TtdcProfileCID44Common,
		Points: []model.TtdcTemplatePoint{{
			BaseDataID: "alarm", DataType: 1, Address: &address, ByteCount: &byteCount, ConvertType: &convertType,
		}},
	}
	if err := validateTemplateForActivation(item); err == nil {
		t.Fatal("CID44 binary point without bit index was accepted")
	}
}

func TestValidateTemplateForActivationRejectsIncompleteCID42Profile(t *testing.T) {
	address, byteCount, convertType := 0, 1, 3
	item := model.TtdcTemplate{
		CID2Type:      model.TtdcRoleCID42,
		ParserProfile: model.TtdcProfileCID42Sodium,
		Points: []model.TtdcTemplatePoint{{
			BaseDataID: "cell_count", DataType: 1, Address: &address, ByteCount: &byteCount, ConvertType: &convertType,
		}},
	}
	if err := validateTemplateForActivation(item); err == nil {
		t.Fatal("incomplete CID42 profile was accepted")
	}
}

func TestValidateTemplateForActivationRejectsIncompleteCID42SOCDraft(t *testing.T) {
	item := model.TtdcTemplate{
		CID2Type:      model.TtdcRoleCID42SOC,
		ParserProfile: model.TtdcProfileDerivedAlgorithm,
		Points:        []model.TtdcTemplatePoint{{BaseDataID: "alg_soc", DataType: 1}},
	}
	if err := validateTemplateForActivation(item); err == nil {
		t.Fatal("incomplete 42_SOC draft was accepted for runtime activation")
	}
}

func TestCID42PointLimitsFollowTemplateShape(t *testing.T) {
	cellLimit, temperatureLimit, err := model.TtdcCID42PointLimits([]string{
		"cell_count", "temperature_count",
		"cell01_voltage", "cell02_voltage", "cell03_voltage",
		"cell01_temperature", "cell02_temperature",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cellLimit != 3 || temperatureLimit != 2 {
		t.Fatalf("limits = (%d,%d), want (3,2)", cellLimit, temperatureLimit)
	}
}

func TestNormalizeTemplateRequestAcceptsSocPointIDWithoutAlgorithmMapping(t *testing.T) {
	points := make([]TtdcTemplatePointRequest, 0, 24)
	for index := 1; index <= 24; index++ {
		points = append(points, TtdcTemplatePointRequest{SortNo: index, BaseDataID: fmt.Sprintf("alg_cell%02d_soc", index), Name: fmt.Sprintf("%d号单体SOC", index), DataType: 1, Enabled: true})
	}
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{
		GroupID: 1, TemplateName: "24路SOC", CID2Type: model.TtdcRoleCID42SOC,
		Points: points,
	})
	if err != nil {
		t.Fatal(err)
	}
	if item.Points[23].AlgorithmKey != nil {
		t.Fatal("new 42_SOC point unexpectedly persisted a separate algorithm mapping")
	}
	if dataType, ok := model.TtdcAlgorithmDataType("alg_cell24_soc"); !ok || dataType != 1 {
		t.Fatal("dynamic cell SOC result ID was not recognized")
	}
	if _, ok := model.TtdcAlgorithmDataType("alg_cell256_soc"); ok {
		t.Fatal("cell SOC result beyond the one-byte protocol limit was accepted")
	}
}

func TestNormalizeTemplateRequestUsesSodiumCID42Layout(t *testing.T) {
	points := []TtdcTemplatePointRequest{
		{SortNo: 1, Name: "数据标识", DataType: 1, Address: 0, ByteCount: 1, ConvertType: 3, Enabled: true},
		{SortNo: 2, Name: "电池位置", DataType: 1, Address: 2, ByteCount: 1, ConvertType: 3, Enabled: true},
		{SortNo: 3, Name: "电芯数量", DataType: 1, Address: 4, ByteCount: 1, ConvertType: 3, Enabled: true},
	}
	for index := 1; index <= 16; index++ {
		points = append(points, TtdcTemplatePointRequest{SortNo: len(points) + 1, Name: fmt.Sprintf("单体%d电压", index), DataType: 1, Address: 6 + (index-1)*4, ByteCount: 2, ConvertType: 1, Enabled: true})
	}
	points = append(points, TtdcTemplatePointRequest{SortNo: len(points) + 1, Name: "温度数量N", DataType: 1, Address: 70, ByteCount: 1, ConvertType: 3, Enabled: true})
	for index := 1; index <= 8; index++ {
		points = append(points, TtdcTemplatePointRequest{SortNo: len(points) + 1, Name: fmt.Sprintf("温度%d", index), DataType: 1, Address: 72 + (index-1)*4, ByteCount: 2, ConvertType: 1, Enabled: true})
	}
	for _, point := range []struct {
		address int
		name    string
	}{
		{104, "环境温度"}, {108, "MOS温度"}, {112, "电流"}, {116, "电压"}, {120, "剩余容量"}, {124, "总容量"}, {128, "循环次数"}, {132, "自定义遥测量数量P"}, {134, "SOC"}, {138, "SOH"},
	} {
		points = append(points, TtdcTemplatePointRequest{SortNo: len(points) + 1, Name: point.name, DataType: 1, Address: point.address, ByteCount: 2, ConvertType: 1, Enabled: true})
	}
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{GroupID: 1, TemplateName: "钠电16路42", CID2Type: model.TtdcRoleCID42, Points: points})
	if err != nil {
		t.Fatal(err)
	}
	want := map[int]string{0: "data_flag", 2: "cell_count", 3: "cell01_voltage", 18: "cell16_voltage", 19: "temperature_count", 20: "cell01_temperature", 27: "cell08_temperature", 28: "environment_temperature", 35: "custom_count", 36: "pack_soc", 37: "pack_soh"}
	for index, baseDataID := range want {
		if item.Points[index].BaseDataID != baseDataID {
			t.Fatalf("point %d baseDataID=%q, want %q", index+1, item.Points[index].BaseDataID, baseDataID)
		}
	}
	baseDataIDs := make([]string, 0, len(item.Points))
	for _, point := range item.Points {
		baseDataIDs = append(baseDataIDs, point.BaseDataID)
	}
	cellLimit, temperatureLimit, err := model.TtdcCID42PointLimits(baseDataIDs)
	if err != nil || cellLimit != 16 || temperatureLimit != 8 {
		t.Fatalf("sodium CID42 limits=(%d,%d), err=%v", cellLimit, temperatureLimit, err)
	}
}

func TestNormalizeTemplateRequestUsesLithiumSulfurCID42Layout(t *testing.T) {
	points := []TtdcTemplatePointRequest{
		{SortNo: 1, Name: "温度数量N", DataType: 1, Address: 70, ByteCount: 1, ConvertType: 3, Enabled: true},
	}
	for index := 1; index <= 4; index++ {
		points = append(points, TtdcTemplatePointRequest{SortNo: len(points) + 1, Name: fmt.Sprintf("温度%d", index), DataType: 1, Address: 72 + (index-1)*4, ByteCount: 2, ConvertType: 1, Enabled: true})
	}
	points = append(points,
		TtdcTemplatePointRequest{SortNo: len(points) + 1, Name: "环境温度", DataType: 1, Address: 88, ByteCount: 2, ConvertType: 1, Enabled: true},
		TtdcTemplatePointRequest{SortNo: len(points) + 2, Name: "自定义遥测量数量P", DataType: 1, Address: 116, ByteCount: 1, ConvertType: 3, Enabled: true},
		TtdcTemplatePointRequest{SortNo: len(points) + 3, Name: "SOC", DataType: 1, Address: 118, ByteCount: 2, ConvertType: 1, Enabled: true},
	)
	item, err := normalizeTemplateRequest(TtdcTemplateRequest{GroupID: 1, TemplateName: "锂硫16路42", CID2Type: model.TtdcRoleCID42, Points: points})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"temperature_count", "cell01_temperature", "cell02_temperature", "cell03_temperature", "cell04_temperature", "environment_temperature", "custom_count", "pack_soc"}
	for index, point := range item.Points {
		if point.BaseDataID != want[index] {
			t.Fatalf("point %d baseDataID=%q, want %q", index+1, point.BaseDataID, want[index])
		}
	}
}

func TestCID42PersistedAnchorsSurviveDisplayNameChanges(t *testing.T) {
	points, err := assignCID42BaseDataIDs([]TtdcTemplatePointRequest{
		{BaseDataID: "temperature_count", Name: "用户改名后的N", Address: 70},
		{BaseDataID: "", Name: "温度数量", Address: 74},
		{BaseDataID: "custom_count", Name: "用户改名后的P", Address: 132},
	})
	if err != nil {
		t.Fatal(err)
	}
	if points[0].BaseDataID != "temperature_count" || points[2].BaseDataID != "custom_count" {
		t.Fatalf("persisted anchors changed: %+v", points)
	}
}

func TestCID42InvalidPersistedAnchorsFallBackToProtocolMarkers(t *testing.T) {
	points, err := assignCID42BaseDataIDs([]TtdcTemplatePointRequest{
		{BaseDataID: "temperature_count", Name: "单体电池1电压", Address: 6},
		{BaseDataID: "custom_count", Name: "旧的普通点", Address: 102},
		{BaseDataID: "", Name: "N（电芯温度数量）", Address: 70},
		{BaseDataID: "", Name: "P（自定义遥测量数量）", Address: 132},
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"cell01_voltage", "", "temperature_count", "custom_count"}
	for index, point := range points {
		if point.BaseDataID != want[index] {
			t.Fatalf("point %d baseDataID=%q, want %q", index+1, point.BaseDataID, want[index])
		}
	}
}

func TestGenericProfilesAreRoleBasedAndLegacyProfilesRemainCompatible(t *testing.T) {
	for _, role := range []string{model.TtdcRoleCID42, model.TtdcRoleCID44} {
		profile, ok := model.TtdcParserProfileForRole(role)
		if !ok || profile != model.TtdcProfileRawFields {
			t.Fatalf("role %s profile=%q ok=%t", role, profile, ok)
		}
	}
	profile, ok := model.TtdcParserProfileForRole(model.TtdcRoleCID42SOC)
	if !ok || profile != model.TtdcProfileDerivedAlgorithm {
		t.Fatalf("42_SOC profile=%q ok=%t", profile, ok)
	}
	if !model.TtdcProfileCompatibleWithRole(model.TtdcRoleCID42, model.TtdcProfileCID42Sodium) ||
		!model.TtdcProfileCompatibleWithRole(model.TtdcRoleCID44, model.TtdcProfileCID44Common) {
		t.Fatal("legacy role profiles must remain readable")
	}
	if model.TtdcProfileCompatibleWithRole(model.TtdcRoleCID42, model.TtdcProfileCID44Common) {
		t.Fatal("legacy CID44 profile must not be accepted for CID42")
	}
}
