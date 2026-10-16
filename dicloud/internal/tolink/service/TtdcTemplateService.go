package service

import (
	"dicloud/internal/tolink/model"
	"dicloud/util/opengauss"
	"dicloud/util/pointid"
	"dicloud/util/response"
	"errors"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ttdcTopicSegmentPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]{0,63}$`)
	ttdcBaseDataIDPattern   = regexp.MustCompile(`^[a-z][a-z0-9_]{0,47}$`)
	ttdcMachineCodePattern  = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)
)

type TtdcTemplatePageRequest struct {
	PageNum  int    `json:"pageNum" form:"pageNum"`
	PageSize int    `json:"pageSize" form:"pageSize"`
	Keyword  string `json:"keyword" form:"keyword"`
	GroupID  uint   `json:"groupId" form:"groupId"`
}

type TtdcTemplatePointRequest struct {
	ID           uint   `json:"id"`
	SortNo       int    `json:"sortNo"`
	BaseDataID   string `json:"baseDataId"`
	Name         string `json:"name"`
	DataType     int    `json:"dataType"`
	Address      int    `json:"address"`
	ByteCount    int    `json:"byteCount"`
	ConvertType  int    `json:"convertType"`
	BitIndex     *int   `json:"bitIndex"`
	AlgorithmKey string `json:"algorithmKey"`
	Transport    string `json:"transport"`
	Formula      string `json:"formula"`
	Unformula    string `json:"unformula"`
	Enabled      bool   `json:"enabled"`
	Description  string `json:"description"`
}

// Kept for import compatibility. Topic matching now belongs to a group.
type TtdcTemplateTopicRequest struct {
	ID                  uint   `json:"id"`
	TopicSegment        string `json:"topicSegment"`
	RawCID2             string `json:"rawCid2"`
	RuntimeMatchEnabled bool   `json:"runtimeMatchEnabled"`
	SortNo              int    `json:"sortNo"`
}

type TtdcTemplateRequest struct {
	ID                    uint                       `json:"id"`
	GroupID               uint                       `json:"groupId"`
	TemplateCode          string                     `json:"templateCode"`
	TemplateName          string                     `json:"templateName"`
	CID2Type              string                     `json:"cid2Type"`
	ProtocolCode          string                     `json:"protocolCode"`
	MachineCID2           string                     `json:"machineCid2"`
	ParserProfile         string                     `json:"parserProfile"`
	Enabled               *bool                      `json:"enabled"`
	AutoCreateEnabled     *bool                      `json:"autoCreateEnabled"`
	Version               int                        `json:"version"`
	Description           string                     `json:"description"`
	ConfirmRuntimeChanges bool                       `json:"confirmRuntimeChanges"`
	Topics                []TtdcTemplateTopicRequest `json:"topics"`
	Points                []TtdcTemplatePointRequest `json:"points"`
}

type TtdcTemplateGroupRequest struct {
	ID                    uint   `json:"id"`
	GroupCode             string `json:"groupCode"`
	GroupName             string `json:"groupName"`
	TopicSegment          string `json:"topicSegment"`
	Enabled               *bool  `json:"enabled"`
	RuntimeMatchEnabled   *bool  `json:"runtimeMatchEnabled"`
	AutoCreateEnabled     *bool  `json:"autoCreateEnabled"`
	Version               int    `json:"version"`
	Description           string `json:"description"`
	ConfirmRuntimeChanges bool   `json:"confirmRuntimeChanges"`
}

type AlgorithmKeyOption struct {
	Key      string `json:"key"`
	DataType int    `json:"dataType"`
	Name     string `json:"name"`
}

func AlgorithmKeyOptions() []AlgorithmKeyOption {
	result := make([]AlgorithmKeyOption, 0, model.TtdcMaxIndexedPointCount+6)
	for i := 1; i <= model.TtdcMaxIndexedPointCount; i++ {
		result = append(result, AlgorithmKeyOption{Key: fmt.Sprintf("alg_cell%02d_soc", i), DataType: 1, Name: fmt.Sprintf("%d号单体SOC", i)})
	}
	return append(result,
		AlgorithmKeyOption{Key: pointid.RemainingDischargeTime, DataType: 1, Name: "放电时长"},
		AlgorithmKeyOption{Key: "alg_soc", DataType: 1, Name: "SOC"},
		AlgorithmKeyOption{Key: "alg_soh", DataType: 1, Name: "SOH"},
		AlgorithmKeyOption{Key: pointid.TemperatureDifference, DataType: 1, Name: "温差"},
		AlgorithmKeyOption{Key: pointid.VoltageDifference, DataType: 1, Name: "压差"},
		AlgorithmKeyOption{Key: pointid.ThermalRunawayRiskLevel, DataType: 2, Name: "热失控风险等级"},
	)
}

func profiles() []string {
	return []string{model.TtdcProfileRawFields, model.TtdcProfileDerivedAlgorithm}
}

func normalizeTemplateRequest(req TtdcTemplateRequest) (model.TtdcTemplate, error) {
	role := strings.ToUpper(strings.TrimSpace(req.CID2Type))
	profile, _ := model.TtdcParserProfileForRole(role)
	templateCode := generatedTemplateMachineCode("TTDC_TEMPLATE")
	protocolCode := generatedTemplateMachineCode("TTDC_PROTOCOL")
	item := model.TtdcTemplate{
		ID: req.ID, GroupID: req.GroupID, TemplateCode: templateCode,
		TemplateName: strings.TrimSpace(req.TemplateName), CID2Type: role,
		ProtocolCode: protocolCode, MachineCID2: role,
		ParserProfile: profile, Description: strings.TrimSpace(req.Description), Version: req.Version,
	}
	if item.GroupID == 0 {
		return item, errors.New("请选择模板组")
	}
	if !ttdcMachineCodePattern.MatchString(item.TemplateCode) || !ttdcMachineCodePattern.MatchString(item.ProtocolCode) {
		return item, errors.New("模板编码和协议编码必须是大写机器编码")
	}
	if item.TemplateName == "" || len(item.TemplateName) > 128 || item.MachineCID2 == "" {
		return item, errors.New("模板名称和机器控制标识码不能为空")
	}
	if item.CID2Type != model.TtdcRoleCID42 && item.CID2Type != model.TtdcRoleCID42SOC && item.CID2Type != model.TtdcRoleCID44 {
		return item, errors.New("模板角色只支持42、42_SOC或44")
	}
	if !model.TtdcProfileCompatibleWithRole(item.CID2Type, item.ParserProfile) {
		return item, errors.New("模板角色与解析Profile不匹配")
	}
	if len(req.Topics) > 0 {
		return item, errors.New("主题段必须配置在模板组，模板不能单独配置主题")
	}
	if req.AutoCreateEnabled != nil && *req.AutoCreateEnabled {
		return item, errors.New("自动创建由模板组统一控制")
	}
	item.Enabled = true
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	item.AutoCreateEnabled = false
	if item.Version <= 0 {
		item.Version = 1
	}
	if item.CID2Type == model.TtdcRoleCID42 {
		var layoutErr error
		req.Points, layoutErr = assignCID42BaseDataIDs(req.Points)
		if layoutErr != nil {
			return item, layoutErr
		}
	}

	type seenPoint struct {
		row  int
		name string
	}
	seenBase, seenSort, seenAlgorithm := map[string]seenPoint{}, map[int]seenPoint{}, map[string]bool{}
	for index, requestPoint := range req.Points {
		baseID := strings.TrimSpace(requestPoint.BaseDataID)
		name := strings.TrimSpace(requestPoint.Name)
		if name == "" {
			return item, fmt.Errorf("第%d个数据点名称不合法", index+1)
		}
		if item.CID2Type != model.TtdcRoleCID42SOC && baseID == "" {
			baseID = generatedRawPointBaseDataID()
		}
		if !ttdcBaseDataIDPattern.MatchString(baseID) {
			if item.CID2Type == model.TtdcRoleCID42SOC {
				return item, fmt.Errorf("第%d个42_SOC数据点ID不合法", index+1)
			}
			return item, fmt.Errorf("第%d个数据点系统ID生成失败", index+1)
		}
		if requestPoint.SortNo <= 0 {
			return item, fmt.Errorf("第%d个数据点“%s”的排序号必须大于0", index+1, name)
		}
		if previous, duplicated := seenBase[baseID]; duplicated {
			return item, fmt.Errorf("第%d个数据点“%s”的系统ID“%s”与第%d个数据点“%s”重复", index+1, name, baseID, previous.row, previous.name)
		}
		if previous, duplicated := seenSort[requestPoint.SortNo]; duplicated {
			return item, fmt.Errorf("第%d个数据点“%s”的排序号%d与第%d个数据点“%s”重复", index+1, name, requestPoint.SortNo, previous.row, previous.name)
		}
		if requestPoint.DataType != 1 && requestPoint.DataType != 2 {
			return item, fmt.Errorf("第%d个数据点类型不合法", index+1)
		}
		point := model.TtdcTemplatePoint{ID: requestPoint.ID, GroupID: item.GroupID, SortNo: requestPoint.SortNo, BaseDataID: baseID, Name: name, DataType: requestPoint.DataType, Enabled: requestPoint.Enabled, Description: strings.TrimSpace(requestPoint.Description)}
		if item.CID2Type == model.TtdcRoleCID42SOC {
			// 沿用原版算法 ID 校验，并兼容历史模板通过算法键关联输出。
			var algorithmKey *string
			if value := strings.TrimSpace(requestPoint.AlgorithmKey); value != "" {
				if _, ok := model.TtdcAlgorithmDataType(value); !ok {
					return item, fmt.Errorf("第%d个算法键不合法", index+1)
				}
				algorithmKey = stringPointer(value)
			}
			key, expected, ok := model.TtdcAlgorithmResultKey(baseID, algorithmKey)
			if !ok || seenAlgorithm[key] || expected != requestPoint.DataType {
				return item, fmt.Errorf("第%d个42_SOC数据点ID不受支持、重复或数据类型不匹配", index+1)
			}
			seenAlgorithm[key] = true
			point.AlgorithmKey = algorithmKey
			point.Transport, point.Formula, point.Unformula = "-", "x", "x"
		} else {
			if requestPoint.Address < 0 || requestPoint.Address%2 != 0 || requestPoint.ByteCount <= 0 || requestPoint.ConvertType < 1 || requestPoint.ConvertType > 3 {
				return item, fmt.Errorf("第%d个数据点地址、字节数或转换类型不合法", index+1)
			}
			if item.CID2Type == model.TtdcRoleCID42 && requestPoint.BitIndex != nil {
				return item, fmt.Errorf("第%d个CID42数据点不能配置位索引", index+1)
			}
			if requestPoint.ConvertType == 2 && requestPoint.BitIndex == nil {
				return item, fmt.Errorf("第%d个二进制数据点必须配置位索引", index+1)
			}
			if requestPoint.ConvertType != 2 && requestPoint.BitIndex != nil {
				return item, fmt.Errorf("第%d个十进制数据点不能配置位索引", index+1)
			}
			if requestPoint.DataType != 1 {
				return item, fmt.Errorf("第%d个原始报文数据点必须是数字类型", index+1)
			}
			if requestPoint.BitIndex != nil && (*requestPoint.BitIndex < 0 || *requestPoint.BitIndex >= requestPoint.ByteCount*8) {
				return item, fmt.Errorf("第%d个数据点位索引越界", index+1)
			}
			point.Address, point.ByteCount, point.ConvertType = intPointer(requestPoint.Address), intPointer(requestPoint.ByteCount), intPointer(requestPoint.ConvertType)
			point.BitIndex = requestPoint.BitIndex
			point.Transport = fmt.Sprintf("%d %d %d", requestPoint.Address, requestPoint.ByteCount, requestPoint.ConvertType)
			if requestPoint.BitIndex != nil {
				point.Transport += fmt.Sprintf(" %d", *requestPoint.BitIndex)
			}
			point.Formula, point.Unformula = defaultFormula(requestPoint.Formula), defaultFormula(requestPoint.Unformula)
		}
		seenBase[baseID] = seenPoint{row: index + 1, name: name}
		seenSort[requestPoint.SortNo] = seenPoint{row: index + 1, name: name}
		item.Points = append(item.Points, point)
	}
	if len(item.Points) == 0 {
		return item, errors.New("数据点列表不能为空")
	}
	return item, nil
}

func intPointer(value int) *int          { return &value }
func stringPointer(value string) *string { return &value }
func defaultFormula(value string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return "x"
}

func generatedTemplateMachineCode(prefix string) string {
	return prefix + "_" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
}

func generatedRawPointBaseDataID() string {
	return "point_" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

var cid42ReservedBaseDataIDs = map[string]bool{
	"data_flag": true, "pack_position": true, "cell_count": true,
	"temperature_count": true, "environment_temperature": true,
	"mos_temperature": true, "battery_current": true,
	"battery_voltage": true, "remaining_capacity": true,
	"total_capacity": true, "cycle_count": true, "custom_count": true,
	"pack_soc": true, "pack_soh": true,
}

// assignCID42BaseDataIDs derives parser-facing IDs from the TTDC M/N/P layout.
// Names are used only to locate N and P when a template is first created; after
// that their persisted IDs keep the anchors stable even if display names change.
func assignCID42BaseDataIDs(points []TtdcTemplatePointRequest) ([]TtdcTemplatePointRequest, error) {
	result := append([]TtdcTemplatePointRequest(nil), points...)
	nIndex, pIndex, err := resolveCID42Anchors(result)
	if err != nil {
		return nil, err
	}

	// A template can be saved as an incomplete draft. Derive whichever part of
	// the protocol layout is already present, and defer the full M/N/P check to
	// runtime activation. This keeps editing incremental without accepting an
	// invalid active template.
	nAddress, pAddress := -1, -1
	if nIndex >= 0 {
		nAddress = result[nIndex].Address
		if nAddress < 10 || (nAddress-6)%4 != 0 {
			return nil, fmt.Errorf("42模板N数据点地址%d不合法，应位于单体电压区之后", nAddress)
		}
	}
	if pIndex >= 0 {
		pAddress = result[pIndex].Address
		if pAddress < 28 {
			return nil, fmt.Errorf("42模板P数据点地址%d不合法，无法容纳协议固定字段", pAddress)
		}
	}

	cellCount, temperatureCount := 0, 0
	if nAddress >= 0 {
		cellCount = (nAddress - 6) / 4
	}
	if nAddress >= 0 && pAddress >= 0 {
		temperatureEnd := pAddress - 28
		if temperatureEnd < nAddress+2 || (temperatureEnd-(nAddress+2))%4 != 0 {
			return nil, fmt.Errorf("42模板P数据点地址%d与N数据点地址%d不符合协议布局", pAddress, nAddress)
		}
		temperatureCount = (temperatureEnd - (nAddress + 2)) / 4
	}
	if cellCount > model.TtdcMaxIndexedPointCount || temperatureCount > model.TtdcMaxIndexedPointCount {
		return nil, errors.New("42模板根据N/P地址计算出的电芯或温度数量超出协议范围")
	}

	expected := map[int]string{
		0: "data_flag", 2: "pack_position", 4: "cell_count",
	}
	if nAddress >= 0 {
		expected[nAddress] = "temperature_count"
		for index := 1; index <= cellCount; index++ {
			expected[6+(index-1)*4] = fmt.Sprintf("cell%02d_voltage", index)
		}
	}
	if pAddress >= 0 {
		expected[pAddress-28] = "environment_temperature"
		expected[pAddress-24] = "mos_temperature"
		expected[pAddress-20] = "battery_current"
		expected[pAddress-16] = "battery_voltage"
		expected[pAddress-12] = "remaining_capacity"
		expected[pAddress-8] = "total_capacity"
		expected[pAddress-4] = "cycle_count"
		expected[pAddress] = "custom_count"
		expected[pAddress+2] = "pack_soc"
		expected[pAddress+6] = "pack_soh"
	}
	if nAddress >= 0 && pAddress >= 0 {
		for index := 1; index <= temperatureCount; index++ {
			expected[nAddress+2+(index-1)*4] = fmt.Sprintf("cell%02d_temperature", index)
		}
	}
	completeLayout := nAddress >= 0 && pAddress >= 0

	assigned := make(map[string]bool, len(expected))
	for index := range result {
		point := &result[index]
		baseID, semantic := expected[point.Address]
		if semantic && !assigned[baseID] {
			point.BaseDataID = baseID
			assigned[baseID] = true
			continue
		}
		current := strings.TrimSpace(point.BaseDataID)
		if completeLayout && isCID42ReservedBaseDataID(current) {
			point.BaseDataID = ""
		}
	}
	return result, nil
}

type cid42AnchorCandidates struct {
	named     int
	persisted int
}

func resolveCID42Anchors(points []TtdcTemplatePointRequest) (int, int, error) {
	nCandidates, err := findCID42AnchorCandidates(points, "temperature_count", isCID42NName)
	if err != nil {
		return -1, -1, err
	}
	pCandidates, err := findCID42AnchorCandidates(points, "custom_count", isCID42PName)
	if err != nil {
		return -1, -1, err
	}
	nChoices := rankedCID42AnchorChoices(points, nCandidates, "temperature_count")
	pChoices := rankedCID42AnchorChoices(points, pCandidates, "custom_count")
	if len(nChoices) == 0 {
		if len(pChoices) == 0 {
			return -1, -1, nil
		}
		return -1, pChoices[0], nil
	}
	if len(pChoices) == 0 {
		return nChoices[0], -1, nil
	}
	for _, nIndex := range nChoices {
		for _, pIndex := range pChoices {
			if validCID42AnchorPair(points[nIndex].Address, points[pIndex].Address) {
				return nIndex, pIndex, nil
			}
		}
	}
	// Keep the highest-ranked pair so the caller returns the existing detailed
	// N/P layout error rather than hiding malformed operator input.
	return nChoices[0], pChoices[0], nil
}

func findCID42AnchorCandidates(points []TtdcTemplatePointRequest, baseDataID string, nameMatcher func(string) bool) (cid42AnchorCandidates, error) {
	candidates := cid42AnchorCandidates{named: -1, persisted: -1}
	named, persisted := -1, -1
	for index, point := range points {
		if nameMatcher(point.Name) {
			if named >= 0 {
				return candidates, fmt.Errorf("42模板存在多个%s数据点", cid42AnchorLabel(baseDataID))
			}
			named = index
		}
		if strings.TrimSpace(point.BaseDataID) == baseDataID {
			if persisted >= 0 {
				return candidates, fmt.Errorf("42模板存在多个%s系统ID", cid42AnchorLabel(baseDataID))
			}
			persisted = index
		}
	}
	candidates.named, candidates.persisted = named, persisted
	return candidates, nil
}

func rankedCID42AnchorChoices(points []TtdcTemplatePointRequest, candidates cid42AnchorCandidates, baseDataID string) []int {
	result := make([]int, 0, 2)
	appendCandidate := func(index int) {
		if index < 0 || !validCID42AnchorAddress(baseDataID, points[index].Address) {
			return
		}
		for _, existing := range result {
			if existing == index {
				return
			}
		}
		result = append(result, index)
	}
	// A valid persisted ID remains stable after a display-name edit. If that
	// old ID points to an impossible address, the named protocol marker is used
	// to repair templates saved by earlier releases.
	appendCandidate(candidates.persisted)
	appendCandidate(candidates.named)
	if len(result) == 0 {
		if candidates.persisted >= 0 {
			result = append(result, candidates.persisted)
		}
		if candidates.named >= 0 && candidates.named != candidates.persisted {
			result = append(result, candidates.named)
		}
	}
	return result
}

func validCID42AnchorAddress(baseDataID string, address int) bool {
	if baseDataID == "temperature_count" {
		return address >= 10 && (address-6)%4 == 0
	}
	return address >= 28 && address%2 == 0
}

func validCID42AnchorPair(nAddress, pAddress int) bool {
	if !validCID42AnchorAddress("temperature_count", nAddress) || !validCID42AnchorAddress("custom_count", pAddress) {
		return false
	}
	temperatureEnd := pAddress - 28
	return temperatureEnd >= nAddress+2 && (temperatureEnd-(nAddress+2))%4 == 0
}

func cid42AnchorLabel(baseDataID string) string {
	if baseDataID == "temperature_count" {
		return "N（电芯温度数量）"
	}
	return "P（自定义遥测量数量）"
}

func normalizedCID42PointName(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	return strings.NewReplacer(" ", "", "\t", "", "（", "(", "）", ")").Replace(value)
}

func isCID42NName(value string) bool {
	name := normalizedCID42PointName(value)
	return strings.HasPrefix(name, "N(") || name == "N" || strings.Contains(name, "电芯温度数量") || strings.Contains(name, "温度数量")
}

func isCID42PName(value string) bool {
	name := normalizedCID42PointName(value)
	return strings.HasPrefix(name, "P(") || name == "P" || strings.Contains(name, "自定义遥测量数量")
}

func isCID42ReservedBaseDataID(value string) bool {
	value = strings.TrimSpace(value)
	if cid42ReservedBaseDataIDs[value] {
		return true
	}
	if _, ok := model.TtdcCID42CellVoltageIndex(value); ok {
		return true
	}
	_, ok := model.TtdcCID42CellTemperatureIndex(value)
	return ok
}

func NewTtdcTemplateService(_ *gin.Context) *TtdcTemplateService {
	return &TtdcTemplateService{db: opengauss.GetDefaultGormDB()}
}

type TtdcTemplateService struct{ db *gorm.DB }

func requireTtdcSchema(db *gorm.DB, runtimeReady bool) error {
	if db == nil {
		return errors.New("默认OpenGauss数据源未初始化")
	}
	if !db.Migrator().HasTable(&model.TtdcSchemaVersionRecord{}) {
		return errors.New("TTDC模板组数据库结构尚未迁移")
	}
	var state model.TtdcSchemaVersionRecord
	if err := db.Where("component = ?", model.TtdcSchemaComponent).First(&state).Error; err != nil {
		return errors.New("TTDC模板组数据库结构尚未准备完成")
	}
	if state.Version != model.TtdcSchemaVersion {
		return fmt.Errorf("TTDC模板组数据库版本不匹配：当前%d，要求%d", state.Version, model.TtdcSchemaVersion)
	}
	if runtimeReady {
		if !model.TtdcSchemaRuntimeUsable(state.Status) {
			return fmt.Errorf("TTDC模板组数据库结构尚未完成初始化，当前状态：%s", state.Status)
		}
		return nil
	}
	if !model.TtdcSchemaRuntimeUsable(state.Status) {
		return fmt.Errorf("TTDC模板组当前不可配置，迁移状态：%s", state.Status)
	}
	return nil
}

func (s *TtdcTemplateService) checkDB() error {
	if s == nil || s.db == nil {
		return errors.New("默认OpenGauss数据源未初始化")
	}
	return requireTtdcSchema(s.db, false)
}

func (s *TtdcTemplateService) FindPage(req TtdcTemplatePageRequest) (any, int64, error) {
	if err := s.checkDB(); err != nil {
		return nil, 0, err
	}
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 || req.PageSize > 200 {
		req.PageSize = 20
	}
	query := s.db.Model(&model.TtdcTemplate{}).Order("id desc")
	if req.GroupID > 0 {
		query = query.Where("group_id = ?", req.GroupID)
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("template_name LIKE ? OR template_code LIKE ?", like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.TtdcTemplate
	if err := query.Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	if err := s.attachPointCounts(rows); err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

func (s *TtdcTemplateService) attachPointCounts(rows []model.TtdcTemplate) error {
	if len(rows) == 0 {
		return nil
	}
	ids := make([]uint, len(rows))
	for i := range rows {
		ids[i] = rows[i].ID
	}
	var counts []struct {
		TemplateID uint
		Count      int64
	}
	if err := s.db.Model(&model.TtdcTemplatePoint{}).Select("template_id, count(*) AS count").Where("template_id IN ?", ids).Group("template_id").Scan(&counts).Error; err != nil {
		return err
	}
	byID := map[uint]int64{}
	for _, count := range counts {
		byID[count.TemplateID] = count.Count
	}
	for i := range rows {
		rows[i].PointCount = byID[rows[i].ID]
	}
	return nil
}

func (s *TtdcTemplateService) Detail(id uint) (model.TtdcTemplate, error) {
	var item model.TtdcTemplate
	if err := s.checkDB(); err != nil {
		return item, err
	}
	err := s.db.Preload("Points", func(db *gorm.DB) *gorm.DB { return db.Order("sort_no asc, id asc") }).First(&item, id).Error
	return item, err
}

type TtdcTemplateOptionView struct {
	ID            uint   `json:"id"`
	GroupID       uint   `json:"groupId"`
	GroupCode     string `json:"groupCode"`
	GroupName     string `json:"groupName"`
	TopicSegment  string `json:"topicSegment"`
	TemplateCode  string `json:"templateCode"`
	TemplateName  string `json:"templateName"`
	CID2Type      string `json:"cid2Type"`
	ProtocolCode  string `json:"protocolCode"`
	MachineCID2   string `json:"machineCid2"`
	ParserProfile string `json:"parserProfile"`
	Version       int    `json:"version"`
}

func (s *TtdcTemplateService) Options(groupID uint, gatewayID, moduleADR string) ([]TtdcTemplateOptionView, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("默认OpenGauss数据源未初始化")
	}
	if err := requireTtdcSchema(s.db, true); err != nil {
		return nil, err
	}
	if strings.TrimSpace(gatewayID) != "" && strings.TrimSpace(moduleADR) != "" {
		var binding model.GatewayTtdcAddressGroupBinding
		err := s.db.Where("gateway_id = ? AND module_adr = ?", strings.TrimSpace(gatewayID), strings.ToUpper(strings.TrimSpace(moduleADR))).First(&binding).Error
		if err == nil {
			groupID = binding.TemplateGroupID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	query := s.db.Table("ttdc_template AS t").Select("t.id, t.group_id, g.group_code, g.group_name, g.topic_segment, t.template_code, t.template_name, t.cid2_type, t.protocol_code, t.machine_cid2, t.parser_profile, t.version").Joins("JOIN ttdc_template_group g ON g.id = t.group_id AND g.deleted_at IS NULL").Where("t.enabled = ? AND t.deleted_at IS NULL AND g.enabled = ?", true, true)
	if groupID > 0 {
		query = query.Where("t.group_id = ?", groupID)
	}
	var rows []TtdcTemplateOptionView
	err := query.Order("g.group_name asc, t.cid2_type asc").Scan(&rows).Error
	return rows, err
}

func (s *TtdcTemplateService) Save(req TtdcTemplateRequest, userID int) (model.TtdcTemplate, error) {
	item, err := normalizeTemplateRequest(req)
	if err != nil {
		return item, err
	}
	if err = s.checkDB(); err != nil {
		return item, err
	}
	points := item.Points
	hasBindings := false
	originalGroupID := item.GroupID
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var previousPoints []model.TtdcTemplatePoint
		var group model.TtdcTemplateGroup
		if err := tx.First(&group, item.GroupID).Error; err != nil {
			return errors.New("模板组不存在")
		}
		operation := "create"
		if item.ID == 0 {
			item.CreatedBy, item.UpdatedBy = userID, userID
			if err := ensureTemplateMachineCodesAvailable(tx, item); err != nil {
				return err
			}
			if err := tx.Omit("Points", "Topics").Create(&item).Error; err != nil {
				return err
			}
		} else {
			operation = "update"
			var current model.TtdcTemplate
			if err := tx.First(&current, item.ID).Error; err != nil {
				return err
			}
			originalGroupID = current.GroupID
			if req.Version > 0 && req.Version != current.Version {
				return errors.New("模板已被其他用户修改")
			}
			item.TemplateCode, item.ProtocolCode = current.TemplateCode, current.ProtocolCode
			var bindings int64
			if err := tx.Model(&model.GatewayTtdcTemplateBinding{}).Where("template_id = ?", item.ID).Count(&bindings).Error; err != nil {
				return err
			}
			hasBindings = bindings > 0
			if bindings > 0 && (item.GroupID != current.GroupID || item.CID2Type != current.CID2Type) {
				return errors.New("模板已有实际通道引用，机器身份字段不可修改")
			}
			if bindings > 0 {
				item.MachineCID2, item.ParserProfile = current.MachineCID2, current.ParserProfile
			}
			item.Version, item.UpdatedBy = current.Version+1, userID
			if err := tx.Model(&current).Select("group_id", "template_name", "cid2_type", "machine_cid2", "parser_profile", "enabled", "version", "description", "updated_by").Updates(&item).Error; err != nil {
				return err
			}
		}
		for i := range points {
			points[i].TemplateID, points[i].GroupID = item.ID, item.GroupID
		}
		// Older releases used soft deletion for template points. Remove those
		// hidden rows before validating IDs so they no longer occupy uniqueness.
		if err := tx.Unscoped().Where("group_id = ? AND deleted_at IS NOT NULL", item.GroupID).Delete(&model.TtdcTemplatePoint{}).Error; err != nil {
			return err
		}
		if err := validateGroupPointIDs(tx, item.GroupID, item.ID, points); err != nil {
			return err
		}
		if hasBindings && item.CID2Type == model.TtdcRoleCID42SOC {
			if err := validateBoundAlgorithmKeys(tx, item.ID, points); err != nil {
				return err
			}
		}
		if hasBindings {
			if err := tx.Where("template_id = ?", item.ID).Find(&previousPoints).Error; err != nil {
				return err
			}
		}
		// Actual channels own independent ModuleTtdcData/GatewayData snapshots.
		// Permit only protocol-derived CID42 ID repairs in the source template;
		// arbitrary IDs and algorithm-result IDs stay immutable once referenced.
		allowCID42ProtocolRepair := hasBindings && item.CID2Type == model.TtdcRoleCID42
		if err := reconcileTemplatePoints(tx, item.ID, points, !hasBindings, allowCID42ProtocolRepair); err != nil {
			return err
		}
		if hasBindings {
			addedPoints := addedEnabledTemplatePoints(previousPoints, points)
			if err := propagateTemplatePointAdditions(tx, item.ID, addedPoints); err != nil {
				return fmt.Errorf("向已有网关补充新增数据点失败：%w", err)
			}
		}
		if group.RuntimeMatchEnabled || group.AutoCreateEnabled {
			if err := requireCompleteGroup(tx, group.ID); err != nil {
				return fmt.Errorf("在线模板组修改后不完整：%w", err)
			}
		}
		if originalGroupID != item.GroupID {
			var originalGroup model.TtdcTemplateGroup
			if err := tx.First(&originalGroup, originalGroupID).Error; err != nil {
				return err
			}
			if originalGroup.RuntimeMatchEnabled || originalGroup.AutoCreateEnabled {
				if err := requireCompleteGroup(tx, originalGroup.ID); err != nil {
					return fmt.Errorf("原在线模板组修改后不完整：%w", err)
				}
			}
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "template", TargetKey: strconv.FormatUint(uint64(item.ID), 10), Operation: operation}).Error
	})
	if err != nil {
		return item, err
	}
	return s.Detail(item.ID)
}

func addedEnabledTemplatePoints(previous, current []model.TtdcTemplatePoint) []model.TtdcTemplatePoint {
	previousIDs := make(map[uint]bool, len(previous))
	previousBaseIDs := make(map[string]bool, len(previous))
	for _, point := range previous {
		previousIDs[point.ID] = true
		previousBaseIDs[point.BaseDataID] = true
	}
	added := make([]model.TtdcTemplatePoint, 0)
	for _, point := range current {
		if !point.Enabled || previousIDs[point.ID] || previousBaseIDs[point.BaseDataID] {
			continue
		}
		added = append(added, point)
	}
	return added
}

// propagateTemplatePointAdditions copies only newly-created template points to
// channels already bound to this exact template. Existing channel rows are
// deliberately never updated or deleted, preserving manual gateway settings.
func propagateTemplatePointAdditions(tx *gorm.DB, templateID uint, points []model.TtdcTemplatePoint) error {
	if len(points) == 0 {
		return nil
	}
	const bindingBatchSize = 500
	lastID := uint(0)
	for {
		var bindings []model.GatewayTtdcTemplateBinding
		if err := tx.Where("template_id = ? AND id > ?", templateID, lastID).Order("id asc").Limit(bindingBatchSize).Find(&bindings).Error; err != nil {
			return err
		}
		if len(bindings) == 0 {
			return nil
		}

		moduleIDs := make([]uint, 0, len(bindings))
		gatewayIDs := make([]string, 0, len(bindings))
		dataIDs := make([]string, 0, len(bindings)*len(points))
		for _, binding := range bindings {
			moduleIDs = append(moduleIDs, binding.ModuleID)
			gatewayIDs = append(gatewayIDs, binding.GatewayID)
			for _, point := range points {
				dataIDs = append(dataIDs, strings.ToUpper(binding.ModuleADR)+"_"+point.BaseDataID)
			}
		}

		var existingModulePoints []model.ModuleTtdcData
		if err := tx.Where("module_id IN ? AND data_id IN ?", moduleIDs, dataIDs).Find(&existingModulePoints).Error; err != nil {
			return err
		}
		modulePointExists := make(map[string]bool, len(existingModulePoints))
		for _, point := range existingModulePoints {
			modulePointExists[fmt.Sprintf("%d:%s", point.ModuleId, point.DataId)] = true
		}

		var existingGatewayPoints []model.GatewayData
		if err := tx.Where("gateway_id IN ? AND data_id IN ?", gatewayIDs, dataIDs).Find(&existingGatewayPoints).Error; err != nil {
			return err
		}
		gatewayPointExists := make(map[string]bool, len(existingGatewayPoints))
		for _, point := range existingGatewayPoints {
			gatewayPointExists[point.GatewayId+":"+point.DataId] = true
		}

		moduleRows := make([]model.ModuleTtdcData, 0, len(bindings)*len(points))
		gatewayRows := make([]model.GatewayData, 0, len(bindings)*len(points))
		changedGateways := make(map[string]bool)
		for _, binding := range bindings {
			adr := strings.ToUpper(strings.TrimSpace(binding.ModuleADR))
			for _, point := range points {
				dataID := adr + "_" + point.BaseDataID
				name := adr + "_" + point.Name
				moduleExists := modulePointExists[fmt.Sprintf("%d:%s", binding.ModuleID, dataID)]
				gatewayExists := gatewayPointExists[binding.GatewayID+":"+dataID]
				// A gateway-level ID owned elsewhere is a configuration conflict. Do
				// not create a second module point with that same runtime identity.
				if !moduleExists && !gatewayExists {
					moduleRows = append(moduleRows, model.ModuleTtdcData{
						ModuleId: int(binding.ModuleID), Name: name, DataId: dataID,
						DataType: point.DataType, Transport: templatePointTransport(point),
						Formula: point.Formula, Unformula: point.Unformula, AlgorithmKey: point.AlgorithmKey,
					})
					changedGateways[binding.GatewayID] = true
				}
				if !gatewayExists {
					gatewayRows = append(gatewayRows, model.GatewayData{
						GatewayId: binding.GatewayID, DataId: dataID, DataType: point.DataType, Name: name,
					})
					changedGateways[binding.GatewayID] = true
				}
			}
		}
		if len(moduleRows) > 0 {
			if err := tx.CreateInBatches(moduleRows, 500).Error; err != nil {
				return err
			}
		}
		if len(gatewayRows) > 0 {
			if err := tx.CreateInBatches(gatewayRows, 500).Error; err != nil {
				return err
			}
		}
		if len(changedGateways) > 0 {
			revisions := make([]model.TtdcConfigRevision, 0, len(changedGateways))
			for gatewayID := range changedGateways {
				revisions = append(revisions, model.TtdcConfigRevision{ConfigType: "module", TargetKey: gatewayID, Operation: "update"})
			}
			if err := tx.CreateInBatches(revisions, 500).Error; err != nil {
				return err
			}
		}
		lastID = bindings[len(bindings)-1].ID
	}
}

func templatePointTransport(point model.TtdcTemplatePoint) string {
	transport := strings.TrimSpace(point.Transport)
	if transport == "" && point.Address != nil && point.ByteCount != nil && point.ConvertType != nil {
		transport = fmt.Sprintf("%d %d %d", *point.Address, *point.ByteCount, *point.ConvertType)
		if point.BitIndex != nil {
			transport += fmt.Sprintf(" %d", *point.BitIndex)
		}
	}
	return transport
}

func validateBoundAlgorithmKeys(tx *gorm.DB, templateID uint, incoming []model.TtdcTemplatePoint) error {
	var existing []model.TtdcTemplatePoint
	if err := tx.Where("template_id = ?", templateID).Find(&existing).Error; err != nil {
		return err
	}
	byID := make(map[uint]model.TtdcTemplatePoint, len(incoming))
	byBase := make(map[string]model.TtdcTemplatePoint, len(incoming))
	for _, point := range incoming {
		byID[point.ID] = point
		byBase[point.BaseDataID] = point
	}
	for _, current := range existing {
		next, ok := byID[current.ID]
		if !ok {
			next, ok = byBase[current.BaseDataID]
		}
		if !ok {
			continue
		}
		if pointerStringValue(current.AlgorithmKey) != pointerStringValue(next.AlgorithmKey) {
			return fmt.Errorf("数据点%s已有实际通道引用，algorithm_key不可修改", current.BaseDataID)
		}
	}
	return nil
}

func pointerStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func ensureTemplateMachineCodesAvailable(tx *gorm.DB, item model.TtdcTemplate) error {
	var count int64
	if err := tx.Unscoped().Model(&model.TtdcTemplate{}).Where("template_code = ? OR protocol_code = ?", item.TemplateCode, item.ProtocolCode).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New("模板编码或协议编码已存在（包括已删除记录）")
	}
	if err := tx.Model(&model.TtdcTemplate{}).Where("group_id = ? AND cid2_type = ?", item.GroupID, item.CID2Type).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("模板组内已存在%s模板", item.CID2Type)
	}
	return nil
}

func validateGroupPointIDs(tx *gorm.DB, groupID, templateID uint, points []model.TtdcTemplatePoint) error {
	baseIDs := make([]string, len(points))
	for i := range points {
		baseIDs[i] = points[i].BaseDataID
	}
	var conflicts int64
	err := tx.Unscoped().Model(&model.TtdcTemplatePoint{}).Where("group_id = ? AND template_id <> ? AND base_data_id IN ?", groupID, templateID, baseIDs).Count(&conflicts).Error
	if err != nil {
		return err
	}
	if conflicts > 0 {
		return errors.New("同一模板组的42、42_SOC和44数据点ID不能重复")
	}
	return nil
}

func reconcileTemplatePoints(tx *gorm.DB, templateID uint, incoming []model.TtdcTemplatePoint, allowBaseIDChange, allowCID42ProtocolRepair bool) error {
	var existing []model.TtdcTemplatePoint
	if err := tx.Unscoped().Where("template_id = ?", templateID).Find(&existing).Error; err != nil {
		return err
	}
	byID, byBase := map[uint]model.TtdcTemplatePoint{}, map[string]model.TtdcTemplatePoint{}
	for _, point := range existing {
		byID[point.ID], byBase[point.BaseDataID] = point, point
	}
	kept := map[uint]bool{}
	for i := range incoming {
		point := &incoming[i]
		current, ok := byID[point.ID]
		if !ok {
			current, ok = byBase[point.BaseDataID]
		}
		if ok {
			protocolRepair := allowCID42ProtocolRepair && (isCID42ReservedBaseDataID(current.BaseDataID) || isCID42ReservedBaseDataID(point.BaseDataID))
			if current.BaseDataID != point.BaseDataID && !allowBaseIDChange && !protocolRepair {
				return fmt.Errorf("模板已被实际通道引用，数据点“%s”的系统ID不能修改", current.Name)
			}
			point.ID, kept[current.ID] = current.ID, true
			if err := tx.Unscoped().Model(&current).Select("group_id", "sort_no", "base_data_id", "name", "data_type", "address", "byte_count", "convert_type", "bit_index", "algorithm_key", "transport", "formula", "unformula", "enabled", "description", "deleted_at").Updates(map[string]interface{}{
				"group_id": point.GroupID, "sort_no": point.SortNo, "base_data_id": point.BaseDataID, "name": point.Name, "data_type": point.DataType,
				"address": point.Address, "byte_count": point.ByteCount, "convert_type": point.ConvertType, "bit_index": point.BitIndex,
				"algorithm_key": point.AlgorithmKey, "transport": point.Transport, "formula": point.Formula, "unformula": point.Unformula,
				"enabled": point.Enabled, "description": point.Description, "deleted_at": nil,
			}).Error; err != nil {
				return err
			}
			continue
		}
		point.ID = 0
		if err := tx.Create(point).Error; err != nil {
			return err
		}
		kept[point.ID] = true
	}
	for _, point := range existing {
		if !kept[point.ID] {
			if err := tx.Unscoped().Delete(&point).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func hardDeleteTemplateDefinitions(tx *gorm.DB, templateIDs []uint) error {
	if len(templateIDs) == 0 {
		return nil
	}
	// ttdc_template_topic is a legacy compatibility table and is not present
	// in every deployment. Its absence must not block deleting current tables.
	if tx.Migrator().HasTable(&model.TtdcTemplateTopic{}) {
		if err := tx.Unscoped().Where("template_id IN ?", templateIDs).Delete(&model.TtdcTemplateTopic{}).Error; err != nil {
			return err
		}
	}
	if err := tx.Unscoped().Where("template_id IN ?", templateIDs).Delete(&model.TtdcTemplatePoint{}).Error; err != nil {
		return err
	}
	return tx.Unscoped().Where("id IN ?", templateIDs).Delete(&model.TtdcTemplate{}).Error
}

func (s *TtdcTemplateService) SetEnabled(id uint, field string, value bool, _ bool) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	if field != "enabled" {
		return errors.New("模板只支持修改手工可选状态，运行和自动创建由模板组控制")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.TtdcTemplate{}).Where("id = ?", id).Update("enabled", value).Error; err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "template", TargetKey: strconv.FormatUint(uint64(id), 10), Operation: "update"}).Error
	})
}

func (s *TtdcTemplateService) Delete(id uint) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var bindings int64
		if err := tx.Model(&model.GatewayTtdcTemplateBinding{}).Where("template_id = ?", id).Count(&bindings).Error; err != nil {
			return err
		}
		if bindings > 0 {
			return errors.New("模板已被实际通道引用，不能删除")
		}
		var item model.TtdcTemplate
		if err := tx.First(&item, id).Error; err != nil {
			return err
		}
		var group model.TtdcTemplateGroup
		if err := tx.First(&group, item.GroupID).Error; err != nil {
			return err
		}
		if group.RuntimeMatchEnabled || group.AutoCreateEnabled {
			return errors.New("请先关闭模板组运行时接收和自动创建")
		}
		if err := hardDeleteTemplateDefinitions(tx, []uint{item.ID}); err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "template", TargetKey: strconv.FormatUint(uint64(id), 10), Operation: "delete"}).Error
	})
}

func (s *TtdcTemplateService) Profiles() []string { return profiles() }

type TtdcTemplateGroupService struct{ db *gorm.DB }

func NewTtdcTemplateGroupService() *TtdcTemplateGroupService {
	return &TtdcTemplateGroupService{db: opengauss.GetDefaultGormDB()}
}

func (s *TtdcTemplateGroupService) checkDB() error {
	if s == nil {
		return errors.New("默认OpenGauss数据源未初始化")
	}
	return requireTtdcSchema(s.db, false)
}

func (s *TtdcTemplateGroupService) FindPage(req TtdcTemplatePageRequest) ([]model.TtdcTemplateGroup, int64, error) {
	if err := s.checkDB(); err != nil {
		return nil, 0, err
	}
	if req.PageNum <= 0 {
		req.PageNum = 1
	}
	if req.PageSize <= 0 || req.PageSize > 200 {
		req.PageSize = 20
	}
	query := s.db.Model(&model.TtdcTemplateGroup{}).Order("id desc")
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("group_name LIKE ? OR group_code LIKE ? OR topic_segment LIKE ?", like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []model.TtdcTemplateGroup
	if err := query.Preload("Templates").Offset((req.PageNum - 1) * req.PageSize).Limit(req.PageSize).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	if err := s.attachGroupTemplatePointCounts(rows); err != nil {
		return nil, 0, err
	}
	for i := range rows {
		rows[i].TemplateCount = int64(len(rows[i].Templates))
		rows[i].Complete = templatesComplete(rows[i].Templates)
	}
	return rows, total, nil
}

func (s *TtdcTemplateGroupService) Detail(id uint) (model.TtdcTemplateGroup, error) {
	var item model.TtdcTemplateGroup
	if err := s.checkDB(); err != nil {
		return item, err
	}
	// 告警规则的数据点选择器依赖模板组详情返回完整的模板点；
	// 点仍由 GORM 的默认软删除条件过滤，避免把已删除点暴露给配置界面。
	err := s.db.
		Preload("Templates", func(db *gorm.DB) *gorm.DB { return db.Order("cid2_type asc") }).
		Preload("Templates.Points", func(db *gorm.DB) *gorm.DB { return db.Order("sort_no asc, id asc") }).
		First(&item, id).Error
	if err == nil {
		groups := []model.TtdcTemplateGroup{item}
		err = s.attachGroupTemplatePointCounts(groups)
		if err == nil {
			item = groups[0]
		}
	}
	item.TemplateCount, item.Complete = int64(len(item.Templates)), templatesComplete(item.Templates)
	return item, err
}

func (s *TtdcTemplateGroupService) attachGroupTemplatePointCounts(groups []model.TtdcTemplateGroup) error {
	ids := make([]uint, 0)
	for _, group := range groups {
		for _, item := range group.Templates {
			ids = append(ids, item.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	var counts []struct {
		TemplateID uint
		Count      int64
	}
	if err := s.db.Model(&model.TtdcTemplatePoint{}).Select("template_id, count(*) AS count").Where("template_id IN ?", ids).Group("template_id").Scan(&counts).Error; err != nil {
		return err
	}
	byID := make(map[uint]int64, len(counts))
	for _, count := range counts {
		byID[count.TemplateID] = count.Count
	}
	for groupIndex := range groups {
		for templateIndex := range groups[groupIndex].Templates {
			item := &groups[groupIndex].Templates[templateIndex]
			item.PointCount = byID[item.ID]
		}
	}
	return nil
}

func (s *TtdcTemplateGroupService) Options() ([]model.TtdcTemplateGroup, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	var rows []model.TtdcTemplateGroup
	err := s.db.Preload("Templates", func(db *gorm.DB) *gorm.DB { return db.Where("enabled = ?", true).Order("cid2_type asc") }).Where("enabled = ?", true).Order("group_name asc").Find(&rows).Error
	for i := range rows {
		rows[i].Complete = templatesComplete(rows[i].Templates)
		rows[i].TemplateCount = int64(len(rows[i].Templates))
	}
	return rows, err
}

func normalizeGroupRequest(req TtdcTemplateGroupRequest) (model.TtdcTemplateGroup, error) {
	item := model.TtdcTemplateGroup{ID: req.ID, GroupCode: strings.TrimSpace(req.GroupCode), GroupName: strings.TrimSpace(req.GroupName), TopicSegment: strings.TrimSpace(req.TopicSegment), Version: req.Version, Description: strings.TrimSpace(req.Description)}
	if !ttdcMachineCodePattern.MatchString(item.GroupCode) || item.GroupName == "" || !ttdcTopicSegmentPattern.MatchString(item.TopicSegment) || strings.EqualFold(item.TopicSegment, "PUB") {
		return item, errors.New("模板组编码、名称或主题区分段不合法")
	}
	item.Enabled = true
	if req.Enabled != nil {
		item.Enabled = *req.Enabled
	}
	if req.RuntimeMatchEnabled != nil {
		item.RuntimeMatchEnabled = *req.RuntimeMatchEnabled
	}
	if req.AutoCreateEnabled != nil {
		item.AutoCreateEnabled = *req.AutoCreateEnabled
	}
	if item.Version <= 0 {
		item.Version = 1
	}
	return item, nil
}

func (s *TtdcTemplateGroupService) Save(req TtdcTemplateGroupRequest, userID int) (model.TtdcTemplateGroup, error) {
	item, err := normalizeGroupRequest(req)
	if err != nil {
		return item, err
	}
	if err = s.checkDB(); err != nil {
		return item, err
	}
	if item.RuntimeMatchEnabled || item.AutoCreateEnabled {
		if err = requireTtdcSchema(s.db, true); err != nil {
			return item, err
		}
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		op := "create"
		if item.ID == 0 {
			if item.RuntimeMatchEnabled || item.AutoCreateEnabled {
				return errors.New("新模板组尚无42、42_SOC和44模板，不能开启运行时接收或自动创建")
			}
			var legacyGroup model.TtdcTemplateGroup
			legacyErr := tx.Unscoped().Where("group_code = ? AND deleted_at IS NOT NULL", item.GroupCode).First(&legacyGroup).Error
			if legacyErr == nil {
				var bindings int64
				if err := tx.Model(&model.GatewayTtdcAddressGroupBinding{}).Where("template_group_id = ?", legacyGroup.ID).Count(&bindings).Error; err != nil {
					return err
				}
				if bindings == 0 {
					if err := tx.Model(&model.GatewayTtdcTemplateBinding{}).Where("template_group_id = ?", legacyGroup.ID).Count(&bindings).Error; err != nil {
						return err
					}
				}
				if bindings > 0 {
					return errors.New("同编码的历史模板组仍有实际通道引用，不能重新创建")
				}
				var legacyTemplateIDs []uint
				if err := tx.Unscoped().Model(&model.TtdcTemplate{}).Where("group_id = ?", legacyGroup.ID).Pluck("id", &legacyTemplateIDs).Error; err != nil {
					return err
				}
				if err := hardDeleteTemplateDefinitions(tx, legacyTemplateIDs); err != nil {
					return err
				}
				if err := tx.Unscoped().Delete(&legacyGroup).Error; err != nil {
					return err
				}
			} else if !errors.Is(legacyErr, gorm.ErrRecordNotFound) {
				return legacyErr
			}
			var count int64
			if err := tx.Unscoped().Model(&model.TtdcTemplateGroup{}).Where("group_code = ?", item.GroupCode).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("模板组编码已存在")
			}
			item.CreatedBy, item.UpdatedBy = userID, userID
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
		} else {
			op = "update"
			var current model.TtdcTemplateGroup
			if err := tx.First(&current, item.ID).Error; err != nil {
				return err
			}
			if req.Version > 0 && req.Version != current.Version {
				return errors.New("模板组已被其他用户修改")
			}
			if item.GroupCode != current.GroupCode {
				return errors.New("模板组编码创建后不可修改")
			}
			var bindings int64
			if err := tx.Model(&model.GatewayTtdcAddressGroupBinding{}).Where("template_group_id = ?", item.ID).Count(&bindings).Error; err != nil {
				return err
			}
			if bindings > 0 && item.TopicSegment != current.TopicSegment {
				return errors.New("模板组已有ADR绑定，主题区分段不可修改")
			}
			if (!current.RuntimeMatchEnabled && item.RuntimeMatchEnabled) || (!current.AutoCreateEnabled && item.AutoCreateEnabled) {
				if err := requireCompleteGroup(tx, item.ID); err != nil {
					return err
				}
			}
			if current.RuntimeMatchEnabled && !item.RuntimeMatchEnabled && !req.ConfirmRuntimeChanges {
				return errors.New("关闭运行时接收必须确认")
			}
			item.Version, item.UpdatedBy = current.Version+1, userID
			if err := tx.Model(&current).Select("group_name", "topic_segment", "enabled", "runtime_match_enabled", "auto_create_enabled", "version", "description", "updated_by").Updates(&item).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "template_group", TargetKey: strconv.FormatUint(uint64(item.ID), 10), Operation: op}).Error
	})
	if err != nil {
		return item, err
	}
	return s.Detail(item.ID)
}

func (s *TtdcTemplateGroupService) SetState(id uint, field string, value, confirm bool) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	allowed := map[string]bool{"enabled": true, "runtime_match_enabled": true, "auto_create_enabled": true}
	if !allowed[field] {
		return errors.New("不支持的模板组状态字段")
	}
	if field == "runtime_match_enabled" && !value && !confirm {
		return errors.New("关闭运行时接收必须确认")
	}
	if value && (field == "runtime_match_enabled" || field == "auto_create_enabled") {
		if err := requireTtdcSchema(s.db, true); err != nil {
			return err
		}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		if value && (field == "runtime_match_enabled" || field == "auto_create_enabled") {
			if err := requireCompleteGroup(tx, id); err != nil {
				return err
			}
		}
		if err := tx.Model(&model.TtdcTemplateGroup{}).Where("id = ?", id).Update(field, value).Error; err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "template_group", TargetKey: strconv.FormatUint(uint64(id), 10), Operation: "update"}).Error
	})
}

func (s *TtdcTemplateGroupService) Delete(id uint) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var group model.TtdcTemplateGroup
		if err := tx.First(&group, id).Error; err != nil {
			return err
		}
		if group.RuntimeMatchEnabled || group.AutoCreateEnabled {
			return errors.New("请先关闭模板组运行时接收和自动创建")
		}
		var bindings int64
		if err := tx.Model(&model.GatewayTtdcAddressGroupBinding{}).Where("template_group_id = ?", id).Count(&bindings).Error; err != nil {
			return err
		}
		if bindings > 0 {
			return errors.New("模板组已有ADR历史绑定，不能删除")
		}
		if err := tx.Model(&model.GatewayTtdcTemplateBinding{}).Where("template_group_id = ?", id).Count(&bindings).Error; err != nil {
			return err
		}
		if bindings > 0 {
			return errors.New("模板组已有实际通道引用，不能删除")
		}
		var templates int64
		if err := tx.Model(&model.TtdcTemplate{}).Where("group_id = ?", id).Count(&templates).Error; err != nil {
			return err
		}
		if templates > 0 {
			return errors.New("请先删除组内模板")
		}
		var legacyTemplateIDs []uint
		if err := tx.Unscoped().Model(&model.TtdcTemplate{}).Where("group_id = ?", id).Pluck("id", &legacyTemplateIDs).Error; err != nil {
			return err
		}
		if err := hardDeleteTemplateDefinitions(tx, legacyTemplateIDs); err != nil {
			return err
		}
		if err := tx.Unscoped().Delete(&group).Error; err != nil {
			return err
		}
		return tx.Create(&model.TtdcConfigRevision{ConfigType: "template_group", TargetKey: strconv.FormatUint(uint64(id), 10), Operation: "delete"}).Error
	})
}

func templatesComplete(items []model.TtdcTemplate) bool {
	roles := map[string]int{}
	for _, item := range items {
		if item.DeletedAt.Valid {
			continue
		}
		roles[item.CID2Type]++
	}
	return roles[model.TtdcRoleCID42] == 1 && roles[model.TtdcRoleCID42SOC] == 1 && roles[model.TtdcRoleCID44] == 1
}

func requireCompleteGroup(tx *gorm.DB, id uint) error {
	var items []model.TtdcTemplate
	if err := tx.Preload("Points", func(db *gorm.DB) *gorm.DB { return db.Where("enabled = ?", true) }).Where("group_id = ?", id).Find(&items).Error; err != nil {
		return err
	}
	if !templatesComplete(items) {
		return errors.New("模板组必须完整包含且只能包含一个42、42_SOC和44模板")
	}
	cid42CellLimit, cid42SOCCellLimit := 0, 0
	for _, item := range items {
		if len(item.Points) == 0 {
			return fmt.Errorf("模板%s没有启用数据点", item.TemplateName)
		}
		if err := validateTemplateForActivation(item); err != nil {
			return fmt.Errorf("模板%s配置不合法：%w", item.TemplateName, err)
		}
		switch item.CID2Type {
		case model.TtdcRoleCID42:
			baseDataIDs := make([]string, 0, len(item.Points))
			for _, point := range item.Points {
				baseDataIDs = append(baseDataIDs, point.BaseDataID)
			}
			var err error
			cid42CellLimit, _, err = model.TtdcCID42PointLimits(baseDataIDs)
			if err != nil {
				return fmt.Errorf("模板%s配置不合法：%w", item.TemplateName, err)
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
				return fmt.Errorf("模板%s配置不合法：%w", item.TemplateName, err)
			}
		}
	}
	if cid42CellLimit != cid42SOCCellLimit {
		return fmt.Errorf("42模板配置了%d路单体电压，但42_SOC模板配置了%d路单体SOC", cid42CellLimit, cid42SOCCellLimit)
	}
	return nil
}

func validateTemplateForActivation(item model.TtdcTemplate) error {
	if !model.TtdcProfileCompatibleWithRole(item.CID2Type, item.ParserProfile) {
		return errors.New("模板角色与解析Profile不匹配")
	}
	if item.CID2Type != model.TtdcRoleCID42 && item.CID2Type != model.TtdcRoleCID42SOC && item.CID2Type != model.TtdcRoleCID44 {
		return errors.New("模板角色不支持")
	}
	seenAlgorithms := make(map[string]bool, len(item.Points))
	for _, point := range item.Points {
		if item.CID2Type == model.TtdcRoleCID42SOC {
			key, expectedType, ok := model.TtdcAlgorithmResultKey(point.BaseDataID, point.AlgorithmKey)
			if !ok || seenAlgorithms[key] || expectedType != point.DataType {
				return fmt.Errorf("42_SOC数据点%s的ID不受支持、重复或类型不匹配", point.BaseDataID)
			}
			seenAlgorithms[key] = true
			if point.Address != nil || point.ByteCount != nil || point.ConvertType != nil || point.BitIndex != nil {
				return fmt.Errorf("算法数据点%s不能包含原始报文地址", point.BaseDataID)
			}
			continue
		}
		if point.DataType != 1 || point.Address == nil || point.ByteCount == nil || point.ConvertType == nil {
			return fmt.Errorf("原始数据点%s的类型或传输字段不完整", point.BaseDataID)
		}
		if *point.Address < 0 || *point.Address%2 != 0 || *point.ByteCount <= 0 || *point.ConvertType < 1 || *point.ConvertType > 3 {
			return fmt.Errorf("原始数据点%s的传输字段非法", point.BaseDataID)
		}
		if item.CID2Type == model.TtdcRoleCID42 && point.BitIndex != nil {
			return fmt.Errorf("CID42数据点%s不能配置位索引", point.BaseDataID)
		}
		if *point.ConvertType == 2 && point.BitIndex == nil {
			return fmt.Errorf("二进制数据点%s缺少位索引", point.BaseDataID)
		}
		if *point.ConvertType != 2 && point.BitIndex != nil {
			return fmt.Errorf("十进制数据点%s不能配置位索引", point.BaseDataID)
		}
		if point.BitIndex != nil && (*point.BitIndex < 0 || *point.BitIndex >= *point.ByteCount*8) {
			return fmt.Errorf("数据点%s位索引越界", point.BaseDataID)
		}
		if point.AlgorithmKey != nil {
			return fmt.Errorf("原始数据点%s不能配置algorithm_key", point.BaseDataID)
		}
	}
	if item.CID2Type == model.TtdcRoleCID42 {
		baseDataIDs := make([]string, 0, len(item.Points))
		for _, point := range item.Points {
			baseDataIDs = append(baseDataIDs, point.BaseDataID)
		}
		if _, _, validationErr := model.TtdcCID42PointLimits(baseDataIDs); validationErr != nil {
			return validationErr
		}
	} else if item.CID2Type == model.TtdcRoleCID42SOC {
		keys := make([]string, 0, len(item.Points))
		for _, point := range item.Points {
			key, _, _ := model.TtdcAlgorithmResultKey(point.BaseDataID, point.AlgorithmKey)
			keys = append(keys, key)
		}
		if _, validationErr := model.TtdcCID42SOCPointLimit(keys); validationErr != nil {
			return validationErr
		}
	}
	return nil
}

func WriteTemplateFailure(ctx *gin.Context, err error) {
	response.Fail(ctx, templateErrorMessage(err), nil)
}

func templateErrorMessage(err error) string {
	if err == nil {
		return "模板操作失败"
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "未找到对应的模板数据"
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "模板操作失败"
	}
	for _, value := range message {
		if unicode.Is(unicode.Han, value) {
			return message
		}
	}
	log.Printf("[TTDC-TEMPLATE] interface error: %v", err)
	return "模板操作失败，请查看服务日志"
}
func SortTemplatePoints(points []model.TtdcTemplatePoint) {
	sort.SliceStable(points, func(i, j int) bool { return points[i].SortNo < points[j].SortNo })
}
