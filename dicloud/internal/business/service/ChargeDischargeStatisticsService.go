package service

import (
	"context"
	"dicloud/internal/business/client"
	businessdao "dicloud/internal/business/dao"
	businessmodel "dicloud/internal/business/model"
	commondao "dicloud/internal/common/dao"
	"dicloud/internal/common/grpc/tolink_grpc"
	commonmodel "dicloud/internal/common/model"
	commonservice "dicloud/internal/common/service"
	"dicloud/pkg/cache"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	chargeDischargeHistoryBatchSize = 1000
	chargeDischargeHistoryTimeout   = 30 * time.Second
	chargeDischargeMaxDays          = 31
)

type ChargeDischargeSeries struct {
	Code  string    `json:"code"`
	Mark  []string  `json:"mark"`
	Value []float64 `json:"value"`
}

type ChargeDischargeDailyQuality struct {
	MarkDate       string `json:"markDate"`
	EquipmentCount int    `json:"equipmentCount"`
	DischargeValid int    `json:"dischargeValidCount"`
	ChargeValid    int    `json:"chargeValidCount"`
	DischargeMiss  int    `json:"dischargeMissingCount"`
	ChargeMiss     int    `json:"chargeMissingCount"`
	RealtimeCount  int    `json:"realtimeCount"`
	CarriedCount   int    `json:"carriedCount"`
	Realtime       bool   `json:"realtime"`
	Complete       bool   `json:"complete"`
}

type ChargeDischargeStatisticsResult struct {
	Data     []ChargeDischargeSeries       `json:"data"`
	Quality  []ChargeDischargeDailyQuality `json:"quality"`
	Complete bool                          `json:"complete"`
}

type chargeDischargeDay struct {
	MarkDate string
	Cutoff   int64
	Realtime bool
}

type chargeDischargeHistoryNeed struct {
	Key       string
	Equipment businessmodel.Equipment
	Day       chargeDischargeDay
	Code      string
	Points    []chargeDischargePoint
}

type chargeDischargeHistoryRequest struct {
	NeedKey string
	Request *tolink_grpc.GetDataDto
}

func FindChargeDischargeDailyStatistics(ctx *gin.Context, equipments []businessmodel.Equipment, startTime, endTime int64) (ChargeDischargeStatisticsResult, error) {
	days, err := buildChargeDischargeDays(startTime, endTime, time.Now())
	if err != nil {
		return ChargeDischargeStatisticsResult{}, err
	}
	result := newChargeDischargeStatisticsResult(days)
	if len(equipments) == 0 {
		return result, nil
	}

	equipmentIDs := make([]int, 0, len(equipments))
	for _, equipment := range equipments {
		id := int(equipment.ID)
		equipmentIDs = append(equipmentIDs, id)
	}

	historyDates := make([]string, 0, len(days))
	for _, day := range days {
		if !day.Realtime {
			historyDates = append(historyDates, day.MarkDate)
		}
	}
	statisticDAO := commondao.NewStatisticDataDao(ctx)
	records := make(map[string]commonmodel.StatisticData)
	if len(historyDates) > 0 {
		stored, queryErr := statisticDAO.FindEquipmentDailyCumulative(equipmentIDs, chargeDischargeCodes, historyDates)
		if queryErr != nil {
			return ChargeDischargeStatisticsResult{}, queryErr
		}
		mergeChargeDischargeRecords(records, stored)
	}

	pointsByEquipment := resolveChargeDischargePoints(ctx, equipments)
	needs := findChargeDischargeHistoryNeeds(equipments, days, pointsByEquipment, records)
	if len(needs) > 0 {
		backfilled := backfillChargeDischargeHistory(ctx, needs, statisticDAO)
		mergeChargeDischargeRecords(records, backfilled)
	}

	realtimeReadings, realtimeErr := loadChargeDischargeRealtime(ctx, pointsByEquipment)
	if realtimeErr != nil {
		log.Println("[ChargeDischargeStatistics] realtime query failed, use historical carry-forward:", realtimeErr)
	}

	result = aggregateChargeDischargeStatistics(days, equipments, pointsByEquipment, records, realtimeReadings)
	return result, nil
}

