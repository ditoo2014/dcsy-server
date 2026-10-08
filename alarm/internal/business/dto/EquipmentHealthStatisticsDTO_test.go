package dto

import "testing"

func TestBuildEquipmentHealthStatistics(t *testing.T) {
	statistics := BuildEquipmentHealthStatistics(81, map[string]int64{
		"SOH优": 2,
		"SOH良": 69,
		"SOH中": 1,
	}, 1787883000)

	if statistics.ValidCount != 72 || statistics.UnknownCount != 9 {
		t.Fatalf("valid=%d unknown=%d, want valid=72 unknown=9", statistics.ValidCount, statistics.UnknownCount)
	}
	if statistics.CoverageRate != 88.89 {
		t.Fatalf("coverageRate=%v, want 88.89", statistics.CoverageRate)
	}
	wantPercentages := []float64{2.78, 95.83, 1.39, 0}
	for index, level := range statistics.Levels {
		if level.Percentage != wantPercentages[index] {
			t.Fatalf("level %s percentage=%v, want %v", level.Code, level.Percentage, wantPercentages[index])
		}
	}
}

func TestBuildEquipmentHealthStatisticsReturnsAllZeroLevels(t *testing.T) {
	statistics := BuildEquipmentHealthStatistics(0, nil, 0)

	if len(statistics.Levels) != 4 {
		t.Fatalf("level count=%d, want 4", len(statistics.Levels))
	}
	for _, level := range statistics.Levels {
		if level.Count != 0 || level.Percentage != 0 {
			t.Fatalf("level %+v is not zero", level)
		}
	}
}
