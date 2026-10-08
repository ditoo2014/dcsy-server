package migration

import (
	"fmt"
	"sort"
	"sync"

	authmodel "dicloud/models/model"

	"gorm.io/gorm"
)

// 权限字典不等于授权：这里只补缺失的菜单/接口定义，绝不创建用户、角色或角色权限关系。
// 业务页面按中文名称匹配，接口权限按 VerifyMiddleware 使用的 url 匹配。
type businessPermissionSeed struct {
	key, name, url, parent string
	kind                   int
	aliases                []string
}

var permissionSeedMutex sync.Mutex

// ensureBusinessPermissions 在默认库及租户库补齐业务云实际使用的权限字典。
// PostgreSQL/OpenGauss 表锁串行化跨进程初始化，避免 auth/business/tolink 同时启动插入重复项。
// 不增加唯一索引、不清理历史重复项，避免改变已有权限 ID 和角色授权。
func ensureBusinessPermissions(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("初始化业务权限字典失败：数据库连接为空")
	}
	permissionSeedMutex.Lock()
	defer permissionSeedMutex.Unlock()
	return db.Transaction(func(tx *gorm.DB) error {
		switch tx.Dialector.Name() {
		case "postgres":
			if err := tx.Exec(`LOCK TABLE "permission" IN SHARE ROW EXCLUSIVE MODE`).Error; err != nil {
				return fmt.Errorf("锁定业务权限字典失败: %w", err)
			}
		case "sqlite": // 仅供现有内存数据库迁移测试使用，同进程由互斥锁保护。
		default:
			return fmt.Errorf("业务权限初始化不支持数据库类型 %s", tx.Dialector.Name())
		}
		var existing []authmodel.Permission
		if err := tx.Order("id asc").Find(&existing).Error; err != nil {
			return fmt.Errorf("读取业务权限字典失败: %w", err)
		}
		sequenceChecked := false
		return seedBusinessPermissions(existing, func(item *authmodel.Permission) error {
			// 兼容手工导入过权限 ID、但未同步序列的旧库。只向上推进序列，不修改任何记录。
			if !sequenceChecked && tx.Dialector.Name() == "postgres" {
				sequenceChecked = true
				if err := advancePermissionSequence(tx, existing); err != nil {
					return err
				}
			}
			if err := tx.Create(item).Error; err != nil {
				return fmt.Errorf("补充业务权限 %s 失败: %w", item.PermissionName, err)
			}
			return nil
		})
	})
}

func advancePermissionSequence(db *gorm.DB, existing []authmodel.Permission) error {
	var maxID uint
	for _, item := range existing {
		if item.ID > maxID {
			maxID = item.ID
		}
	}
	if maxID == 0 {
		return nil
	}
	var nextID int64
	if err := db.Raw(`SELECT nextval(pg_get_serial_sequence('permission', 'id'))`).Scan(&nextID).Error; err != nil {
		return fmt.Errorf("检查权限主键序列失败: %w", err)
	}
	if nextID <= int64(maxID) {
		if err := db.Exec(`SELECT setval(pg_get_serial_sequence('permission', 'id'), ?, true)`, maxID).Error; err != nil {
			return fmt.Errorf("同步权限主键序列失败: %w", err)
		}
	}
	return nil
}

// seedBusinessPermissions 保留全部已有字段；新增子项引用实际父项 ID，不依赖固定主键。
func seedBusinessPermissions(existing []authmodel.Permission, create func(*authmodel.Permission) error) error {
	known := append([]authmodel.Permission(nil), existing...)
	sort.SliceStable(known, func(i, j int) bool { return known[i].ID < known[j].ID })
	parents := make(map[string]authmodel.Permission)
	for _, seed := range businessPermissionSeeds() {
		var found *authmodel.Permission
		for i := range known {
			item := &known[i]
			// 已有公共权限（平台 0）可以复用；采集云/组态云权限不能替代业务云权限。
			if item.PlatId != 0 && item.PlatId != 1 {
				continue
			}
			if item.Url == seed.url && item.Url != "" && (seed.kind == 2 || item.PermissionType == 1) {
				found = item
				break
			}
			if seed.kind == 1 && item.PermissionType == 1 && permissionSeedNameMatches(seed, item.PermissionName) {
				found = item
				break
			}
		}
		if found != nil {
			parents[seed.key] = *found
			continue
		}
		item := authmodel.Permission{PermissionName: seed.name, Url: seed.url, PermissionType: seed.kind, PlatId: 1}
		if seed.parent != "" {
			parent, ok := parents[seed.parent]
			if !ok {
				return fmt.Errorf("权限 %s 缺少父级定义 %s", seed.name, seed.parent)
			}
			item.ParentId, item.Level = int(parent.ID), parent.Level+1
		}
		if err := create(&item); err != nil {
			return err
		}
		if item.ID == 0 {
			return fmt.Errorf("权限 %s 创建后未返回主键", seed.name)
		}
		parents[seed.key] = item
		known = append(known, item)
	}
	return nil
}