func buildChargeDischargeDays(startTime, endTime int64, now time.Time) ([]chargeDischargeDay, error) {
	if startTime <= 0 || endTime <= 0 || endTime < startTime {
		return nil, errors.New("invalid statistic time range")
	}
	location := now.Location()
	start := time.Unix(startTime, 0).In(location)
	end := time.Unix(endTime, 0).In(location)
	startDay := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, location)
	endDay := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, location)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location)
	if endDay.After(today) {
		endDay = today
	}
	if startDay.After(endDay) {
		return nil, errors.New("statistic time range is in the future")
	}
	days := make([]chargeDischargeDay, 0, 7)
	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		if len(days) >= chargeDischargeMaxDays {
			return nil, fmt.Errorf("statistic time range exceeds %d days", chargeDischargeMaxDays)
		}
		days = append(days, chargeDischargeDay{
			MarkDate: day.Format("2006-01-02"),
			Cutoff:   day.AddDate(0, 0, 1).Add(-time.Second).Unix(),
			Realtime: day.Equal(today),
		})
	}
	return days, nil
}

func newChargeDischargeStatisticsResult(days []chargeDischargeDay) ChargeDischargeStatisticsResult {
	marks := make([]string, len(days))
	values := make([]float64, len(days))
	quality := make([]ChargeDischargeDailyQuality, len(days))
	for index, day := range days {
		marks[index] = day.MarkDate
		quality[index] = ChargeDischargeDailyQuality{MarkDate: day.MarkDate, Realtime: day.Realtime, Complete: true}
	}
	return ChargeDischargeStatisticsResult{
		Data: []ChargeDischargeSeries{
			{Code: ChargeDischargeCodeDischarge, Mark: append([]string(nil), marks...), Value: append([]float64(nil), values...)},
			{Code: ChargeDischargeCodeCharge, Mark: append([]string(nil), marks...), Value: append([]float64(nil), values...)},
		},
		Quality:  quality,
		Complete: true,
	}
}

func findChargeDischargeHistoryNeeds(equipments []businessmodel.Equipment, days []chargeDischargeDay, pointsByEquipment map[int]map[string][]chargeDischargePoint, records map[string]commonmodel.StatisticData) []chargeDischargeHistoryNeed {
	needs := make([]chargeDischargeHistoryNeed, 0)
	for _, day := range days {
		if day.Realtime {
			continue
		}
		for _, equipment := range equipments {
			for _, code := range chargeDischargeCodes {
				key := chargeDischargeRecordKey(int(equipment.ID), day.MarkDate, code)
				if _, exists := records[key]; exists {
					continue
				}
				needs = append(needs, chargeDischargeHistoryNeed{
					Key:       key,
					Equipment: equipment,
					Day:       day,
					Code:      code,
					Points:    pointsByEquipment[int(equipment.ID)][code],
				})
			}
		}
	}
	return needs
}

