package battery

import (
	"deal/internal/batterylog"
	"deal/internal/tolink/model"
	"deal/util"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

const (
	cid42 = "42"
	cid44 = "44"
)

var (
	cellVoltagePattern     = regexp.MustCompile(`^单体电池(1[0-6]|[1-9])电压$`)
	cellTemperaturePattern = regexp.MustCompile(`^(?:电芯温度|NTC)([1-8])(?:数据|点温度)$`)
	nameReplacer           = strings.NewReplacer("（", "(", "）", ")", " ", "", "\t", "", "\r", "", "\n", "")
)

func Deal(ctx *gin.Context, msgOrg *primitive.MessageExt) error {
	if !Enabled() {
		if Configured() {
			return fmt.Errorf("battery parser is not ready")
		}
		return nil
	}
	configAccessMu.RLock()
	defer configAccessMu.RUnlock()

	var msg MQTTMessage
	if err := json.Unmarshal(msgOrg.Body, &msg); err != nil {
		return permanentf("invalid rocket payload: %v", err)
	}
	debugLogf("[BATTERY-DEAL] decoded messageID=%s rocketMsgID=%s gatewayID=%s topic=%s receivedAt=%d payloadLen=%d payload=%q\n", msg.MessageID, msgOrg.MsgId, msg.GatewayID, msg.Topic, msg.ReceivedAt, len(msg.Payload), msg.Payload)

	if msg.MessageID == "" {
		msg.MessageID = msgOrg.MsgId
	}
	if msg.MessageID != "" {
		processed, err := isMessageProcessed(ctx, msg.MessageID)
		if err != nil {
			return fmt.Errorf("check processed battery message: %w", err)
		}
		if processed {
			debugLogf("[BATTERY-DEAL] duplicate message skipped messageID=%s gatewayID=%s\n", msg.MessageID, msg.GatewayID)
			return nil
		}
	}

	receivedAt := msg.ReceivedAt
	if receivedAt <= 0 && msgOrg.BornTimestamp > 0 {
		receivedAt = msgOrg.BornTimestamp / 1000
	}
	if receivedAt <= 0 {
		receivedAt = time.Now().Unix()
	}

	packet, err := normalizeCompletePacket(msg.Payload)
	if err != nil {
		return permanentf("normalize complete packet: %v", err)
	}
	debugLogf("[BATTERY-PARSE] normalized packet messageID=%s gatewayID=%s packetLen=%d\n", msg.MessageID, msg.GatewayID, len(packet))
	topicSegment, topicGatewayID, ok := ParseBatteryTopic(msg.Topic)
	if !ok {
		return permanentf("unsupported battery topic: %s", msg.Topic)
	}
	if strings.TrimSpace(msg.GatewayID) != "" && strings.TrimSpace(msg.GatewayID) != topicGatewayID {
		return permanentf("message gatewayID %s does not match topic gatewayID %s", msg.GatewayID, topicGatewayID)
	}
	decoded, err := decodePacket(packet)
	if err != nil {
		return permanentf("validate packet failed: %v", err)
	}
	template, matched, matchErr := MatchTemplateAt(ctx, topicSegment, decoded.RawCID2)
	if matchErr != nil {
		return fmt.Errorf("match runtime template: %w", matchErr)
	}
	var parsed ParsedData
	if LegacyTemplateMode() {
		debugLogf("[BATTERY-TEMPLATE] template-group schema is not initialized; using legacy channel match gatewayID=%s topic=%s matchedLegacyTemplate=%t\n", topicGatewayID, msg.Topic, matched)
		parsed, err = ParsePacketAt(msg.Topic, packet, time.Unix(receivedAt, 0))
	} else if !matched {
		return permanentf("no runtime template for topic segment %s and CID2 %s", topicSegment, decoded.RawCID2)
	} else {
		group, groupOK := MatchTemplateGroup(topicSegment)
		if !groupOK || group.ID != template.GroupID {
			return permanentf("runtime template group is unavailable for topic segment %s", topicSegment)
		}
		// CID42 INFO is variable-length. Validate the template contract here,
		// then let each point decide whether it is present in this packet.
		if structureErr := validateTemplateStructure(template); structureErr != nil {
			return permanentf("template %s: %v", template.TemplateCode, structureErr)
		}
		if _, err = EnsureGateway(ctx, topicGatewayID); err == nil {
			var prepared PreparedBatteryChannels
			prepared, err = EnsureModules(ctx, topicGatewayID, topicSegment, decoded.Request, decoded.RawCID2, len(decoded.Info), group)
			if err == nil {
				parsed, err = ParsePreparedBatteryPacket(prepared, decoded, time.Unix(receivedAt, 0))
			}
		}
	}
	if err != nil {
		return fmt.Errorf("parse packet failed: %w", err)
	}
	if err := WriteParsed(ctx, parsed); err != nil {
		return err
	}
	debugLogf("[BATTERY-DEAL] stored messageID=%s gatewayID=%s numericPoints=%d stringPoints=%d receivedAt=%d\n", msg.MessageID, parsed.GatewayID, len(parsed.Points), len(parsed.StrPoints), receivedAt)
	if msg.MessageID != "" {
		if err := markMessageProcessed(ctx, msg.MessageID); err != nil {
			return fmt.Errorf("mark battery message processed: %w", err)
		}
	}
	return nil
}

func ParsePacket(topic string, packet string) (ParsedData, error) {
	return ParsePacketAt(topic, packet, time.Now())
}

func ParsePacketAt(topic string, packet string, receivedAt time.Time) (ParsedData, error) {
	topicKind, gatewayID, ok := ParseTopic(topic)
	if !ok {
		return ParsedData{}, fmt.Errorf("unsupported battery topic: %s", topic)
	}

	modules, ok := GetModulesByGatewayID(gatewayID)
	if !ok {
		batterylog.Printf("[BATTERY-PARSE] module cache missed gatewayID=%s topic=%s\n", gatewayID, topic)
		return ParsedData{}, fmt.Errorf("gateway module cache missed: %s", gatewayID)
	}
	debugLogf("[BATTERY-PARSE] begin gatewayID=%s topicKind=%s modules=%d packetLen=%d request=%q response=%q receivedAt=%d\n", gatewayID, topicKind, len(modules), len(packet), firstPacketFrame(packet), secondPacketFrame(packet), receivedAt.Unix())

	decoded, err := decodePacket(packet)
	if err != nil {
		return ParsedData{}, err
	}

	receivedAt = time.Unix(receivedAt.Unix(), 0).In(receivedAt.Location())
	result := ParsedData{GatewayID: gatewayID}
	for _, module := range modules {
		if !moduleMatchesFrame(module, topicKind, decoded.Request) {
			continue
		}
		rawCID2 := decoded.RawCID2
		switch rawCID2 {
		case cid42:
			parsed := parseCID42(module, topicKind, gatewayID, decoded.Info, receivedAt)
			result.Points = append(result.Points, parsed.Points...)
			result.StrPoints = append(result.StrPoints, parsed.StrPoints...)
		case cid44:
			parsed := parseCID44(module, gatewayID, decoded.Info, receivedAt)
			result.Points = append(result.Points, parsed.Points...)
			result.StrPoints = append(result.StrPoints, parsed.StrPoints...)
		}
	}
	if len(result.Points) == 0 && len(result.StrPoints) == 0 {
		batterylog.Printf("[BATTERY-PARSE] no module matched gatewayID=%s topicKind=%s request=%q modules=%d\n", gatewayID, topicKind, decoded.Request, len(modules))
		return ParsedData{}, fmt.Errorf("no module matched packet")
	}
	debugLogf("[BATTERY-PARSE] success gatewayID=%s topicKind=%s numericPoints=%d stringPoints=%d\n", gatewayID, topicKind, len(result.Points), len(result.StrPoints))
	return result, nil
}

type decodedPacket struct{ Request, Response, Info, RawCID2 string }

func decodePacket(packet string) (decodedPacket, error) {
	lines := strings.Split(packet, "\r")
	if len(lines) != 2 {
		return decodedPacket{}, fmt.Errorf("paired frame is incomplete")
	}
	if len(lines[0]) < minProtocolFrameLen || len(lines[1]) < minProtocolFrameLen {
		return decodedPacket{}, fmt.Errorf("frame too short")
	}
	if !strings.HasPrefix(lines[0], "~") || !strings.HasPrefix(lines[1], "~") {
		return decodedPacket{}, fmt.Errorf("frame missing SOI")
	}
	if !strings.EqualFold(lines[0][len(lines[0])-4:], CalcChecksum(lines[0])) {
		return decodedPacket{}, fmt.Errorf("request checksum mismatch")
	}
	if !strings.EqualFold(lines[1][len(lines[1])-4:], CalcChecksum(lines[1])) {
		return decodedPacket{}, fmt.Errorf("response checksum mismatch")
	}
	if !IsNormalData(lines[1][7:9]) {
		return decodedPacket{}, fmt.Errorf("response RTN is not normal: %s", lines[1][7:9])
	}
	if lines[0][1:7] != lines[1][1:7] {
		return decodedPacket{}, fmt.Errorf("request and response VER/ADR/CID1 headers differ")
	}
	if _, validADR := model.NormalizeTtdcADR(lines[0][3:5]); !validADR {
		return decodedPacket{}, fmt.Errorf("invalid ADR: %s", lines[0][3:5])
	}
	infoLen := GetLength(lines[1][9:13])
	if infoLen <= 0 {
		return decodedPacket{}, fmt.Errorf("empty INFO")
	}
	infoEnd := 13 + infoLen
	if infoEnd > len(lines[1])-4 {
		return decodedPacket{}, fmt.Errorf("INFO length overflow")
	}
	return decodedPacket{Request: lines[0], Response: lines[1], Info: lines[1][13:infoEnd], RawCID2: strings.ToUpper(lines[0][7:9])}, nil
}

// ParsePreparedBatteryPacket parses one exact role channel and maps derived
// values through the current template group's actual 42_SOC channel.
func ParsePreparedBatteryPacket(prepared PreparedBatteryChannels, decoded decodedPacket, receivedAt time.Time) (ParsedData, error) {
	if !frameHeaderMatches(prepared.CurrentModule, decoded.Request) {
		return ParsedData{}, errors.New("prepared channel header does not match packet")
	}
	if decoded.RawCID2 != prepared.RawCID2 {
		return ParsedData{}, errors.New("prepared channel CID2 does not match packet")
	}
	receivedAt = time.Unix(receivedAt.Unix(), 0).In(receivedAt.Location())
	var result ParsedData
	switch decoded.RawCID2 {
	case cid42:
		if !model.TtdcRawProfileCompatible(prepared.CurrentModule.ParserProfile) {
			return ParsedData{}, errors.New("CID42 channel does not use a raw-fields profile")
		}
		result = parsePreparedCID42(prepared, decoded.Info, receivedAt)
	case cid44:
		if !model.TtdcRawProfileCompatible(prepared.CurrentModule.ParserProfile) {
			return ParsedData{}, errors.New("CID44 channel does not use a raw-fields profile")
		}
		result = parseCID44(prepared.CurrentModule, prepared.GatewayID, decoded.Info, receivedAt)
	default:
		return ParsedData{}, fmt.Errorf("unsupported CID2: %s", decoded.RawCID2)
	}
	if len(result.Points) == 0 && len(result.StrPoints) == 0 {
		return ParsedData{}, errors.New("prepared channel produced no points")
	}
	debugLogf("[BATTERY-PARSE] group prepared success gatewayID=%s segment=%s groupID=%d role=%s moduleID=%d numericPoints=%d stringPoints=%d\n", prepared.GatewayID, prepared.TopicSegment, prepared.TemplateGroupID, prepared.RawCID2, prepared.CurrentModule.ID, len(result.Points), len(result.StrPoints))
	return result, nil
}

func parsePreparedCID42(prepared PreparedBatteryChannels, info string, ts time.Time) ParsedData {
	module := prepared.CurrentModule
	adr := strings.ToUpper(strings.TrimSpace(module.TtdcConfig.ModuleAdr))
	result := ParsedData{GatewayID: prepared.GatewayID}
	values := make(map[string]float64)
	cellVoltages := make(map[int]float64)
	cellTemperatures := make(map[int]float64)
	skippedPoints := 0
	aiData := DcToAIData{PackNo: prepared.GatewayID, Timestamp: ts.Format("2006-01-02 15:04:05")}

	for _, item := range module.ModuleDatas {
		raw, ok := decodeTransport(info, item.Transport)
		if !ok || item.DataType != 1 {
			skippedPoints++
			continue
		}
		value := raw
		if formula := strings.TrimSpace(item.Formula); formula != "" && formula != "x" {
			calculated, formulaErr := util.ArithmeticNew(raw, formula)
			if formulaErr != nil {
				batterylog.Printf("[BATTERY] formula failed, dataId=%s formula=%s err=%v\n", item.DataId, item.Formula, formulaErr)
				continue
			}
			value = calculated
		}
		result.Points = append(result.Points, Point{DataID: item.DataId, Value: value, Status: 1, TS: ts})
		baseID := strings.TrimPrefix(item.DataId, adr+"_")
		values[baseID] = value
		if index, ok := model.TtdcCID42CellVoltageIndex(baseID); ok {
			cellVoltages[index] = value
		}
		if index, ok := model.TtdcCID42CellTemperatureIndex(baseID); ok {
			cellTemperatures[index] = value
		}
	}
	if skippedPoints > 0 {
		debugLogf("[BATTERY-PARSE] CID42 skipped unavailable points gatewayID=%s ADR=%s skipped=%d INFOChars=%d\n", prepared.GatewayID, adr, skippedPoints, len(info))
	}

	cellLimit, temperatureLimit := configuredCID42Limits(module, adr)
	cellCount := boundedReportedCount(values["cell_count"], cellLimit)
	temperatureCount := boundedReportedCount(values["temperature_count"], temperatureLimit)
	aiData.CellNum = float64(cellCount)
	aiData.TepNum = float64(temperatureCount)
	aiData.BatteryVoltage = values["battery_voltage"]
	aiData.BatteryCurrent = values["battery_current"]
	aiData.BatteryTemperature = values["environment_temperature"]
	aiData.BmsMosTemperature = values["mos_temperature"]
	cellVoltagesComplete := cellCount > 0
	for i := 1; i <= cellCount; i++ {
		if value, ok := cellVoltages[i]; ok {
			aiData.CellVoltages = append(aiData.CellVoltages, value)
		} else {
			cellVoltagesComplete = false
		}
	}
	for i := 1; i <= temperatureCount; i++ {
		if value, ok := cellTemperatures[i]; ok {
			aiData.CellTemperatures = append(aiData.CellTemperatures, value)
		}
	}
	if remaining, ok := values["remaining_capacity"]; ok {
		if total, totalOK := values["total_capacity"]; totalOK && total > 0 {
			aiData.Soc, aiData.DCZRL = remaining, total
		}
	}
	if aiData.DCZRL == 0 {
		if soc, ok := values["pack_soc"]; ok {
			aiData.Soc = soc
		}
	}

	algorithms := make(map[string]algorithmResult)
	if cellVoltagesComplete {
		algorithms = evaluateAI(aiData)
	} else {
		batterylog.Printf("[BATTERY-AI] skipped gatewayID=%s ADR=%s reportedCells=%d decodedCells=%d: cell voltages are incomplete\n", prepared.GatewayID, adr, cellCount, len(aiData.CellVoltages))
	}
	if spread, ok := spreadOfPresent(pointsWithinCount(cellVoltages, cellCount)); ok {
		algorithms["ditoo_yacha"] = numericAlgorithmResult(spread)
	}
	if spread, ok := spreadOfPresent(pointsWithinCount(cellTemperatures, temperatureCount)); ok {
		algorithms["ditoo_wencha"] = numericAlgorithmResult(spread)
	}
	appendMappedAlgorithmResults(&result, prepared.SocModule, algorithms, ts)
	return result
}

func configuredCID42Limits(module model.GatewayModuleTtdc, adr string) (cellLimit, temperatureLimit int) {
	for _, item := range module.ModuleDatas {
		baseID := strings.TrimPrefix(item.DataId, adr+"_")
		if index, ok := model.TtdcCID42CellVoltageIndex(baseID); ok {
			if index > cellLimit {
				cellLimit = index
			}
		}
		if index, ok := model.TtdcCID42CellTemperatureIndex(baseID); ok {
			if index > temperatureLimit {
				temperatureLimit = index
			}
		}
	}
	return cellLimit, temperatureLimit
}

func boundedReportedCount(value float64, maximum int) int {
	count := int(value)
	if count < 0 {
		return 0
	}
	if count > maximum {
		return maximum
	}
	return count
}

func pointsWithinCount(values map[int]float64, count int) map[int]float64 {
	result := make(map[int]float64, count)
	for index, value := range values {
		if index >= 1 && index <= count {
			result[index] = value
		}
	}
	return result
}

func spreadOfPresent(values map[int]float64) (float64, bool) {
	if len(values) < 2 {
		return 0, false
	}
	first := true
	minValue, maxValue := 0.0, 0.0
	for _, value := range values {
		if first {
			minValue, maxValue, first = value, value, false
			continue
		}
		if value < minValue {
			minValue = value
		}
		if value > maxValue {
			maxValue = value
		}
	}
	return mathRound3(maxValue - minValue), true
}

func appendMappedAlgorithmResults(result *ParsedData, socModule model.GatewayModuleTtdc, values map[string]algorithmResult, ts time.Time) {
	adr := strings.ToUpper(strings.TrimSpace(socModule.TtdcConfig.ModuleAdr))
	for _, point := range socModule.ModuleDatas {
		baseDataID := strings.TrimPrefix(point.DataId, adr+"_")
		key, expectedType, supported := model.TtdcAlgorithmResultKey(baseDataID, point.AlgorithmKey)
		if !supported || expectedType != point.DataType {
			continue
		}
		value, ok := values[key]
		if !ok {
			continue
		}
		if point.DataType == 1 && value.Number != nil {
			result.Points = append(result.Points, Point{DataID: point.DataId, Value: *value.Number, Status: 1, TS: ts})
		}
		if point.DataType == 2 && value.Text != nil {
			result.StrPoints = append(result.StrPoints, PointStr{DataID: point.DataId, Value: *value.Text, Status: 1, TS: ts})
		}
	}
}

func ParsePreparedPacket(prepared PreparedModule, decoded decodedPacket, receivedAt time.Time) (ParsedData, error) {
	if !frameHeaderMatches(prepared.Module, decoded.Request) {
		return ParsedData{}, errors.New("prepared channel header does not match packet")
	}
	if decoded.RawCID2 != prepared.RawCID2 {
		return ParsedData{}, errors.New("prepared channel CID2 does not match packet")
	}
	receivedAt = time.Unix(receivedAt.Unix(), 0).In(receivedAt.Location())
	var result ParsedData
	switch prepared.ParserProfile {
	case model.TtdcProfileRawFields:
		switch decoded.RawCID2 {
		case cid42:
			result = parsePreparedCID42(PreparedBatteryChannels{GatewayID: prepared.GatewayID, RawCID2: decoded.RawCID2, CurrentModule: prepared.Module}, decoded.Info, receivedAt)
		case cid44:
			result = parseCID44(prepared.Module, prepared.GatewayID, decoded.Info, receivedAt)
		default:
			return ParsedData{}, fmt.Errorf("raw-fields profile received unsupported CID2 %s", decoded.RawCID2)
		}
	case model.TtdcProfileCID42LithiumSulfur:
		if decoded.RawCID2 != cid42 {
			return ParsedData{}, errors.New("CID42 lithium-sulfur profile received another CID2")
		}
		result = parseCID42(prepared.Module, LithiumSulfurTopic, prepared.GatewayID, decoded.Info, receivedAt)
	case model.TtdcProfileCID42Sodium:
		if decoded.RawCID2 != cid42 {
			return ParsedData{}, errors.New("CID42 sodium profile received another CID2")
		}
		result = parseCID42(prepared.Module, SodiumTopic, prepared.GatewayID, decoded.Info, receivedAt)
	case model.TtdcProfileCID44Common:
		if decoded.RawCID2 != cid44 {
			return ParsedData{}, errors.New("CID44 profile received another CID2")
		}
		result = parseCID44(prepared.Module, prepared.GatewayID, decoded.Info, receivedAt)
	default:
		return ParsedData{}, fmt.Errorf("unsupported parser profile: %s", prepared.ParserProfile)
	}
	if len(result.Points) == 0 && len(result.StrPoints) == 0 {
		return ParsedData{}, errors.New("prepared channel produced no points")
	}
	debugLogf("[BATTERY-PARSE] prepared success gatewayID=%s segment=%s profile=%s moduleID=%d numericPoints=%d stringPoints=%d\n", prepared.GatewayID, prepared.TopicSegment, prepared.ParserProfile, prepared.Module.ID, len(result.Points), len(result.StrPoints))
	return result, nil
}

func firstPacketFrame(packet string) string {
	if idx := strings.IndexByte(packet, '\r'); idx >= 0 {
		return packet[:idx]
	}
	return packet
}

func secondPacketFrame(packet string) string {
	if idx := strings.IndexByte(packet, '\r'); idx >= 0 && idx+1 < len(packet) {
		return packet[idx+1:]
	}
	return ""
}

// Each MQTT battery message is one complete request/response packet. Keep the
// frame extraction local to this message so no cross-message assembler state
// can delay or mix packets from the same gateway.
func normalizeCompletePacket(payload string) (string, error) {
	normalized := normalizeChunk(payload)
	frames := extractFrames(&normalized)
	if len(frames) != 2 || strings.TrimSpace(normalized) != "" {
		return "", fmt.Errorf("expected one complete request/response packet")
	}
	if frames[0][7:9] == "00" || frames[1][7:9] != "00" {
		return "", fmt.Errorf("packet frame order is invalid")
	}
	return frames[0] + "\r" + frames[1], nil
}

func moduleMatchesFrame(module model.GatewayModuleTtdc, topicKind TopicKind, requestFrame string) bool {
	cfg := module.TtdcConfig
	if strings.TrimSpace(cfg.ModuleSoi) != requestFrame[0:1] {
		return false
	}
	if strings.TrimSpace(cfg.ModuleVer) != requestFrame[1:3] {
		return false
	}
	if strings.TrimSpace(cfg.ModuleAdr) != requestFrame[3:5] {
		return false
	}
	if strings.TrimSpace(cfg.ModuleCid1) != requestFrame[5:7] {
		return false
	}
	return cid2Matches(topicKind, requestFrame[7:9], cfg.ModuleCid2)
}

func cid2Matches(topicKind TopicKind, rawCID2 string, configured string) bool {
	value := strings.TrimSpace(configured)
	if value == rawCID2 || strings.EqualFold(value, rawCID2+"H") {
		return topicKind == LithiumSulfurTopic
	}
	switch topicKind {
	case SodiumTopic:
		return value == rawCID2+"_钠电"
	case LithiumSulfurTopic:
		return value == rawCID2+"_锂硫电池"
	default:
		return false
	}
}

func parseCID42(module model.GatewayModuleTtdc, topicKind TopicKind, gatewayID string, info string, ts time.Time) ParsedData {
	adr := strings.TrimSpace(module.TtdcConfig.ModuleAdr)
	result := ParsedData{GatewayID: gatewayID}

	temperatureSlots := 4
	if topicKind == SodiumTopic {
		temperatureSlots = 8
	}
	cellVoltages := make([]float64, 16)
	cellTemperatures := make([]float64, temperatureSlots)
	aiData := DcToAIData{
		PackNo:    gatewayID,
		Timestamp: ts.Format("2006-01-02 15:04:05"),
	}

	for _, item := range module.ModuleDatas {
		raw, ok := decodeTransport(info, item.Transport)
		if !ok || item.DataType != 1 {
			continue
		}
		value := raw
		if strings.TrimSpace(item.Formula) != "" && strings.TrimSpace(item.Formula) != "x" {
			if calculated, err := util.ArithmeticNew(raw, item.Formula); err == nil {
				value = calculated
			} else {
				batterylog.Printf("[BATTERY] formula failed, dataId=%s formula=%s err=%v\n", item.DataId, item.Formula, err)
				continue
			}
		}
		result.Points = append(result.Points, Point{DataID: item.DataId, Value: value, Status: 1, TS: ts})
		fillAIData(adr, item.Name, raw, value, &aiData, cellVoltages, cellTemperatures)
	}

	aiData.CellVoltages = cellVoltages
	aiData.CellTemperatures = cellTemperatures
	result.Points = append(result.Points, Point{DataID: adr + "_ditoo_yacha", Value: maxMinusMin(cellVoltages), Status: 1, TS: ts})
	result.Points = append(result.Points, Point{DataID: adr + "_ditoo_wencha", Value: maxMinusMin(cellTemperatures), Status: 1, TS: ts})
	appendAIResultPoints(&result, adr, aiData, ts)
	return result
}

func parseCID44(module model.GatewayModuleTtdc, gatewayID string, info string, ts time.Time) ParsedData {
	result := ParsedData{GatewayID: gatewayID}
	skippedPoints := 0
	for _, item := range module.ModuleDatas {
		raw, ok := decodeTransport(info, item.Transport)
		if !ok || item.DataType != 1 {
			skippedPoints++
			continue
		}
		value := raw
		if strings.TrimSpace(item.Formula) != "" && strings.TrimSpace(item.Formula) != "x" {
			if calculated, err := util.ArithmeticNew(raw, item.Formula); err == nil {
				value = calculated
			} else {
				batterylog.Printf("[BATTERY] formula failed, dataId=%s formula=%s err=%v\n", item.DataId, item.Formula, err)
				continue
			}
		}
		result.Points = append(result.Points, Point{DataID: item.DataId, Value: value, Status: 1, TS: ts})
	}
	if skippedPoints > 0 {
		debugLogf("[BATTERY-PARSE] CID44 skipped unavailable points gatewayID=%s ADR=%s skipped=%d INFOChars=%d\n", gatewayID, strings.ToUpper(strings.TrimSpace(module.TtdcConfig.ModuleAdr)), skippedPoints, len(info))
	}
	return result
}

func fillAIData(adr string, name string, raw float64, value float64, aiData *DcToAIData, cellVoltages []float64, cellTemperatures []float64) {
	dataName := strings.TrimPrefix(name, adr+"_")
	key := normalizePointName(dataName)
	switch key {
	case "M(单体电池数量)", "单体电池数量":
		aiData.CellNum = raw
	case "N(电芯温度数量)", "电芯温度数量":
		aiData.TepNum = raw
	case "电池总压数据", "总电压":
		aiData.BatteryVoltage = value
	case "电池电流数据", "总电流":
		aiData.BatteryCurrent = value
	case "环境温度数据", "环境温度":
		aiData.BatteryTemperature = value
	case "MOS温度数据", "MOS温度":
		aiData.BmsMosTemperature = value
	case "电池剩余容量":
		aiData.Soc = value
	case "电池总容量":
		aiData.DCZRL = value
	case "电池组荷电状态(SOC)":
		aiData.Soc = value
	}
	if matches := cellVoltagePattern.FindStringSubmatch(key); matches != nil {
		index := atoi(matches[1]) - 1
		if index >= 0 && index < len(cellVoltages) {
			cellVoltages[index] = value
		}
		return
	}
	if matches := cellTemperaturePattern.FindStringSubmatch(key); matches != nil {
		index := atoi(matches[1]) - 1
		if index >= 0 && index < len(cellTemperatures) {
			cellTemperatures[index] = value
		}
	}
}

func normalizePointName(name string) string {
	return nameReplacer.Replace(strings.TrimSpace(name))
}

func atoi(value string) int {
	n := 0
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return n
		}
		n = n*10 + int(value[i]-'0')
	}
	return n
}

func maxMinusMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	maxVal := values[0]
	for _, value := range values {
		if value < minVal {
			minVal = value
		}
		if value > maxVal {
			maxVal = value
		}
	}
	return mathRound3(maxVal - minVal)
}

func mathRound3(value float64) float64 {
	if value >= 0 {
		return float64(int64(value*1000+0.5)) / 1000
	}
	return float64(int64(value*1000-0.5)) / 1000
}
