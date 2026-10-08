package migration

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"

	authmodel "dicloud/models/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestBusinessPermissionCatalogCoversRuntimeRoutes(t *testing.T) {
	urls := map[string]bool{}
	keys := map[string]bool{}
	pageNames := map[string]bool{}
	for _, seed := range businessPermissionSeeds() {
		if keys[seed.key] || urls[seed.url] {
			t.Fatalf("重复的权限定义：%s", seed.key)
		}
		if seed.parent != "" && !keys[seed.parent] {
			t.Fatalf("父权限必须先声明：%s", seed.key)
		}
		keys[seed.key], urls[seed.url] = true, true
		if seed.kind == 1 {
			pageNames[seed.name] = true
		}
		if strings.Contains(seed.url, "wechat") || strings.Contains(seed.url, "maintain:") || strings.Contains(seed.url, "todynalt") {
			t.Fatalf("不得迁入已排除平台的权限：%s", seed.url)
		}
	}
	pages := []string{"驾驶舱", "电池状态", "轨迹追踪", "运维管理", "健康状态监测", "历史数据查询", "实时告警", "历史告警", "实时预警", "历史预警", "库房信息管理", "站点信息管理", "电池出厂信息", "网关配置", "数据接入", "通用触发器管理", "角色管理", "用户管理", "系统日志"}
	for _, name := range pages {
		if !pageNames[name] {
			t.Errorf("缺少当前菜单：%s", name)
		}
	}
	for _, filename := range []string{"../../cmd/business/routes.go", "../../cmd/auth/routes.go"} {
		file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}
			method, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (method.Sel.Name != "POST" && method.Sel.Name != "GET") {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok {
				return true
			}
			path, _ := strconv.Unquote(literal.Value)
			if strings.Contains(filename, "/auth/") && strings.HasPrefix(path, "/tolink/") {
				return false
			}
			// 云组态入口不属于 dcsy 本次提取范围。
			if strings.HasSuffix(path, "/todynalt") {
				return false
			}
			for _, arg := range call.Args[1:] {
				verify, ok := arg.(*ast.CallExpr)
				if !ok || len(verify.Args) != 2 {
					continue
				}
				name, ok := verify.Fun.(*ast.SelectorExpr)
				if !ok || name.Sel.Name != "VerifyMiddleware" {
					continue
				}
				code, ok := verify.Args[0].(*ast.BasicLit)
				if !ok {
					continue
				}
				url, _ := strconv.Unquote(code.Value)
				if !urls[url] {
					t.Errorf("路由 %s 缺少权限 %s", path, url)
				}
			}
			return false
		})
	}
}

func TestBusinessPermissionSeedPreservesExistingAndIsIdempotent(t *testing.T) {
	rows := []authmodel.Permission{
		{ID: 901, PermissionName: "库房", PermissionType: 1, Url: "", PlatId: 1, Level: 2, ParentId: 812, ApiStage: 9},
		{ID: 902, PermissionName: "自定义查用户名称", PermissionType: 2, Url: "user:findpage", PlatId: 0, Level: 7, ParentId: 899},
		{ID: 903, PermissionName: "已有的自定义驾驶舱", PermissionType: 1, Url: "/jsc", PlatId: 1, Level: 3, ParentId: 888},
		{ID: 904, PermissionName: "采集角色", PermissionType: 2, Url: "role:add", PlatId: 2},
	}
	original := append([]authmodel.Permission(nil), rows...)
	create := func(item *authmodel.Permission) error {
		item.ID = uint(1000 + len(rows))
		rows = append(rows, *item)
		return nil
	}
	if err := seedBusinessPermissions(rows, create); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows[:len(original)], original) {
		t.Fatal("初始化修改了已有 ID、名称、层级或平台")
	}
	count := len(rows)
	if err := seedBusinessPermissions(rows, create); err != nil || len(rows) != count {
		t.Fatalf("重复初始化应为零新增：%v", err)
	}
	var aliases, public, businessRole int
	for _, item := range rows {
		if item.PermissionType == 1 && (item.PermissionName == "库房" || item.PermissionName == "库房信息管理") {
			aliases++
		}
		if item.Url == "user:findpage" {
			public++
		}
		if item.Url == "role:add" && item.PlatId == 1 {
			businessRole++
		}
	}
	if aliases != 1 || public != 1 || businessRole != 1 {
		t.Fatalf("旧名/公共权限复用或跨平台隔离错误：%d %d %d", aliases, public, businessRole)
	}
}

