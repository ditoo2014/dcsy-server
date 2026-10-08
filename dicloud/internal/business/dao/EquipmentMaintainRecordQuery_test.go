package dao

import (
	"reflect"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 只生成 SQL，不建立数据库网络连接，用于验证权限条件和参数占位符。
func maintainRecordDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(postgres.New(postgres.Config{DSN: "host=localhost user=offline dbname=offline sslmode=disable"}), &gorm.Config{
		DryRun: true, DisableAutomaticPing: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMaintainRecordQueryFiltersAndPermissions(t *testing.T) {
	filter := MaintainRecordFilter{
		EquipmentID: 7, EquipmentCode: "BAT' OR 1=1 --", EquipmentTypeName: "钠电池'",
		MinimumCount: 3, StartTime: 10, EndTime: 90, CorpIDs: []int{2, 3},
	}
	query := (&EquipmentMaintainRecordDao{db: maintainRecordDryRunDB(t)}).maintainRecordQuery(filter).
		Select("r.*").Order("r.create_time DESC, r.id DESC").Find(&[]MaintainRecordRow{})
	if query.Error != nil {
		t.Fatal(query.Error)
	}
	sql := query.Statement.SQL.String()
	for _, expected := range []string{"e.corp_id IN", "r.equipment_id =", "e.equipment_name) LIKE", "e.equipment_type_name LIKE", "GROUP BY", "HAVING COUNT(*) >=", "r.create_time DESC, r.id DESC"} {
		if !strings.Contains(sql, expected) {
			t.Fatalf("query missing %q: %s", expected, sql)
		}
	}
	if strings.Contains(sql, filter.EquipmentCode) || strings.Contains(sql, filter.EquipmentTypeName) {
		t.Fatal("query directly interpolates untrusted values")
	}
	// 最少次数统计与主查询使用完全相同的权限、编码和类型条件。
	wanted := []any{int64(10), int64(90), 2, 3, 7, "%BAT' OR 1=1 --%", "%钠电池'%", int64(10), int64(90), 2, 3, "%BAT' OR 1=1 --%", "%钠电池'%", 3}
	if !reflect.DeepEqual(query.Statement.Vars, wanted) {
		t.Fatalf("query variables = %#v, want %#v", query.Statement.Vars, wanted)
	}
}

func TestMaintainRecordQueryEmptyPermissionAndAdministrator(t *testing.T) {
	dao := &EquipmentMaintainRecordDao{db: maintainRecordDryRunDB(t)}
	for _, admin := range []bool{false, true} {
		query := dao.maintainRecordQuery(MaintainRecordFilter{StartTime: 0, EndTime: 100, IsAdmin: admin}).Find(&[]MaintainRecordRow{})
		sql := query.Statement.SQL.String()
		if strings.Contains(sql, "1 = 0") == admin {
			t.Fatalf("administrator %v gets unexpected permission clause: %s", admin, sql)
		}
		if strings.Contains(sql, "r.equipment_id =") {
			t.Fatal("equipment ID zero incorrectly filters by a single equipment")
		}
	}
}
