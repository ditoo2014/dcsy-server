package dao

import (
	"dicloud/internal/tolink/model"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestTtdcTemplateSchemaReadyAllowsConfiguringState(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err = db.AutoMigrate(&model.TtdcSchemaVersionRecord{}); err != nil {
		t.Fatal(err)
	}
	state := model.TtdcSchemaVersionRecord{Component: model.TtdcSchemaComponent, Version: model.TtdcSchemaVersion, Status: model.TtdcSchemaConfig}
	if err = db.Create(&state).Error; err != nil {
		t.Fatal(err)
	}
	if !ttdcTemplateSchemaReady(db) {
		t.Fatal("configuring schema was rejected for manual template channel creation")
	}
	if err = db.Model(&state).Update("status", model.TtdcSchemaFailed).Error; err != nil {
		t.Fatal(err)
	}
	if ttdcTemplateSchemaReady(db) {
		t.Fatal("failed schema was accepted for manual template channel creation")
	}
}

func TestApplyTemplatePointMetadataRequiresCompleteEnabledPointSet(t *testing.T) {
	algorithmKey := "alg_soc"
	template := model.TtdcTemplate{Points: []model.TtdcTemplatePoint{
		{BaseDataID: "pack_soc", Enabled: true, AlgorithmKey: &algorithmKey},
		{BaseDataID: "disabled_point", Enabled: false},
	}}
	module := model.GatewayModuleTtdc{
		TtdcConfig:  model.ModuleTtdcConfig{ModuleAdr: "01"},
		ModuleDatas: []model.ModuleTtdcData{{DataId: "01_pack_soc"}},
	}
	if err := applyTemplatePointMetadata(&module, template); err != nil {
		t.Fatalf("complete enabled point set rejected: %v", err)
	}
	if module.ModuleDatas[0].AlgorithmKey == nil || *module.ModuleDatas[0].AlgorithmKey != algorithmKey {
		t.Fatal("template algorithm key was not copied")
	}

	module.ModuleDatas = nil
	if err := applyTemplatePointMetadata(&module, template); err == nil {
		t.Fatal("missing enabled template point was accepted")
	}
}

func TestApplyTemplatePointMetadataRejectsWrongPrefixAndDuplicates(t *testing.T) {
	template := model.TtdcTemplate{Points: []model.TtdcTemplatePoint{{BaseDataID: "pack_soc", Enabled: true}}}
	module := model.GatewayModuleTtdc{
		TtdcConfig:  model.ModuleTtdcConfig{ModuleAdr: "01"},
		ModuleDatas: []model.ModuleTtdcData{{DataId: "02_pack_soc"}},
	}
	if err := applyTemplatePointMetadata(&module, template); err == nil {
		t.Fatal("point with another ADR prefix was accepted")
	}

	module.ModuleDatas = []model.ModuleTtdcData{{DataId: "01_pack_soc"}, {DataId: "01_pack_soc"}}
	if err := applyTemplatePointMetadata(&module, template); err == nil {
		t.Fatal("duplicate template point was accepted")
	}
}

func TestDeleteModuleByGatewayIDReleasesAddressBindings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.GatewayModuleTtdc{}, &model.ModuleTtdcConfig{}, &model.ModuleTtdcData{},
		&model.GatewayData{}, &model.GatewayTtdcTemplateBinding{},
		&model.GatewayTtdcAddressGroupBinding{}, &model.TtdcConfigRevision{},
	); err != nil {
		t.Fatal(err)
	}
	const gatewayID = "860000000000002"
	address := model.GatewayTtdcAddressGroupBinding{
		GatewayID: gatewayID, ModuleADR: "01", TemplateGroupID: 9,
		GroupCode: "SODIUM", TopicSegment: "SodiumBattery", ModuleSOI: "~",
		ModuleVER: "20", ModuleCID1: "4A", BindSource: "auto", BindingStatus: "active",
	}
	if err := db.Create(&address).Error; err != nil {
		t.Fatal(err)
	}
	if err := (&ModuleTtdcDAO{db: db}).DeleteModuleByGatewayId(gatewayID); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&model.GatewayTtdcAddressGroupBinding{}).Where("gateway_id = ?", gatewayID).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("ADR bindings remain after gateway deletion: %d", count)
	}
	var revision model.TtdcConfigRevision
	if err := db.Where("config_type = ? AND target_key = ?", "address_group", gatewayID+":01").First(&revision).Error; err != nil {
		t.Fatal(err)
	}
	if revision.Operation != "delete" {
		t.Fatalf("address revision operation=%q", revision.Operation)
	}
}

func TestEnsureManualAddressGroupBindingReclaimsInactiveLegacyRow(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(
		&model.GatewayTtdcAddressGroupBinding{}, &model.GatewayTtdcTemplateBinding{},
		&model.TtdcConfigRevision{},
	); err != nil {
		t.Fatal(err)
	}
	const gatewayID = "860000000000003"
	legacy := model.GatewayTtdcAddressGroupBinding{
		GatewayID: gatewayID, ModuleADR: "01", TemplateGroupID: 1,
		GroupCode: "SODIUM", TopicSegment: "SodiumBattery", ModuleSOI: "~",
		ModuleVER: "20", ModuleCID1: "4A", BindSource: "auto", BindingStatus: "inactive",
	}
	if err := db.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	group := model.TtdcTemplateGroup{ID: 2, GroupCode: "LITHIUM_SULFUR", TopicSegment: "LithiumSulfurBattery"}
	module := model.GatewayModuleTtdc{GatewayId: gatewayID, TtdcConfig: model.ModuleTtdcConfig{ModuleAdr: "01", ModuleSoi: "~", ModuleVer: "20", ModuleCid1: "4A"}}
	if err := db.Transaction(func(tx *gorm.DB) error {
		return ensureManualAddressGroupBinding(tx, module, group)
	}); err != nil {
		t.Fatal(err)
	}
	var saved model.GatewayTtdcAddressGroupBinding
	if err := db.Where("gateway_id = ? AND module_adr = ?", gatewayID, "01").First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.BindingStatus != "active" || saved.TemplateGroupID != group.ID || saved.TopicSegment != group.TopicSegment || saved.BindSource != "manual" {
		t.Fatalf("reclaimed binding = %+v", saved)
	}
}