func TestBusinessPermissionSeedUsesActualParentIDsAndStopsOnError(t *testing.T) {
	rows := []authmodel.Permission{{ID: 7000, PermissionName: "全生命周期管理", PermissionType: 1, PlatId: 0, Level: 4}}
	if err := seedBusinessPermissions(rows, func(item *authmodel.Permission) error {
		item.ID = uint(8000 + len(rows))
		if item.Url == "/dczt" && (item.ParentId != 7000 || item.Level != 5) {
			t.Fatalf("新增子项没有引用旧父项实际 ID/层级：%+v", item)
		}
		rows = append(rows, *item)
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	want := errors.New("测试写入失败")
	if err := seedBusinessPermissions(nil, func(*authmodel.Permission) error { return want }); !errors.Is(err, want) {
		t.Fatalf("写入失败未透传：%v", err)
	}
}

// 以下最小内存 SQL 驱动只接受权限表的锁、读、插入和序列操作，禁止访问任何真实库。
// 其他表或 UPDATE/DELETE 一旦出现就失败，从而验证没有隐式创建用户、角色或赋权。
type permissionTestDB struct {
	mu                   sync.Mutex
	rows                 []authmodel.Permission
	sequence             int64
	locks, rollbacks     int
	failInsert, failLock bool
}
type permissionTestConn struct {
	store  *permissionTestDB
	locked bool
	before []authmodel.Permission
}
type permissionTestRows struct {
	columns []string
	values  [][]driver.Value
	index   int
}

func (s *permissionTestDB) Connect(context.Context) (driver.Conn, error) {
	return &permissionTestConn{store: s}, nil
}
func (s *permissionTestDB) Driver() driver.Driver { return nil }
func (c *permissionTestConn) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("禁止预处理语句")
}
func (c *permissionTestConn) Close() error              { return nil }
func (c *permissionTestConn) Begin() (driver.Tx, error) { return c, nil }
func (c *permissionTestConn) Commit() error {
	if c.locked {
		c.locked = false
		c.store.mu.Unlock()
	}
	return nil
}
func (c *permissionTestConn) Rollback() error {
	if c.locked {
		c.store.rows = c.before
		c.store.rollbacks++
		c.locked = false
		c.store.mu.Unlock()
	}
	return nil
}
func (c *permissionTestConn) ExecContext(_ context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if query == `LOCK TABLE "permission" IN SHARE ROW EXCLUSIVE MODE` {
		if c.store.failLock {
			return nil, errors.New("测试锁定失败")
		}
		c.store.mu.Lock()
		c.locked = true
		c.store.locks++
		c.before = append([]authmodel.Permission(nil), c.store.rows...)
		return driver.RowsAffected(0), nil
	}
	if strings.HasPrefix(query, "SELECT setval(") && c.locked {
		c.store.sequence = args[0].Value.(int64)
		return driver.RowsAffected(1), nil
	}
	return nil, fmt.Errorf("非预期的写入操作：%s", query)
}
func (c *permissionTestConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if !c.locked {
		return nil, errors.New("权限读写之前必须先取得数据库事务锁")
	}
	if strings.HasPrefix(query, "SELECT nextval(") {
		c.store.sequence++
		return &permissionTestRows{columns: []string{"nextval"}, values: [][]driver.Value{{c.store.sequence}}}, nil
	}
	if strings.HasPrefix(query, `SELECT * FROM "permission"`) {
		result := &permissionTestRows{columns: []string{"id", "permission_name", "url", "type", "level", "parent_id", "api_stage", "plat_id"}}
		for _, p := range c.store.rows {
			result.values = append(result.values, []driver.Value{int64(p.ID), p.PermissionName, p.Url, int64(p.PermissionType), int64(p.Level), int64(p.ParentId), int64(p.ApiStage), int64(p.PlatId)})
		}
		return result, nil
	}
	if strings.HasPrefix(query, `INSERT INTO "permission"`) {
		if c.store.failInsert {
			return nil, errors.New("测试插入失败")
		}
		c.store.sequence++
		p := authmodel.Permission{ID: uint(c.store.sequence), PermissionName: args[0].Value.(string), Url: args[1].Value.(string), PermissionType: int(args[2].Value.(int64)), Level: int(args[3].Value.(int64)), ParentId: int(args[4].Value.(int64)), ApiStage: int(args[5].Value.(int64)), PlatId: int(args[6].Value.(int64))}
		for _, existing := range c.store.rows {
			if existing.ID == p.ID {
				return nil, errors.New("主键序列未同步")
			}
		}
		c.store.rows = append(c.store.rows, p)
		return &permissionTestRows{columns: []string{"id"}, values: [][]driver.Value{{int64(p.ID)}}}, nil
	}
	return nil, fmt.Errorf("非预期的数据库查询：%s", query)
}
func (r *permissionTestRows) Columns() []string { return r.columns }
func (r *permissionTestRows) Close() error      { return nil }
func (r *permissionTestRows) Next(dest []driver.Value) error {
	if r.index >= len(r.values) {
		return io.EOF
	}
	copy(dest, r.values[r.index])
	r.index++
	return nil
}

func openPermissionTestDB(t *testing.T, store *permissionTestDB) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(store)
	t.Cleanup(func() { _ = sqlDB.Close() })
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{DisableAutomaticPing: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func TestBusinessPermissionTransactionAndConcurrentStartup(t *testing.T) {
	store := &permissionTestDB{rows: []authmodel.Permission{{ID: 9999, PermissionName: "已有字典", PlatId: 2}}, sequence: 1}
	db := openPermissionTestDB(t, store)
	var group sync.WaitGroup
	errorsCh := make(chan error, 4)
	for i := 0; i < 4; i++ {
		group.Add(1)
		go func() { defer group.Done(); errorsCh <- ensureBusinessPermissions(db) }()
	}
	group.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
	if len(store.rows) != len(businessPermissionSeeds())+1 || store.locks != 4 {
		t.Fatalf("并发启动重复插入或未执行事务锁：rows=%d locks=%d", len(store.rows), store.locks)
	}
	if store.rows[0].ID != 9999 || store.rows[1].ID <= 9999 {
		t.Fatal("旧主键未保留或序列未向上同步")
	}
}

func TestBusinessPermissionInitializationRollsBackFailures(t *testing.T) {
	store := &permissionTestDB{failInsert: true}
	if err := ensureBusinessPermissions(openPermissionTestDB(t, store)); err == nil {
		t.Fatal("写入失败应返回错误")
	}
	if len(store.rows) != 0 || store.rollbacks != 1 {
		t.Fatal("失败事务没有回滚")
	}
	store = &permissionTestDB{failLock: true}
	if err := ensureBusinessPermissions(openPermissionTestDB(t, store)); err == nil {
		t.Fatal("未获得锁时不得初始化")
	}
	if len(store.rows) != 0 {
		t.Fatal("锁定失败后仍写入字典")
	}
}