func backfillChargeDischargeHistory(ctx *gin.Context, needs []chargeDischargeHistoryNeed, statisticDAO *commondao.StatisticDataDao) []commonmodel.StatisticData {
	needs = filterCachedChargeDischargeHistoryMisses(ctx, needs)
	requestItems := make([]chargeDischargeHistoryRequest, 0)
	dbName := ""
	if value, exists := ctx.Get("dbName"); exists {
		dbName = fmt.Sprint(value)
	}
	needByKey := make(map[string]chargeDischargeHistoryNeed, len(needs))
	for _, need := range needs {
		needByKey[need.Key] = need
		if len(need.Points) == 0 {
			cacheChargeDischargeHistoryMiss(ctx, need, 6*time.Hour)
			continue
		}
		for _, point := range need.Points {
			requestItems = append(requestItems, chargeDischargeHistoryRequest{
				NeedKey: need.Key,
				Request: &tolink_grpc.GetDataDto{
					GatewayId: point.GatewayID,
					DataId:    point.DataID,
					CurTime:   need.Day.Cutoff,
					DbName:    dbName,
				},
			})
		}
	}
	if len(requestItems) == 0 {
		return nil
	}

	baseContext := context.Background()
	if ctx.Request != nil {
		baseContext = ctx.Request.Context()
	}
	queryContext, cancel := context.WithTimeout(baseContext, chargeDischargeHistoryTimeout)
	defer cancel()
	bestByNeed := make(map[string]*tolink_grpc.SequenceData)
	queriedNeeds := make(map[string]struct{})
	for start := 0; start < len(requestItems); start += chargeDischargeHistoryBatchSize {
		end := start + chargeDischargeHistoryBatchSize
		if end > len(requestItems) {
			end = len(requestItems)
		}
		requests := make([]*tolink_grpc.GetDataDto, end-start)
		for index := start; index < end; index++ {
			requests[index-start] = requestItems[index].Request
		}
		response, err := client.GetTolinkLastSequenceDataBatch(queryContext, requests)
		if err != nil {
			log.Println("[ChargeDischargeStatistics] batch history query failed:", err)
			break
		}
		if response == nil || response.Code != 200 {
			log.Println("[ChargeDischargeStatistics] batch history query rejected")
			break
		}
		for index, item := range response.Data {
			if index >= len(requests) {
				continue
			}
			needKey := requestItems[start+index].NeedKey
			queriedNeeds[needKey] = struct{}{}
			if item == nil || item.Error != "" || item.SequenceData == nil || item.SequenceData.Status != 1 {
				continue
			}
			if saved := bestByNeed[needKey]; saved == nil || item.SequenceData.CreateTime > saved.CreateTime {
				bestByNeed[needKey] = item.SequenceData
			}
		}
	}
	for needKey := range queriedNeeds {
		if bestByNeed[needKey] == nil {
			cacheChargeDischargeHistoryMiss(ctx, needByKey[needKey], 15*time.Minute)
		}
	}

	backfilled := make([]commonmodel.StatisticData, 0, len(bestByNeed))
	for needKey, sequence := range bestByNeed {
		need := needByKey[needKey]
		statisticTime := sequence.CreateTime
		if need.Day.Cutoff-sequence.CreateTime > int64(time.Hour/time.Second) {
			statisticTime = need.Day.Cutoff
		}
		data := commonmodel.StatisticData{
			SiteId:             need.Equipment.SiteId,
			EquipmentId:        int(need.Equipment.ID),
			CorpId:             need.Equipment.CorpId,
			DataSourceType:     3,
			TargetType:         0,
			StatisticCode:      need.Code,
			StatisticName:      need.Code,
			StartStatisticTime: statisticTime,
			EndStatisticTime:   statisticTime,
			EndValue:           sequence.Val,
			StatisticValue:     sequence.Val,
			StatisticValueType: 0,
			StatisticDateType:  "daily",
			MarkDate:           need.Day.MarkDate,
			CreateTime:         time.Now().Unix(),
		}
		token, locked := commonservice.LockStatisticDailyData(ctx, &data, 3*time.Second)
		if locked {
			if err := statisticDAO.SaveEquipmentDailyCumulative(data); err != nil {
				log.Printf("[ChargeDischargeStatistics] save history failed equipment=%d date=%s code=%s err=%v", data.EquipmentId, data.MarkDate, data.StatisticCode, err)
			}
			commonservice.UnlockStatisticDailyData(ctx, &data, token)
		}
		backfilled = append(backfilled, data)
	}
	return backfilled
}

func filterCachedChargeDischargeHistoryMisses(ctx *gin.Context, needs []chargeDischargeHistoryNeed) []chargeDischargeHistoryNeed {
	if len(needs) == 0 {
		return needs
	}
	redisClient := cache.GetRedisClient(ctx)
	if redisClient == nil {
		return needs
	}
	keys := make([]string, len(needs))
	for index, need := range needs {
		keys[index] = chargeDischargeHistoryMissKey(need)
	}
	values, err := redisClient.MGet(keys...).Result()
	if err != nil {
		return needs
	}
	filtered := make([]chargeDischargeHistoryNeed, 0, len(needs))
	for index, need := range needs {
		if index < len(values) && values[index] != nil {
			continue
		}
		filtered = append(filtered, need)
	}
	return filtered
}

func cacheChargeDischargeHistoryMiss(ctx *gin.Context, need chargeDischargeHistoryNeed, expiration time.Duration) {
	redisClient := cache.GetRedisClient(ctx)
	if redisClient == nil {
		return
	}
	if err := redisClient.Set(chargeDischargeHistoryMissKey(need), "1", expiration).Err(); err != nil {
		log.Println("[ChargeDischargeStatistics] cache history miss failed:", err)
	}
}

func chargeDischargeHistoryMissKey(need chargeDischargeHistoryNeed) string {
	code := "charge"
	if need.Code == ChargeDischargeCodeDischarge {
		code = "discharge"
	}
	return fmt.Sprintf("statistic:charge-discharge:history-miss:%d:%s:%s", need.Equipment.ID, need.Day.MarkDate, code)
}

