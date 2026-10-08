package dao

import (
	"dicloud/internal/business/model"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// 使用 GORM DryRun 只生成 SQL，不连接任何真实数据库，验证流转查询的条件和稳定排序。
func TestCirculationLogFindQueryIsParameterizedAndStable(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  "host=localhost user=offline dbname=offline sslmode=disable",
		PreferSimpleProtocol: true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	dao := &EquipmentCirculationLogDao{db: db}
	query, ok := dao.findQuery(model.EquipmentCirculationLog{
		EquipmentId:       9,
		EquipmentTypeName: "钠电池' OR 1=1 --",
	})
	if !ok {
		t.Fatal("positive equipment id should be queryable")
	}
	query = query.Scan(&[]model.EquipmentCirculationLog{})
	sql := query.Statement.SQL.String()
	if !strings.Contains(sql, "equipment_id = $1") || !strings.Contains(sql, "equipment_type_name LIKE $2") {
		t.Fatalf("query does not use parameter placeholders: %s", sql)
	}
	if !strings.Contains(sql, "ORDER BY create_time ASC,id ASC") && !strings.Contains(sql, `ORDER BY "create_time" ASC,"id" ASC`) {
		t.Fatalf("query does not use stable ascending order: %s", sql)
	}
	if strings.Contains(sql, "钠电池") || strings.Contains(sql, "1=1") {
		t.Fatalf("untrusted filter was interpolated into SQL: %s", sql)
	}
	if len(query.Statement.Vars) != 2 || query.Statement.Vars[0] != 9 || query.Statement.Vars[1] != "%钠电池' OR 1=1 --%" {
		t.Fatalf("unexpected query parameters: %#v", query.Statement.Vars)
	}
}

func TestCirculationLogFindQueryRejectsZeroEquipment(t *testing.T) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN: "host=localhost user=offline dbname=offline sslmode=disable",
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true})
	if err != nil {
		t.Fatal(err)
	}
	if query, ok := (&EquipmentCirculationLogDao{db: db}).findQuery(model.EquipmentCirculationLog{}); ok || query != nil {
		t.Fatal("equipment id zero must not produce a query")
	}
}
