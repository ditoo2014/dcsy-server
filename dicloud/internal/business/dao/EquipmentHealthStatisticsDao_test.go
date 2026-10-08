//go:build cgo

package dao

import (
	commonmodel "dicloud/internal/common/model"
	"fmt"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFindRealtimeEquipmentHealthStatisticsUsesCurrentPrefixedSOH(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=private"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE equipment (
		id INTEGER PRIMARY KEY,
		equipment_code TEXT NOT NULL,
		corp_id INTEGER NOT NULL,
		position_status INTEGER NOT NULL
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO equipment (id, equipment_code, corp_id, position_status) VALUES
		(1, 'BAT-1', 10, 3), (2, 'BAT-2', 10, 3), (3, 'BAT-3', 10, 3),
		(4, 'BAT-4', 10, 3), (5, 'BAT-5', 10, 3), (6, 'BAT-6', 20, 3),
		(7, 'BAT-7', 10, 2), (8, 'BAT-8', 10, 3), (9, 'BAT-9', 10, 3),
		(10, 'BAT-10', 10, 3), (11, 'BAT-11', 10, 3), (12, 'BAT-12', 10, 3)`).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Unix(1_800_000_000, 0)
	values := map[string]map[string]string{
		"GW-1": {
			"alg_soh":    realtimeValue(95, now.Add(-2*time.Second)),
			"01_alg_soh": realtimeValue(85, now.Add(-20*time.Second)),
			"02_alg_soh": realtimeValue(70, now.Add(-5*time.Second)),
		},
		"GW-2":  {"0A_alg_soh": realtimeValue(90, now.Add(-4*time.Second))},
		"GW-3":  {"01_alg_soh": realtimeValue(75, now.Add(-24*time.Hour))},
		"GW-4":  {"01_alg_soh": fmt.Sprintf("0 95 %d", now.Add(-time.Second).Unix())},
		"GW-6":  {"01_alg_soh": realtimeValue(10, now.Add(-time.Second))},
		"GW-7":  {"01_alg_soh": realtimeValue(10, now.Add(-time.Second))},
		"GW-8":  {"01_alg_soh": realtimeValue(60, now.Add(-3*time.Second))},
		"GW-9":  {"01_alg_soh": realtimeValue(59.9, now.Add(-2*time.Second))},
		"GW-10": {"01_alg_soh": realtimeValue(90.1, now.Add(-time.Second))},
		"GW-11": {"alg_soh": realtimeValue(82, now.Add(-2*time.Second))},
		"GW-12": {
			"01_alg_soh": realtimeValue(70, now.Add(-24*time.Hour)),
			"alg_soh":    realtimeValue(99, now.Add(-2*time.Second)),
		},
	}
	marktriggers := []commonmodel.GeneralMarktrigger{
		{EquipmentId: 1, Marktrigger: "GW-1 02_alg_soh"},
		{EquipmentId: 2, Marktrigger: "GW-2 0A_alg_soh"},
		{EquipmentId: 3, Marktrigger: "GW-3 01_alg_soh"},
		{EquipmentId: 4, Marktrigger: "GW-4 01_alg_soh"},
		{EquipmentId: 5, Marktrigger: "GW-5 01_alg_soh"},
		{EquipmentId: 8, Marktrigger: "GW-8 01_alg_soh"},
		{EquipmentId: 9, Marktrigger: "GW-9 01_alg_soh"},
		{EquipmentId: 10, Marktrigger: "GW-10 01_alg_soh"},
		{EquipmentId: 11, Marktrigger: "GW-11 alg_soh"},
		{EquipmentId: 12, Marktrigger: "GW-12 01_alg_soh"},
	}

	dao := &EquipmentHealthStatisticsDAO{
		db: db,
		loadRealtimeData: func(candidates map[string][]string) (map[string]map[string]interface{}, error) {
			result := make(map[string]map[string]interface{}, len(values))
			for gatewayID, fields := range values {
				result[gatewayID] = make(map[string]interface{}, len(fields))
				for dataID, value := range fields {
					result[gatewayID][dataID] = value
				}
			}
			return result, nil
		},
	}

	statistics, err := dao.FindRealtimeStatistics(10, marktriggers)
	if err != nil {
		t.Fatal(err)
	}
	if statistics.TotalCount != 10 || statistics.ValidCount != 8 || statistics.UnknownCount != 2 {
		t.Fatalf("counts=%+v", statistics)
	}
	if statistics.UpdatedAt != now.Add(-time.Second).Unix() {
		t.Fatalf("updatedAt=%d, want %d", statistics.UpdatedAt, now.Add(-time.Second).Unix())
	}
	wantCounts := []int64{1, 2, 4, 1}
	for index, level := range statistics.Levels {
		if level.Count != wantCounts[index] {
			t.Fatalf("level %s count=%d, want %d", level.Code, level.Count, wantCounts[index])
		}
	}
}

func TestFindRealtimeEquipmentHealthStatisticsEmptyAuthorizedScope(t *testing.T) {
	dao := &EquipmentHealthStatisticsDAO{}
	statistics, err := dao.FindRealtimeStatistics(0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if statistics.TotalCount != 0 || statistics.ValidCount != 0 || statistics.UnknownCount != 0 {
		t.Fatalf("unexpected counts: %+v", statistics)
	}
}

func TestFindRealtimeEquipmentHealthStatisticsFindsPrefixedPointWhenMappingUsesLegacyID(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	dao := &EquipmentHealthStatisticsDAO{
		loadRealtimeData: func(candidates map[string][]string) (map[string]map[string]interface{}, error) {
			return map[string]map[string]interface{}{
				"GW-1": {"01_alg_soh": realtimeValue(86, now)},
			}, nil
		},
	}

	statistics, err := dao.FindRealtimeStatistics(1, []commonmodel.GeneralMarktrigger{
		{EquipmentId: 1, Marktrigger: "GW-1 alg_soh"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if statistics.ValidCount != 1 || statistics.UnknownCount != 0 {
		t.Fatalf("counts=%+v, want one valid realtime SOH", statistics)
	}
	if statistics.Levels[1].Count != 1 {
		t.Fatalf("levels=%+v, want SOH良 count 1", statistics.Levels)
	}
}

func realtimeValue(soh float64, timestamp time.Time) string {
	return fmt.Sprintf("1 %v %d", soh, timestamp.Unix())
}
