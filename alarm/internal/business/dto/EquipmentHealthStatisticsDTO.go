package dto

import "math"

const EquipmentHealthPercentageBase = "validCount"

type EquipmentHealthStatisticsRequest struct {
	CorpId *int `json:"corpId"`
}

type EquipmentHealthLevel struct {
	Code       string  `json:"code"`
	Name       string  `json:"name"`
	Count      int64   `json:"count"`
	Percentage float64 `json:"percentage"`
}

type EquipmentHealthStatistics struct {
	TotalCount     int64                  `json:"totalCount"`
	ValidCount     int64                  `json:"validCount"`
	UnknownCount   int64                  `json:"unknownCount"`
	CoverageRate   float64                `json:"coverageRate"`
	PercentageBase string                 `json:"percentageBase"`
	UpdatedAt      int64                  `json:"updatedAt"`
	Levels         []EquipmentHealthLevel `json:"levels"`
}

var equipmentHealthLevelDefinitions = []EquipmentHealthLevel{
	{Code: "SOH优", Name: "健康度优"},
	{Code: "SOH良", Name: "健康度良"},
	{Code: "SOH中", Name: "健康度中"},
	{Code: "SOH差", Name: "健康度差"},
}

func BuildEquipmentHealthStatistics(totalCount int64, levelCounts map[string]int64, updatedAt int64) EquipmentHealthStatistics {
	statistics := EquipmentHealthStatistics{
		TotalCount:     totalCount,
		PercentageBase: EquipmentHealthPercentageBase,
		UpdatedAt:      updatedAt,
		Levels:         make([]EquipmentHealthLevel, len(equipmentHealthLevelDefinitions)),
	}

	for index, definition := range equipmentHealthLevelDefinitions {
		level := definition
		level.Count = levelCounts[level.Code]
		statistics.ValidCount += level.Count
		statistics.Levels[index] = level
	}

	statistics.UnknownCount = statistics.TotalCount - statistics.ValidCount
	if statistics.UnknownCount < 0 {
		statistics.UnknownCount = 0
	}
	if statistics.TotalCount > 0 {
		statistics.CoverageRate = roundEquipmentHealthPercentage(float64(statistics.ValidCount) / float64(statistics.TotalCount) * 100)
	}
	if statistics.ValidCount > 0 {
		for index := range statistics.Levels {
			statistics.Levels[index].Percentage = roundEquipmentHealthPercentage(float64(statistics.Levels[index].Count) / float64(statistics.ValidCount) * 100)
		}
	}

	return statistics
}

func roundEquipmentHealthPercentage(value float64) float64 {
	return math.Round(value*100) / 100
}
