package dao

import (
	"alarm/internal/business/dto"
	commonmodel "alarm/internal/common/model"
	"alarm/pkg/cache"
	"alarm/util/gormutil"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis"
	"gorm.io/gorm"
)

type equipmentHealthRealtimeLoader func(candidates map[string][]string) (map[string]map[string]interface{}, error)

type EquipmentHealthStatisticsDAO struct {
	db               *gorm.DB
	loadRealtimeData equipmentHealthRealtimeLoader
}

type equipmentHealthEquipment struct {
	ID int64 `gorm:"column:id"`
}

type equipmentHealthPoint struct {
	EquipmentID int
	GatewayID   string
	DataID      string
}

type equipmentHealthRealtimeReading struct {
	SOH       float64
	Timestamp int64
}

func NewEquipmentHealthStatisticsDAO(ctx *gin.Context) *EquipmentHealthStatisticsDAO {
	redisClient := cache.GetRedisClient(ctx)
	return &EquipmentHealthStatisticsDAO{
		db: gormutil.GetGormDB(ctx, ""),
		loadRealtimeData: func(candidates map[string][]string) (map[string]map[string]interface{}, error) {
			return loadEquipmentHealthRealtimeData(redisClient, candidates)
		},
	}
}

// FindOnlineEquipmentIDs returns authorized batteries currently on the network.
func (dao *EquipmentHealthStatisticsDAO) FindOnlineEquipmentIDs(corpIDs []int) ([]int64, error) {
	if corpIDs != nil && len(corpIDs) == 0 {
		return []int64{}, nil
	}
	if dao.db == nil {
		return nil, errors.New("business database is nil")
	}

	query := dao.db.Table("equipment AS e").
		Select("e.id").
		Where("e.position_status = ?", 3)
	if corpIDs != nil {
		query = query.Where("e.corp_id IN ?", corpIDs)
	}

	var equipments []equipmentHealthEquipment
	if err := query.Order("e.id").Scan(&equipments).Error; err != nil {
		return nil, err
	}

	equipmentIDs := make([]int64, 0, len(equipments))
	for _, equipment := range equipments {
		equipmentIDs = append(equipmentIDs, equipment.ID)
	}
	return equipmentIDs, nil
}

// FindRealtimeStatistics reads SOH from the exact gateway/data-point mappings
// used by the health-status page. It never derives a gateway ID from the
// business battery code and never falls back to snapshots or historical data.
func (dao *EquipmentHealthStatisticsDAO) FindRealtimeStatistics(totalCount int64, marktriggers []commonmodel.GeneralMarktrigger) (dto.EquipmentHealthStatistics, error) {
	pointsByEquipment, candidates := equipmentHealthPoints(marktriggers)
	if len(candidates) == 0 {
		return dto.BuildEquipmentHealthStatistics(totalCount, nil, 0), nil
	}
	if dao.loadRealtimeData == nil {
		return dto.EquipmentHealthStatistics{}, errors.New("realtime data loader is nil")
	}

	realtimeData, err := dao.loadRealtimeData(candidates)
	if err != nil {
		return dto.EquipmentHealthStatistics{}, fmt.Errorf("load realtime SOH data: %w", err)
	}

	levelCounts := make(map[string]int64, len(commonmodel.StatisticHealthCodeArr))
	var updatedAt int64
	for _, points := range pointsByEquipment {
		fields := make(map[string]interface{})
		for _, point := range points {
			for dataID, value := range realtimeData[point.GatewayID] {
				if isEquipmentHealthSOHDataID(dataID) {
					fields[dataID] = value
				}
			}
		}
		reading, valid := selectEquipmentHealthRealtimeReading(fields)
		if !valid {
			continue
		}

		levelCounts[classifyEquipmentHealthSOH(reading.SOH)]++
		if reading.Timestamp > updatedAt {
			updatedAt = reading.Timestamp
		}
	}

	statistics := dto.BuildEquipmentHealthStatistics(totalCount, levelCounts, updatedAt)
	log.Printf("[EquipmentHealthStatistics] realtime points=%d valid=%d unknown=%d updatedAt=%d", len(pointsByEquipment), statistics.ValidCount, statistics.UnknownCount, statistics.UpdatedAt)
	return statistics, nil
}

func equipmentHealthPoints(marktriggers []commonmodel.GeneralMarktrigger) (map[int][]equipmentHealthPoint, map[string][]string) {
	pointsByEquipment := make(map[int][]equipmentHealthPoint)
	candidates := make(map[string][]string)
	seen := make(map[string]struct{})

	for _, marktrigger := range marktriggers {
		parts := strings.Fields(marktrigger.Marktrigger)
		if marktrigger.EquipmentId == 0 || len(parts) != 2 {
			continue
		}
		gatewayID := strings.TrimSpace(parts[0])
		dataID := strings.TrimSpace(parts[1])
		if gatewayID == "" || !isEquipmentHealthSOHDataID(dataID) {
			continue
		}

		pointKey := fmt.Sprintf("%d\x00%s\x00%s", marktrigger.EquipmentId, gatewayID, dataID)
		if _, duplicate := seen[pointKey]; duplicate {
			continue
		}
		seen[pointKey] = struct{}{}
		pointsByEquipment[marktrigger.EquipmentId] = append(pointsByEquipment[marktrigger.EquipmentId], equipmentHealthPoint{
			EquipmentID: marktrigger.EquipmentId,
			GatewayID:   gatewayID,
			DataID:      dataID,
		})

		candidateKey := gatewayID + "\x00" + dataID
		if _, duplicate := seen[candidateKey]; duplicate {
			continue
		}
		seen[candidateKey] = struct{}{}
		candidates[gatewayID] = append(candidates[gatewayID], dataID)
	}

	return pointsByEquipment, candidates
}