func permissionSeedNameMatches(seed businessPermissionSeed, name string) bool {
	if name == seed.name {
		return true
	}
	for _, alias := range seed.aliases {
		if name == alias {
			return true
		}
	}
	return false
}

func businessPermissionSeeds() []businessPermissionSeed {
	// 19 个实际页面及原版顶层/分组名称；旧库短名与前端别名保持兼容。
	pages := []businessPermissionSeed{
		{key: "jsc", name: "驾驶舱", url: "/jsc"},
		{key: "lifecycle", name: "全生命周期管理", url: "/qsmzqgl"},
		{key: "dczt", name: "电池状态", url: "/dczt", parent: "lifecycle"},
		{key: "gjzz", name: "轨迹追踪", url: "/gjzz", parent: "lifecycle"},
		{key: "ywgl", name: "运维管理", url: "/ywgl", parent: "lifecycle"},
		{key: "health", name: "电池健康管理", url: "/dcjkgl"},
		{key: "jkztjc", name: "健康状态监测", url: "/jkztjc", parent: "health", aliases: []string{"健康状态监测管理"}},
		{key: "lssjcx", name: "历史数据查询", url: "/lssjcx", parent: "health"},
		{key: "alarm", name: "告警信息", url: "/gjxx", parent: "health"},
		{key: "gjxxssgj", name: "实时告警", url: "/gjxxssgj", parent: "alarm"},
		{key: "gjxxlsgj", name: "历史告警", url: "/gjxxlsgj", parent: "alarm"},
		{key: "warning", name: "预警信息", url: "/yjxx", parent: "health"},
		{key: "yjxxssyj", name: "实时预警", url: "/yjxxssyj", parent: "warning"},
		{key: "yjxxlsyj", name: "历史预警", url: "/yjxxlsyj", parent: "warning"},
		{key: "basic", name: "基础信息管理", url: "/jcxxgl"},
		{key: "kfxxgl", name: "库房信息管理", url: "/jcxxgl/kfxxgl", parent: "basic", aliases: []string{"库房", "库房信息"}},
		{key: "zdxxgl", name: "站点信息管理", url: "/zdxxgl/index", parent: "basic", aliases: []string{"站点", "站点信息", "站点管理"}},
		{key: "scxxgl", name: "电池出厂信息", url: "/scxxgl/index", parent: "basic", aliases: []string{"生产信息管理"}},
		{key: "data", name: "电池数据管理", url: "/dcsjgl", parent: "basic"},
		{key: "wgpz", name: "网关配置", url: "/dcsjgl/wgpz", parent: "data"},
		{key: "jrgl", name: "数据接入", url: "/dcsjgl/jrgl", parent: "data"},
		{key: "tycfqgl", name: "通用触发器管理", url: "/dcsjgl/tycfqgl", parent: "data"},
		{key: "people", name: "人员管理", url: "/rygl", parent: "basic"},
		{key: "jsgl", name: "角色管理", url: "/rygl/jsgl", parent: "people"},
		{key: "yhgl", name: "用户管理", url: "/rygl/yhgl", parent: "people"},
		{key: "basicdata", name: "基础数据", url: "/jcsj", parent: "basic"},
		{key: "xtrz", name: "系统日志", url: "/jcsj/xtrz", parent: "basicdata", aliases: []string{"日志信息"}},
	}
	for i := range pages {
		pages[i].kind = 1
	}

	// 功能字典独立分组，选择页面权限不会隐式获得接口操作权限。
	seeds := append(pages, businessPermissionSeed{key: "functions", name: "业务功能权限", url: "dcsy:functions", kind: 2})
	groups := []struct {
		key, name string
		actions   [][2]string
	}{
		{"user", "用户管理功能", [][2]string{{"register", "创建用户"}, {"delete", "删除用户"}, {"update", "修改用户及设备授权"}, {"findpage", "查询用户"}, {"resetuserpassword", "重置用户密码"}}},
		{"role", "角色管理功能", [][2]string{{"list", "查询角色"}, {"add", "新增角色"}, {"delete", "删除角色"}, {"update", "修改角色"}}},
		{"permission", "权限字典管理", [][2]string{{"list", "查询权限字典"}, {"add", "新增权限定义"}, {"delete", "删除权限定义"}, {"update", "修改权限定义"}}},
		{"rolepermission", "角色权限管理", [][2]string{{"list", "查询角色权限关系"}, {"permissions", "查询角色权限详情"}, {"add", "绑定角色权限"}, {"delete", "解绑角色权限"}, {"batchadd", "批量绑定角色权限"}, {"batchdelete", "批量解绑角色权限"}, {"batchdeletelist", "按列表批量解绑角色权限"}}},
		{"userrole", "用户角色管理", [][2]string{{"list", "查询用户角色"}, {"add", "绑定用户角色"}, {"delete", "解绑用户角色"}}},
		{"corp", "组织管理功能", [][2]string{{"list", "查询部门"}, {"add", "新增部门"}, {"delete", "删除部门"}, {"update", "修改部门"}, {"export", "导出部门"}, {"import", "导入部门"}}},
		{"syslog", "日志管理功能", [][2]string{{"list", "查询系统日志"}}},
		{"synclogin", "业务采集平台切换", [][2]string{{"totolink", "同步登录采集云"}}},
		{"trigger", "触发器管理功能", [][2]string{{"add", "新增触发器"}, {"delete", "删除触发器"}, {"update", "修改触发器"}, {"find", "查询触发器"}}},
		{"generaltrigger", "通用触发器功能", [][2]string{{"add", "新增通用触发器"}, {"delete", "删除通用触发器"}, {"update", "修改通用触发器"}, {"find", "查询通用触发器"}}},
		{"statisticdata", "统计数据查询功能", [][2]string{{"find", "查询统计数据"}, {"count", "统计数据计数"}, {"levelcorp", "查询分级部门统计"}, {"markdatelist", "查询日报和时间标记"}}},
		{"alarmrecord", "告警记录功能", [][2]string{{"find", "查询告警及预警"}, {"delete", "删除告警及预警"}, {"countbyequipment", "查询设备告警数量及状态"}}},
		{"sitegrouparr", "站点分组功能", [][2]string{{"add", "新增站点分组"}, {"update", "修改站点分组"}, {"find", "查询站点分组"}}},
		{"sitegroup", "站点分组删除功能", [][2]string{{"delete", "删除站点分组"}}},
		{"site", "站点管理功能", [][2]string{{"add", "新增站点"}, {"delete", "删除站点"}, {"update", "修改站点"}, {"find", "查询站点"}, {"details:find", "查询站点详情"}, {"export", "导出站点"}, {"import", "导入站点及下载模板"}}},
		{"equipment", "电池设备功能", [][2]string{{"add", "新增电池及类型"}, {"delete", "删除电池及类型"}, {"update", "修改电池及类型"}, {"find", "查询电池及健康状态"}, {"countbycorp", "查询部门电池数量"}, {"export", "导出电池"}, {"import", "导入电池及下载模板"}, {"alldata", "查询电池实时数据"}}},
		{"datadetails", "设备数据接入功能", [][2]string{{"add", "新增数据详情"}, {"batchadd", "批量新增数据详情"}, {"update", "修改数据详情"}, {"batchupdate", "批量修改数据详情"}, {"find", "查询数据详情"}, {"delete", "删除数据详情"}}},
		{"warehouse", "库房管理功能", [][2]string{{"add", "新增库房"}, {"update", "修改库房"}, {"delete", "删除库房"}, {"find", "查询库房"}, {"export", "导出库房"}, {"import", "导入库房及下载模板"}}},
		{"equipmentcirlog", "电池流转记录功能", [][2]string{{"add", "新增流转记录"}, {"find", "查询流转记录"}}},
		{"equipmentgoppst", "电池位置功能", [][2]string{{"add", "上报电池位置"}, {"find", "查询电池位置"}}},
		{"equipmentmitrcd", "电池运维记录功能", [][2]string{{"add", "新增电池运维记录"}, {"find", "查询电池运维记录"}, {"export", "导出电池运维记录"}, {"import", "导入电池运维记录"}}},
		{"equipmenthltstd", "电池健康标准功能", [][2]string{{"update", "修改健康判断标准"}, {"find", "查询健康判断标准"}}},
		{"statistictask", "统计任务功能", [][2]string{{"start", "执行统计任务"}}},
	}
	for _, group := range groups {
		key := "function-group:" + group.key
		seeds = append(seeds, businessPermissionSeed{key: key, name: group.name, url: "dcsy:function:" + group.key, parent: "functions", kind: 2})
		for _, action := range group.actions {
			url := group.key + ":" + action[0]
			seeds = append(seeds, businessPermissionSeed{key: url, name: action[1], url: url, parent: key, kind: 2})
		}
	}
	return seeds
}
