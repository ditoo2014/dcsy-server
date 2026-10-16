package model

import (
	"deal/util/pointid"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// TtdcAlgorithmDataType is the parser contract for 42_SOC algorithm outputs.
func TtdcAlgorithmDataType(key string) (int, bool) {
	key = strings.TrimSpace(key)
	if _, ok := TtdcAlgorithmCellSOCIndex(key); ok {
		return 1, true
	}
	switch key {
	case pointid.RemainingDischargeTime, "alg_soc", "alg_soh", pointid.TemperatureDifference, pointid.VoltageDifference:
		return 1, true
	case pointid.ThermalRunawayRiskLevel:
		return 2, true
	default:
		return 0, false
	}
}

// TtdcAlgorithmResultKey 优先按原版数据点 ID 识别算法输出。
// 历史模板使用自定义 ID 时，沿用 algorithm_key 关联算法结果的回退逻辑。
func TtdcAlgorithmResultKey(baseDataID string, algorithmKey *string) (string, int, bool) {
	baseDataID = strings.TrimSpace(baseDataID)
	if dataType, ok := TtdcAlgorithmDataType(baseDataID); ok {
		return baseDataID, dataType, true
	}
	if algorithmKey == nil {
		return "", 0, false
	}
	key := strings.TrimSpace(*algorithmKey)
	dataType, ok := TtdcAlgorithmDataType(key)
	return key, dataType, ok
}

func TtdcAlgorithmCellSOCIndex(value string) (int, bool) {
	return ttdcIndexedPoint(strings.TrimSpace(value), "alg_cell", "_soc", TtdcMaxIndexedPointCount)
}

func TtdcCID42CellVoltageIndex(value string) (int, bool) {
	return ttdcIndexedPoint(strings.TrimSpace(value), "cell", "_voltage", TtdcMaxIndexedPointCount)
}

func TtdcCID42CellTemperatureIndex(value string) (int, bool) {
	return ttdcIndexedPoint(strings.TrimSpace(value), "cell", "_temperature", TtdcMaxIndexedPointCount)
}

// TtdcParserProfileForRole derives the internal parser mode from the protocol
// role. Battery chemistry is selected by the template group's MQTT segment,
// not by a parser profile.
func TtdcParserProfileForRole(role string) (string, bool) {
	switch strings.ToUpper(strings.TrimSpace(role)) {
	case TtdcRoleCID42, TtdcRoleCID44:
		return TtdcProfileRawFields, true
	case TtdcRoleCID42SOC:
		return TtdcProfileDerivedAlgorithm, true
	default:
		return "", false
	}
}

func TtdcRawProfileCompatible(profile string) bool {
	switch strings.TrimSpace(profile) {
	case TtdcProfileRawFields, TtdcProfileCID42LithiumSulfur, TtdcProfileCID42Sodium, TtdcProfileCID44Common:
		return true
	default:
		return false
	}
}

func TtdcAlgorithmProfileCompatible(profile string) bool {
	switch strings.TrimSpace(profile) {
	case TtdcProfileDerivedAlgorithm, TtdcProfileAlgorithmOnly:
		return true
	default:
		return false
	}
}

func TtdcProfileCompatibleWithRole(role, profile string) bool {
	role = strings.ToUpper(strings.TrimSpace(role))
	profile = strings.TrimSpace(profile)
	switch role {
	case TtdcRoleCID42:
		return profile == TtdcProfileRawFields || profile == TtdcProfileCID42LithiumSulfur || profile == TtdcProfileCID42Sodium
	case TtdcRoleCID44:
		return profile == TtdcProfileRawFields || profile == TtdcProfileCID44Common
	case TtdcRoleCID42SOC:
		return TtdcAlgorithmProfileCompatible(profile)
	default:
		return false
	}
}

// TtdcCID42PointLimits validates explicitly mapped algorithm inputs. Cell
// temperatures are optional because not every battery reports them; when they
// are mapped, a temperature-count point and contiguous indexes are required.
func TtdcCID42PointLimits(baseDataIDs []string) (cellLimit, temperatureLimit int, err error) {
	available := make(map[string]bool, len(baseDataIDs))
	cellIndexes := map[int]bool{}
	temperatureIndexes := map[int]bool{}
	for _, value := range baseDataIDs {
		baseID := strings.TrimSpace(value)
		available[baseID] = true
		if index, ok := TtdcCID42CellVoltageIndex(baseID); ok {
			cellIndexes[index] = true
			if index > cellLimit {
				cellLimit = index
			}
		}
		if index, ok := TtdcCID42CellTemperatureIndex(baseID); ok {
			temperatureIndexes[index] = true
			if index > temperatureLimit {
				temperatureLimit = index
			}
		}
	}
	if !available["cell_count"] {
		return 0, 0, fmt.Errorf("42模板缺少电芯数量用途映射")
	}
	if cellLimit == 0 {
		return 0, 0, fmt.Errorf("42模板至少需要映射一个单体电压点")
	}
	if temperatureLimit > 0 && !available["temperature_count"] {
		return 0, 0, fmt.Errorf("已映射电芯温度点，但缺少温度数量用途映射")
	}
	for index := 1; index <= cellLimit; index++ {
		if !cellIndexes[index] {
			return 0, 0, fmt.Errorf("单体电压用途必须从第1路开始连续编号，缺少第%d路", index)
		}
	}
	for index := 1; index <= temperatureLimit; index++ {
		if !temperatureIndexes[index] {
			return 0, 0, fmt.Errorf("电芯温度用途必须从第1路开始连续编号，缺少第%d路", index)
		}
	}
	return cellLimit, temperatureLimit, nil
}

// TtdcCID42SOCPointLimit validates that configured per-cell SOC result keys
// are continuous, then returns the highest configured cell number.
func TtdcCID42SOCPointLimit(keys []string) (int, error) {
	indexes := make(map[int]bool, len(keys))
	limit := 0
	for _, key := range keys {
		index, ok := TtdcAlgorithmCellSOCIndex(key)
		if !ok {
			continue
		}
		indexes[index] = true
		if index > limit {
			limit = index
		}
	}
	if limit == 0 {
		return 0, fmt.Errorf("42_SOC模板至少需要配置一个单体SOC数据点")
	}
	for index := 1; index <= limit; index++ {
		if !indexes[index] {
			return 0, fmt.Errorf("42_SOC单体SOC数据点必须从第1路开始连续编号，缺少第%d路", index)
		}
	}
	return limit, nil
}

func ttdcIndexedPoint(value, prefix, suffix string, maximum int) (int, bool) {
	if !strings.HasPrefix(value, prefix) || !strings.HasSuffix(value, suffix) {
		return 0, false
	}
	raw := strings.TrimSuffix(strings.TrimPrefix(value, prefix), suffix)
	index, parseErr := strconv.Atoi(raw)
	if parseErr != nil || index < 1 || index > maximum || value != fmt.Sprintf("%s%02d%s", prefix, index, suffix) {
		return 0, false
	}
	return index, true
}

const (
	// M and N are one-byte counts in CID42, so a template cannot describe more
	// than 255 cell-voltage, cell-temperature, or per-cell SOC entries.
	TtdcMaxIndexedPointCount = 255

	TtdcRoleCID42    = "42"
	TtdcRoleCID42SOC = "42_SOC"
	TtdcRoleCID44    = "44"

	TtdcProfileRawFields        = "raw_fields"
	TtdcProfileDerivedAlgorithm = "derived_algorithm"

	// Legacy profiles remain readable so existing template bindings continue
	// to work while new templates use the chemistry-neutral profiles above.
	TtdcProfileCID42LithiumSulfur = "cid42_lithium_sulfur"
	TtdcProfileCID42Sodium        = "cid42_sodium"
	TtdcProfileCID44Common        = "cid44_common"
	TtdcProfileAlgorithmOnly      = "algorithm_only"

	TtdcSchemaComponent = "ttdc_template_group"
	TtdcSchemaVersion   = 2
	TtdcSchemaPreparing = "preparing"
	TtdcSchemaConfig    = "configuring"
	TtdcSchemaReady     = "ready"
	TtdcSchemaFailed    = "failed"
)

// TtdcSchemaRuntimeUsable reports whether the TTDC tables are fully created.
// Configuring is the normal state for installations that create new templates
// in the management UI and do not backfill legacy template/channel data.
func TtdcSchemaRuntimeUsable(status string) bool {
	status = strings.TrimSpace(status)
	return status == TtdcSchemaConfig || status == TtdcSchemaReady
}

// TtdcTemplateGroup owns a battery topic segment and one template per role.
type TtdcTemplateGroup struct {
	ID                  uint           `json:"id" gorm:"primaryKey,column:id"`
	GroupCode           string         `json:"groupCode" gorm:"column:group_code;size:64;not null;uniqueIndex"`
	GroupName           string         `json:"groupName" gorm:"column:group_name;size:128;not null"`
	TopicSegment        string         `json:"topicSegment" gorm:"column:topic_segment;size:64;not null;index"`
	Enabled             bool           `json:"enabled" gorm:"column:enabled;not null;default:true"`
	RuntimeMatchEnabled bool           `json:"runtimeMatchEnabled" gorm:"column:runtime_match_enabled;not null;default:false"`
	AutoCreateEnabled   bool           `json:"autoCreateEnabled" gorm:"column:auto_create_enabled;not null;default:false"`
	Version             int            `json:"version" gorm:"column:version;not null;default:1"`
	Description         string         `json:"description" gorm:"column:description;size:512"`
	CreatedBy           int            `json:"createdBy" gorm:"column:created_by;not null;default:0"`
	UpdatedBy           int            `json:"updatedBy" gorm:"column:updated_by;not null;default:0"`
	CreatedAt           time.Time      `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt           time.Time      `json:"updatedAt" gorm:"column:updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index"`
	Templates           []TtdcTemplate `json:"templates,omitempty" gorm:"foreignKey:GroupID;references:ID"`
	TemplateCount       int64          `json:"templateCount" gorm:"-"`
	Complete            bool           `json:"complete" gorm:"-"`
}

func (TtdcTemplateGroup) TableName() string { return "ttdc_template_group" }

type TtdcTemplate struct {
	ID                uint                `json:"id" gorm:"primaryKey,column:id"`
	GroupID           uint                `json:"groupId" gorm:"column:group_id;not null;index"`
	TemplateCode      string              `json:"templateCode" gorm:"column:template_code;size:64;not null;uniqueIndex"`
	TemplateName      string              `json:"templateName" gorm:"column:template_name;size:128;not null"`
	CID2Type          string              `json:"cid2Type" gorm:"column:cid2_type;size:32;not null;index"`
	ProtocolCode      string              `json:"protocolCode" gorm:"column:protocol_code;size:64;not null"`
	MachineCID2       string              `json:"machineCid2" gorm:"column:machine_cid2;size:64;not null"`
	ParserProfile     string              `json:"parserProfile" gorm:"column:parser_profile;size:64;not null"`
	Enabled           bool                `json:"enabled" gorm:"column:enabled;not null;default:true"`
	AutoCreateEnabled bool                `json:"autoCreateEnabled" gorm:"column:auto_create_enabled;not null;default:false"`
	Version           int                 `json:"version" gorm:"column:version;not null;default:1"`
	Description       string              `json:"description" gorm:"column:description;size:512"`
	CreatedBy         int                 `json:"createdBy" gorm:"column:created_by;not null;default:0"`
	UpdatedBy         int                 `json:"updatedBy" gorm:"column:updated_by;not null;default:0"`
	CreatedAt         time.Time           `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt         time.Time           `json:"updatedAt" gorm:"column:updated_at"`
	DeletedAt         gorm.DeletedAt      `json:"-" gorm:"column:deleted_at;index"`
	Topics            []TtdcTemplateTopic `json:"topics,omitempty" gorm:"foreignKey:TemplateID;references:ID"`
	Points            []TtdcTemplatePoint `json:"points" gorm:"foreignKey:TemplateID;references:ID"`
	PointCount        int64               `json:"pointCount" gorm:"-"`
}

func (TtdcTemplate) TableName() string { return "ttdc_template" }

// TtdcTemplateTopic remains readable while old templates are migrated.
type TtdcTemplateTopic struct {
	ID                  uint      `json:"id" gorm:"primaryKey,column:id"`
	TemplateID          uint      `json:"templateId" gorm:"column:template_id;not null;index"`
	TopicSegment        string    `json:"topicSegment" gorm:"column:topic_segment;size:64;not null;index"`
	RawCID2             string    `json:"rawCid2" gorm:"column:raw_cid2;size:16;not null"`
	RuntimeMatchEnabled bool      `json:"runtimeMatchEnabled" gorm:"column:runtime_match_enabled;not null;default:false"`
	SortNo              int       `json:"sortNo" gorm:"column:sort_no;not null;default:0"`
	CreatedAt           time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt           time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (TtdcTemplateTopic) TableName() string { return "ttdc_template_topic" }

type TtdcTemplatePoint struct {
	ID           uint           `json:"id" gorm:"primaryKey,column:id"`
	GroupID      uint           `json:"groupId" gorm:"column:group_id;not null;index"`
	TemplateID   uint           `json:"templateId" gorm:"column:template_id;not null;index"`
	SortNo       int            `json:"sortNo" gorm:"column:sort_no;not null"`
	BaseDataID   string         `json:"baseDataId" gorm:"column:base_data_id;size:48;not null"`
	Name         string         `json:"name" gorm:"column:name;size:128;not null"`
	DataType     int            `json:"dataType" gorm:"column:data_type;type:smallint;not null;default:1"`
	Address      *int           `json:"address" gorm:"column:address"`
	ByteCount    *int           `json:"byteCount" gorm:"column:byte_count"`
	ConvertType  *int           `json:"convertType" gorm:"column:convert_type"`
	BitIndex     *int           `json:"bitIndex" gorm:"column:bit_index"`
	AlgorithmKey *string        `json:"algorithmKey" gorm:"column:algorithm_key;size:64"`
	Transport    string         `json:"transport" gorm:"column:transport;size:128"`
	Formula      string         `json:"formula" gorm:"column:formula;size:256;not null;default:'x'"`
	Unformula    string         `json:"unformula" gorm:"column:unformula;size:256;not null;default:'x'"`
	Enabled      bool           `json:"enabled" gorm:"column:enabled;not null;default:true"`
	Description  string         `json:"description" gorm:"column:description;size:512"`
	CreatedAt    time.Time      `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt    time.Time      `json:"updatedAt" gorm:"column:updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"column:deleted_at;index"`
}

func (TtdcTemplatePoint) TableName() string { return "ttdc_template_point" }

type GatewayTtdcAddressGroupBinding struct {
	ID              uint      `json:"id" gorm:"primaryKey,column:id"`
	GatewayID       string    `json:"gatewayId" gorm:"column:gateway_id;size:64;not null;uniqueIndex:uk_ttdc_address_group,priority:1"`
	ModuleADR       string    `json:"moduleAdr" gorm:"column:module_adr;size:2;not null;uniqueIndex:uk_ttdc_address_group,priority:2"`
	TemplateGroupID uint      `json:"templateGroupId" gorm:"column:template_group_id;not null;index"`
	GroupCode       string    `json:"groupCode" gorm:"column:group_code;size:64;not null"`
	TopicSegment    string    `json:"topicSegment" gorm:"column:topic_segment;size:64;not null"`
	ModuleSOI       string    `json:"moduleSoi" gorm:"column:module_soi;size:8;not null"`
	ModuleVER       string    `json:"moduleVer" gorm:"column:module_ver;size:16;not null"`
	ModuleCID1      string    `json:"moduleCid1" gorm:"column:module_cid1;size:16;not null"`
	BindSource      string    `json:"bindSource" gorm:"column:bind_source;size:16;not null"`
	BindingStatus   string    `json:"bindingStatus" gorm:"column:binding_status;size:16;not null"`
	CreatedAt       time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (GatewayTtdcAddressGroupBinding) TableName() string {
	return "gateway_ttdc_address_group_binding"
}

type GatewayTtdcTemplateBinding struct {
	ID              uint      `json:"id" gorm:"primaryKey,column:id"`
	ModuleID        uint      `json:"moduleId" gorm:"column:module_id;not null;uniqueIndex"`
	GatewayID       string    `json:"gatewayId" gorm:"column:gateway_id;size:64;not null;index;uniqueIndex:uk_ttdc_channel_identity,priority:1"`
	TemplateGroupID uint      `json:"templateGroupId" gorm:"column:template_group_id;not null;index"`
	TemplateID      uint      `json:"templateId" gorm:"column:template_id;not null;index"`
	TemplateCode    string    `json:"templateCode" gorm:"column:template_code;size:64;not null"`
	TemplateVersion int       `json:"templateVersion" gorm:"column:template_version;not null"`
	ProtocolCode    string    `json:"protocolCode" gorm:"column:protocol_code;size:64;not null"`
	ParserProfile   string    `json:"parserProfile" gorm:"column:parser_profile;size:64;not null"`
	ModuleSOI       string    `json:"moduleSoi" gorm:"column:module_soi;size:8;not null"`
	ModuleVER       string    `json:"moduleVer" gorm:"column:module_ver;size:16;not null"`
	ModuleADR       string    `json:"moduleAdr" gorm:"column:module_adr;size:16;not null;uniqueIndex:uk_ttdc_channel_identity,priority:2"`
	ModuleCID1      string    `json:"moduleCid1" gorm:"column:module_cid1;size:16;not null"`
	NormalizedCID2  string    `json:"normalizedCid2" gorm:"column:normalized_cid2;size:32;not null;uniqueIndex:uk_ttdc_channel_identity,priority:3"`
	CreateSource    string    `json:"createSource" gorm:"column:create_source;size:16;not null"`
	CreatedAt       time.Time `json:"createdAt" gorm:"column:created_at"`
	UpdatedAt       time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (GatewayTtdcTemplateBinding) TableName() string { return "gateway_ttdc_template_binding" }

type TtdcConfigRevision struct {
	ID         uint64    `json:"id" gorm:"primaryKey,column:id"`
	ConfigType string    `json:"configType" gorm:"column:config_type;size:32;not null;index"`
	TargetKey  string    `json:"targetKey" gorm:"column:target_key;size:128;not null;index"`
	Operation  string    `json:"operation" gorm:"column:operation;size:16;not null"`
	CreatedAt  time.Time `json:"createdAt" gorm:"column:created_at;index"`
}

func (TtdcConfigRevision) TableName() string { return "ttdc_config_revision" }

type TtdcSchemaVersionRecord struct {
	ID        uint      `json:"id" gorm:"primaryKey,column:id"`
	Component string    `json:"component" gorm:"column:component;size:64;not null;uniqueIndex"`
	Version   int       `json:"version" gorm:"column:version;not null"`
	Status    string    `json:"status" gorm:"column:status;size:16;not null"`
	Checksum  string    `json:"checksum" gorm:"column:checksum;size:64"`
	AppliedAt time.Time `json:"appliedAt" gorm:"column:applied_at"`
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at"`
}

func (TtdcSchemaVersionRecord) TableName() string { return "ttdc_schema_version" }