func loadChargeDischargeRealtime(ctx *gin.Context, pointsByEquipment map[int]map[string][]chargeDischargePoint) (map[string]businessdao.ChargeDischargeRealtimeReading, error) {
	candidates := make(map[string][]string)
	seen := make(map[string]struct{})
	for _, pointsByCode := range pointsByEquipment {
		for _, points := range pointsByCode {
			for _, point := range points {
				key := point.GatewayID + "\x00" + point.DataID
				if _, exists := seen[key]; exists {
					continue
				}
				seen[key] = struct{}{}
				candidates[point.GatewayID] = append(candidates[point.GatewayID], point.DataID)
			}
		}
	}
	data, err := businessdao.LoadChargeDischargeRealtimeData(ctx, candidates)
	if err != nil {
		return nil, err
	}
	result := make(map[string]businessdao.ChargeDischargeRealtimeReading)
	for gatewayID, readings := range data {
		for dataID, reading := range readings {
			result[gatewayID+"\x00"+dataID] = reading
		}
	}
	return result, nil
}

func aggregateChargeDischargeStatistics(days []chargeDischargeDay, equipments []businessmodel.Equipment, pointsByEquipment map[int]map[string][]chargeDischargePoint, records map[string]commonmodel.StatisticData, realtimeReadings map[string]businessdao.ChargeDischargeRealtimeReading) ChargeDischargeStatisticsResult {
	result := newChargeDischargeStatisticsResult(days)
	latestHistory := latestChargeDischargeHistory(records)
	for dayIndex, day := range days {
		quality := &result.Quality[dayIndex]
		quality.EquipmentCount = len(equipments)
		validCounts := map[string]int{}
		for _, equipment := range equipments {
			equipmentID := int(equipment.ID)
			for codeIndex, code := range chargeDischargeCodes {
				value := float64(0)
				valid := false
				if day.Realtime {
					if reading, found := latestChargeDischargeRealtimeReading(pointsByEquipment[equipmentID][code], realtimeReadings); found {
						value = reading.Value
						valid = true
						quality.RealtimeCount++
					} else if history, found := latestHistory[chargeDischargeEquipmentCodeKey(equipmentID, code)]; found {
						value = history.StatisticValue
						valid = true
						quality.CarriedCount++
					}
				} else if record, found := records[chargeDischargeRecordKey(equipmentID, day.MarkDate, code)]; found {
					value = record.StatisticValue
					valid = true
				}
				if valid {
					result.Data[codeIndex].Value[dayIndex] += value
					validCounts[code]++
				}
			}
		}
		quality.DischargeValid = validCounts[ChargeDischargeCodeDischarge]
		quality.ChargeValid = validCounts[ChargeDischargeCodeCharge]
		quality.DischargeMiss = len(equipments) - quality.DischargeValid
		quality.ChargeMiss = len(equipments) - quality.ChargeValid
		quality.Complete = quality.DischargeMiss == 0 && quality.ChargeMiss == 0
		if !quality.Complete {
			result.Complete = false
		}
	}
	return result
}

func latestChargeDischargeRealtimeReading(points []chargeDischargePoint, readings map[string]businessdao.ChargeDischargeRealtimeReading) (businessdao.ChargeDischargeRealtimeReading, bool) {
	var latest businessdao.ChargeDischargeRealtimeReading
	found := false
	for _, point := range points {
		reading, exists := readings[point.GatewayID+"\x00"+point.DataID]
		if exists && (!found || reading.Timestamp > latest.Timestamp) {
			latest = reading
			found = true
		}
	}
	return latest, found
}

func latestChargeDischargeHistory(records map[string]commonmodel.StatisticData) map[string]commonmodel.StatisticData {
	result := make(map[string]commonmodel.StatisticData)
	for _, record := range records {
		key := chargeDischargeEquipmentCodeKey(record.EquipmentId, record.StatisticCode)
		if saved, exists := result[key]; !exists || record.MarkDate > saved.MarkDate || (record.MarkDate == saved.MarkDate && record.EndStatisticTime > saved.EndStatisticTime) {
			result[key] = record
		}
	}
	return result
}

func mergeChargeDischargeRecords(target map[string]commonmodel.StatisticData, records []commonmodel.StatisticData) {
	for _, record := range records {
		key := chargeDischargeRecordKey(record.EquipmentId, record.MarkDate, record.StatisticCode)
		if saved, exists := target[key]; !exists || record.EndStatisticTime > saved.EndStatisticTime || (record.EndStatisticTime == saved.EndStatisticTime && record.ID > saved.ID) {
			target[key] = record
		}
	}
}

func chargeDischargeRecordKey(equipmentID int, markDate, code string) string {
	return fmt.Sprintf("%d\x00%s\x00%s", equipmentID, markDate, code)
}

func chargeDischargeEquipmentCodeKey(equipmentID int, code string) string {
	return fmt.Sprintf("%d\x00%s", equipmentID, code)
}
