package service

import (
	"dicloud/internal/business/client"
	businessmodel "dicloud/internal/business/model"
	"dicloud/internal/common/dto"
	"dicloud/internal/common/grpc/tolink_grpc"
	tolinkdao "dicloud/internal/tolink/dao"
	tolinkmodel "dicloud/internal/tolink/model"
	"dicloud/util/pointid"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	ChargeDischargeCodeDischarge = "累计放电量"
	ChargeDischargeCodeCharge    = "累计充电量"
)

var chargeDischargeCodes = []string{
	ChargeDischargeCodeDischarge,
	ChargeDischargeCodeCharge,
}

var chargeDischargePointDefinitions = []struct {
	DataID string
	Code   string
}{
	{DataID: pointid.TotalDischargeCapacity, Code: ChargeDischargeCodeDischarge},
	{DataID: pointid.TotalChargeCapacity, Code: ChargeDischargeCodeCharge},
}

type chargeDischargePoint struct {
	EquipmentID int
	GatewayID   string
	DataID      string
	Code        string
}

func resolveChargeDischargePoints(ctx *gin.Context, equipments []businessmodel.Equipment) map[int]map[string][]chargeDischargePoint {
	result := make(map[int]map[string][]chargeDischargePoint, len(equipments))
	pointResolver := newStatisticDataPointResolver(ctx)
	streams := loadEquipmentStatisticStreams(ctx, equipments)
	for _, equipment := range equipments {
		points := resolveChargeDischargeStreamPoints(equipment, streams[strings.TrimSpace(equipment.ApiKey)])
		if strings.TrimSpace(equipment.EquipmentCode) != "" {
			// 充电和放电分别补查，避免只绑定其中一项时漏掉另一项。
			foundCodes := make(map[string]bool)
			for _, point := range points {
				foundCodes[point.Code] = true
			}
			standardPoints := make([]statisticDataPoint, 0, len(chargeDischargePointDefinitions))
			for _, definition := range chargeDischargePointDefinitions {
				if !foundCodes[definition.Code] {
					standardPoints = append(standardPoints, statisticDataPoint{DataID: definition.DataID, Code: dto.DP_STATISTIC_CODE_MAP[definition.DataID]})
				}
			}
			if len(standardPoints) > 0 {
				for _, point := range pointResolver.resolve(equipment.EquipmentCode, standardPoints) {
					points = append(points, chargeDischargePoint{
						EquipmentID: int(equipment.ID),
						GatewayID:   equipment.EquipmentCode,
						DataID:      point.DataID,
						Code:        point.Code,
					})
				}
			}
		}
		for _, point := range deduplicateChargeDischargePoints(points) {
			if result[point.EquipmentID] == nil {
				result[point.EquipmentID] = make(map[string][]chargeDischargePoint)
			}
			result[point.EquipmentID][point.Code] = append(result[point.EquipmentID][point.Code], point)
		}
	}
	return result
}

func resolveChargeDischargeStreamPoints(equipment businessmodel.Equipment, stream *tolink_grpc.Stream) []chargeDischargePoint {
	if stream == nil {
		return nil
	}
	points := make([]chargeDischargePoint, 0)
	// 每个指标独立优先选英文 ID；只有没有对应英文点时才沿用已有名称绑定。
	englishCodes := make(map[string]bool)
	for _, label := range stream.List {
		if label != nil && strings.TrimSpace(label.GatewayId) != "" {
			if code := chargeDischargeCodeForPoint(label.DataId); code != "" {
				englishCodes[code] = true
			}
		}
	}
	for _, label := range stream.List {
		if label == nil {
			continue
		}
		code := chargeDischargeCodeForPoint(label.DataId)
		if code == "" {
			code = chargeDischargeCodeForPoint(label.DataId, label.Name, label.DataName)
			if englishCodes[code] {
				continue
			}
		}
		if code == "" || strings.TrimSpace(label.GatewayId) == "" || strings.TrimSpace(label.DataId) == "" {
			continue
		}
		points = append(points, chargeDischargePoint{
			EquipmentID: int(equipment.ID),
			GatewayID:   strings.TrimSpace(label.GatewayId),
			DataID:      strings.TrimSpace(label.DataId),
			Code:        code,
		})
	}
	return points
}

func loadEquipmentStatisticStreams(ctx *gin.Context, equipments []businessmodel.Equipment) map[string]*tolink_grpc.Stream {
	result := make(map[string]*tolink_grpc.Stream)
	apiKeys := make([]string, 0, len(equipments))
	for _, equipment := range equipments {
		apiKey := strings.TrimSpace(equipment.ApiKey)
		if apiKey == "" {
			continue
		}
		if _, exists := result[apiKey]; exists {
			continue
		}
		result[apiKey] = nil
		apiKeys = append(apiKeys, apiKey)
	}
	streamDAO := tolinkdao.NewStreamDAO(ctx)
	streams, err := streamDAO.FindByApiKeys(apiKeys)
	if err == nil {
		for _, stream := range streams {
			result[strings.TrimSpace(stream.ApiKey)] = tolinkStreamGRPCData(stream)
		}
	}
	for _, apiKey := range apiKeys {
		if result[apiKey] != nil {
			continue
		}
		response, fallbackErr := client.ExistApikey(ctx, &tolinkmodel.Stream{ApiKey: apiKey})
		if fallbackErr == nil && response != nil && response.Code == 200 {
			result[apiKey] = response.Stream
		}
	}
	return result
}

func findEquipmentStatisticStream(ctx *gin.Context, apiKey string) (*tolink_grpc.Stream, bool) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return nil, false
	}
	streamDAO := tolinkdao.NewStreamDAO(ctx)
	stream, exists := streamDAO.ExistApiKey(tolinkmodel.Stream{ApiKey: apiKey})
	if exists && stream.ID != 0 {
		return tolinkStreamGRPCData(stream), true
	}
	response, err := client.ExistApikey(ctx, &tolinkmodel.Stream{ApiKey: apiKey})
	if err != nil || response == nil || response.Code != 200 || response.Stream == nil {
		return nil, false
	}
	return response.Stream, true
}

// chargeDischargeCodeForPoint 优先识别英文 ID；可按已有标签名称读取实际绑定，不改写 ID。
func chargeDischargeCodeForPoint(dataID string, names ...string) string {
	normalizedID := strings.ToLower(strings.TrimSpace(dataID))
	for _, definition := range chargeDischargePointDefinitions {
		if pointid.Matches(normalizedID, definition.DataID) {
			return definition.Code
		}
	}
	for _, definition := range chargeDischargePointDefinitions {
		for _, name := range names {
			if matchesCumulativeCapacityName(definition.DataID, name) {
				return definition.Code
			}
		}
	}
	return ""
}

func deduplicateChargeDischargePoints(points []chargeDischargePoint) []chargeDischargePoint {
	result := make([]chargeDischargePoint, 0, len(points))
	seen := make(map[string]struct{}, len(points))
	for _, point := range points {
		key := point.Code + "\x00" + point.GatewayID + "\x00" + point.DataID
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, point)
	}
	return result
}