func loadEquipmentHealthRealtimeData(redisClient *redis.Client, candidates map[string][]string) (map[string]map[string]interface{}, error) {
	if len(candidates) == 0 {
		return map[string]map[string]interface{}{}, nil
	}
	if redisClient == nil {
		return nil, errors.New("Redis client is nil")
	}

	gatewayIDs := make([]string, 0, len(candidates))
	for gatewayID := range candidates {
		gatewayIDs = append(gatewayIDs, gatewayID)
	}
	sort.Strings(gatewayIDs)

	pipeline := redisClient.Pipeline()
	defer pipeline.Close()
	allCommands := make(map[string]*redis.StringStringMapCmd, len(gatewayIDs))
	configuredCommands := make(map[string]*redis.SliceCmd, len(gatewayIDs))
	for _, gatewayID := range gatewayIDs {
		// Read the complete realtime hash. The data-pool mapping can contain
		// legacy alg_soh while the gateway publishes a channel-prefixed point
		// such as 01_alg_soh, so HMGET of one configured ID is insufficient.
		allCommands[gatewayID] = pipeline.HGetAll("xc:gateway:" + gatewayID)
		// Keep a compatibility fallback for Redis implementations that do not
		// fully support HGETALL but do support the HGET/HMGET path used by the
		// health-status page.
		configuredCommands[gatewayID] = pipeline.HMGet("xc:gateway:"+gatewayID, candidates[gatewayID]...)
	}
	_, pipelineErr := pipeline.Exec()

	result := make(map[string]map[string]interface{}, len(allCommands))
	for gatewayID, allCommand := range allCommands {
		values, allErr := allCommand.Result()
		if allErr != nil && !errors.Is(allErr, redis.Nil) {
			values = nil
		}
		result[gatewayID] = make(map[string]interface{}, len(values))
		for dataID, value := range values {
			result[gatewayID][dataID] = value
		}

		configuredValues, configuredErr := configuredCommands[gatewayID].Result()
		if configuredErr != nil && !errors.Is(configuredErr, redis.Nil) {
			if len(result[gatewayID]) == 0 {
				if pipelineErr != nil && !errors.Is(pipelineErr, redis.Nil) {
					return nil, pipelineErr
				}
				return nil, configuredErr
			}
		} else {
			for index, dataID := range candidates[gatewayID] {
				if index < len(configuredValues) && configuredValues[index] != nil {
					if _, exists := result[gatewayID][dataID]; !exists {
						result[gatewayID][dataID] = configuredValues[index]
					}
				}
			}
		}
	}
	return result, nil
}

func selectEquipmentHealthRealtimeReading(fields map[string]interface{}) (equipmentHealthRealtimeReading, bool) {
	var prefixed equipmentHealthRealtimeReading
	var legacy equipmentHealthRealtimeReading
	prefixedValid := false
	legacyValid := false

	for dataID, raw := range fields {
		if !isEquipmentHealthSOHDataID(dataID) {
			continue
		}
		reading, valid := parseEquipmentHealthRealtimeReading(raw)
		if !valid {
			continue
		}

		if strings.EqualFold(strings.TrimSpace(dataID), "alg_soh") {
			if !legacyValid || reading.Timestamp > legacy.Timestamp {
				legacy = reading
				legacyValid = true
			}
			continue
		}
		if !prefixedValid || reading.Timestamp > prefixed.Timestamp {
			prefixed = reading
			prefixedValid = true
		}
	}

	if prefixedValid {
		return prefixed, true
	}
	return legacy, legacyValid
}

func parseEquipmentHealthRealtimeReading(raw interface{}) (equipmentHealthRealtimeReading, bool) {
	var text string
	switch value := raw.(type) {
	case string:
		text = value
	case []byte:
		text = string(value)
	default:
		return equipmentHealthRealtimeReading{}, false
	}

	parts := strings.Fields(text)
	if len(parts) < 3 {
		return equipmentHealthRealtimeReading{}, false
	}
	status, err := strconv.Atoi(parts[0])
	if err != nil || status != 1 {
		return equipmentHealthRealtimeReading{}, false
	}
	timestamp, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil || timestamp <= 0 {
		return equipmentHealthRealtimeReading{}, false
	}
	valueText := strings.TrimSuffix(strings.TrimSpace(strings.Join(parts[1:len(parts)-1], " ")), "%")
	soh, err := strconv.ParseFloat(valueText, 64)
	if err != nil || math.IsNaN(soh) || math.IsInf(soh, 0) || soh < 0 {
		return equipmentHealthRealtimeReading{}, false
	}

	return equipmentHealthRealtimeReading{SOH: soh, Timestamp: timestamp}, true
}

func isEquipmentHealthSOHDataID(dataID string) bool {
	dataID = strings.ToLower(strings.TrimSpace(dataID))
	return dataID == "alg_soh" || strings.HasSuffix(dataID, "_alg_soh")
}

func classifyEquipmentHealthSOH(soh float64) string {
	switch {
	case soh > 90:
		return commonmodel.StatisticHealthCodeExcellent
	case soh >= 80:
		return commonmodel.StatisticHealthCodeGood
	case soh >= 60:
		return commonmodel.StatisticHealthCodeFair
	default:
		return commonmodel.StatisticHealthCodePoor
	}
}
