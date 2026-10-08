package migration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// 内存驱动只接受迁移白名单查询；不会打开数据库网络连接或修改业务记录。
type recordTimeTestDB struct {
	types                      map[string]string
	values                     []driver.Value
	locked                     string
	alters, commits, rollbacks int
	failLock                   bool
}

func (s *recordTimeTestDB) Connect(context.Context) (driver.Conn, error) { return s, nil }
func (s *recordTimeTestDB) Driver() driver.Driver                        { return recordTimeTestDriver{} }

type recordTimeTestDriver struct{}

func (recordTimeTestDriver) Open(string) (driver.Conn, error) {
	return nil, errors.New("禁止真实连接")
}
func (s *recordTimeTestDB) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("不支持预处理")
}
func (s *recordTimeTestDB) Close() error              { return nil }
func (s *recordTimeTestDB) Begin() (driver.Tx, error) { return s, nil }
func (s *recordTimeTestDB) Commit() error             { s.commits++; s.locked = ""; return nil }
func (s *recordTimeTestDB) Rollback() error           { s.rollbacks++; s.locked = ""; return nil }
func (s *recordTimeTestDB) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if strings.HasPrefix(query, "LOCK TABLE public.") {
		if s.failLock {
			return nil, errors.New("锁定失败")
		}
		s.locked = strings.Fields(query)[2][len("public."):]
		return driver.RowsAffected(0), nil
	}
	if s.locked != "" && strings.HasPrefix(query, "ALTER TABLE public."+s.locked+" ALTER COLUMN create_time TYPE bigint") && strings.Contains(query, "USING CASE") {
		s.types[s.locked] = "bigint"
		s.alters++
		return driver.RowsAffected(0), nil
	}
	return nil, fmt.Errorf("禁止执行非预期语句：%s", query)
}
func (s *recordTimeTestDB) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if strings.Contains(query, "FROM information_schema.columns") {
		columnType, exists := s.types[args[0].Value.(string)]
		rows := &recordTimeTestRows{columns: []string{"data_type"}}
		if exists {
			rows.values = [][]driver.Value{{columnType}}
		}
		return rows, nil
	}
	if s.locked != "" && query == "SELECT id, create_time::text FROM public."+s.locked+" WHERE create_time IS NOT NULL" {
		rows := &recordTimeTestRows{columns: []string{"id", "create_time"}}
		for i, value := range s.values {
			rows.values = append(rows.values, []driver.Value{int64(i + 1), value})
		}
		return rows, nil
	}
	return nil, fmt.Errorf("禁止非预期查询：%s", query)
}

type recordTimeTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (r *recordTimeTestRows) Columns() []string { return r.columns }
func (r *recordTimeTestRows) Close() error      { return nil }
func (r *recordTimeTestRows) Next(values []driver.Value) error {
	if r.index == len(r.values) {
		return io.EOF
	}
	copy(values, r.values[r.index])
	r.index++
	return nil
}
func recordTimeDatabase(t *testing.T, store *recordTimeTestDB) *gorm.DB {
	t.Helper()
	pool := sql.OpenDB(store)
	t.Cleanup(func() { _ = pool.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: pool}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}
func TestBusinessRecordTimeMigration(t *testing.T) {
	tables := []string{"equipment_maintain_record", "equipment_circulation_log", "equipment_geography_position"}
	store := &recordTimeTestDB{types: map[string]string{}, values: []driver.Value{"1726862400", `"1726862401"`, "null"}}
	for _, table := range tables {
		store.types[table] = "json"
	}
	db := recordTimeDatabase(t, store)
	if err := EnsureBusinessRecordTimeColumns(db); err != nil {
		t.Fatal(err)
	}
	if store.alters != 3 || store.commits != 3 || store.rollbacks != 0 {
		t.Fatalf("迁移未覆盖三张表：%+v", store)
	}
	if err := EnsureBusinessRecordTimeColumns(db); err != nil {
		t.Fatal(err)
	}
	if store.alters != 3 {
		t.Fatal("整数列不应重复修改")
	}
}
func TestBusinessRecordTimeMigrationEmptyDatabase(t *testing.T) {
	store := &recordTimeTestDB{types: map[string]string{}}
	if err := EnsureBusinessRecordTimeColumns(recordTimeDatabase(t, store)); err != nil {
		t.Fatal(err)
	}
	if store.alters != 0 {
		t.Fatal("不存在的表应交由 AutoMigrate 新建")
	}
}
func TestBusinessRecordTimeMigrationInvalidValueAndLock(t *testing.T) {
	for _, lockFailure := range []bool{false, true} {
		store := &recordTimeTestDB{types: map[string]string{"equipment_maintain_record": "jsonb"}, values: []driver.Value{"{}"}, failLock: lockFailure}
		if err := EnsureBusinessRecordTimeColumns(recordTimeDatabase(t, store)); err == nil {
			t.Fatal("错误必须向上传递")
		}
		if store.alters != 0 || store.commits != 0 || store.rollbacks != 1 {
			t.Fatal("错误分支不能转换或提交")
		}
	}
}
