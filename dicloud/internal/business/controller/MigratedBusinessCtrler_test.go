package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	authcontroller "dicloud/internal/auth/controller"
	"dicloud/util/opengauss"

	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// migratedResponse 是迁移接口统一响应的最小测试结构。
type migratedResponse struct {
	Code   int            `json:"code"`
	Msg    string         `json:"msg"`
	Return map[string]any `json:"return"`
}

// newMigratedControllerDB 使用独立内存库验证 Controller 的真实 SQL，
// 避免测试依赖开发机上的 OpenGauss 数据库。
func newMigratedControllerDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:migrated_controller_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("打开SQLite测试库失败: %v", err)
	}
	previous := opengauss.OpengaussGormDB
	opengauss.OpengaussGormDB = db
	t.Cleanup(func() { opengauss.OpengaussGormDB = previous })
	return db
}

func callMigratedController(t *testing.T, method, target string, body any, handler gin.HandlerFunc) migratedResponse {
	t.Helper()
	gin.SetMode(gin.TestMode)
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("序列化请求失败: %v", err)
	}
	request := httptest.NewRequest(method, target, bytes.NewReader(payload))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = request
	handler(ctx)

	var response migratedResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析响应失败: %v; body=%s", err, recorder.Body.String())
	}
	return response
}

func requireMigratedSuccess(t *testing.T, response migratedResponse) {
	t.Helper()
	if response.Code != http.StatusOK {
		t.Fatalf("接口返回失败: code=%d msg=%s return=%v", response.Code, response.Msg, response.Return)
	}
}

func TestEquipmentTypeControllerCRUDAndReferenceProtection(t *testing.T) {
	db := newMigratedControllerDB(t)
	if err := db.Exec(`CREATE TABLE equipment_type (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		equipment_type_name TEXT NOT NULL UNIQUE,
		equipment_type_describe TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE equipment (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		equipment_type_name TEXT
	)`).Error; err != nil {
		t.Fatal(err)
	}

	controller := EquipmentTypeCtrler{}
	created := callMigratedController(t, http.MethodPost, "/business/equipment/saveEquipmentType", map[string]any{
		"equipmentTypeName": " 锂硫电池 ", "equipmentTypeDescribe": " 主力机型 ",
	}, controller.Add)
	requireMigratedSuccess(t, created)

	var typeID uint
	if err := db.Table("equipment_type").Select("id").Where("equipment_type_name = ?", "锂硫电池").Scan(&typeID).Error; err != nil || typeID == 0 {
		t.Fatalf("设备类型未保存: id=%d err=%v", typeID, err)
	}
	if err := db.Exec("INSERT INTO equipment(equipment_type_name) VALUES (?)", "锂硫电池").Error; err != nil {
		t.Fatal(err)
	}

	updated := callMigratedController(t, http.MethodPost, "/business/equipment/modifyEquipmentType", map[string]any{
		"id": typeID, "equipmentTypeName": "钠电池", "equipmentTypeDescribe": "钠离子电池",
	}, controller.Update)
	requireMigratedSuccess(t, updated)

	var equipmentTypeName string
	if err := db.Table("equipment").Select("equipment_type_name").Scan(&equipmentTypeName).Error; err != nil {
		t.Fatal(err)
	}
	if equipmentTypeName != "钠电池" {
		t.Fatalf("设备类型名称未同步更新: %q", equipmentTypeName)
	}

	blocked := callMigratedController(t, http.MethodPost, "/business/equipment/deleteEquipmentType", map[string]any{"id": typeID}, controller.Delete)
	if blocked.Code != http.StatusBadRequest {
		t.Fatalf("仍被设备引用的类型不应删除: %+v", blocked)
	}
	if err := db.Exec("DELETE FROM equipment").Error; err != nil {
		t.Fatal(err)
	}
	deleted := callMigratedController(t, http.MethodPost, "/business/equipment/deleteEquipmentType", map[string]any{"id": typeID}, controller.Delete)
	requireMigratedSuccess(t, deleted)
}

func TestAlarmUsersControllerFindAndDelete(t *testing.T) {
	db := newMigratedControllerDB(t)
	statements := []string{
		`CREATE TABLE alarm_users (id INTEGER PRIMARY KEY, trigger_id INTEGER, alarm_code TEXT, alarm_type TEXT, alarm_stage INTEGER, alarm_channel TEXT, alarm_users TEXT)`,
		`CREATE TABLE trigger_config (id INTEGER PRIMARY KEY, site_id INTEGER, equipment_id INTEGER, code TEXT)`,
		`CREATE TABLE equipment (id INTEGER PRIMARY KEY, equipment_name TEXT)`,
		`CREATE TABLE site (id INTEGER PRIMARY KEY, site_name TEXT)`,
		`INSERT INTO site(id, site_name) VALUES (1, '成都站')`,
		`INSERT INTO equipment(id, equipment_name) VALUES (2, '钠电池-01')`,
		`INSERT INTO trigger_config(id, site_id, equipment_id, code) VALUES (3, 1, 2, 'CELL_OV')`,
		`INSERT INTO alarm_users(id, trigger_id, alarm_code, alarm_type, alarm_stage, alarm_channel, alarm_users) VALUES (4, 3, 'CELL_OV', '0', 2, '0', '7, 9')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("准备告警接收人数据失败: %v", err)
		}
	}

	controller := AlarmUsersCtrler{}
	found := callMigratedController(t, http.MethodPost, "/business/alarmUsers/findAlarmUsers", map[string]any{
		"equipmentId": 2, "coding": "CELL_OV",
	}, controller.Find)
	requireMigratedSuccess(t, found)
	list, ok := found.Return["list"].([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("告警接收人查询结果错误: %#v", found.Return["list"])
	}
	item, _ := list[0].(map[string]any)
	if item["shortMsgAlarm"] != true {
		t.Fatalf("短信渠道未正确转换: %#v", item)
	}
	userIDs, _ := item["shortMsgAlarmUserIds"].([]any)
	if len(userIDs) != 2 || userIDs[0] != float64(7) || userIDs[1] != float64(9) {
		t.Fatalf("短信接收人未正确转换: %#v", userIDs)
	}

	deleted := callMigratedController(t, http.MethodPost, "/business/alarmUsers/deleteAlarmUsers", map[string]any{"id": 4}, controller.Delete)
	requireMigratedSuccess(t, deleted)
	var count int64
	if err := db.Table("alarm_users").Count(&count).Error; err != nil || count != 0 {
		t.Fatalf("告警接收人未删除: count=%d err=%v", count, err)
	}
}

func TestLiveAlarmControllerFindAndRemove(t *testing.T) {
	db := newMigratedControllerDB(t)
	if err := db.Exec(`CREATE TABLE alarm_data_record (
		id INTEGER PRIMARY KEY, site_id INTEGER, equipment_id INTEGER, alarm_status INTEGER,
		alarm_code TEXT, alarm_type TEXT, alarm_stage INTEGER, begin_time INTEGER, end_time INTEGER,
		alarm_value TEXT, recover_value TEXT, site_name TEXT, equipment_name TEXT, corp_id INTEGER
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`INSERT INTO alarm_data_record
		(id, site_id, equipment_id, alarm_status, alarm_code, alarm_type, alarm_stage, begin_time, end_time, alarm_value, site_name, equipment_name)
		VALUES (11, 1, 2, 1, 'TEMP_HIGH', '0', 3, 100, 0, '52.1', '成都站', '锂硫电池-01')`).Error; err != nil {
		t.Fatal(err)
	}

	controller := LiveAlarmCtrler{}
	found := callMigratedController(t, http.MethodPost, "/business/collect/findCollectLiveTriggerTempData?pageNumber=1&pageSize=10", map[string]any{
		"equipmentId": 2, "coding": "TEMP_HIGH",
	}, controller.Find)
	requireMigratedSuccess(t, found)
	if found.Return["total"] != float64(1) {
		t.Fatalf("实时告警数量错误: %#v", found.Return)
	}

	removed := callMigratedController(t, http.MethodPost, "/business/collect/removeCollectLiveTriggerTempData", map[string]any{"id": 11}, controller.Remove)
	requireMigratedSuccess(t, removed)
	var status int
	var endTime int64
	if err := db.Table("alarm_data_record").Select("alarm_status", "end_time").Where("id = ?", 11).Row().Scan(&status, &endTime); err != nil {
		t.Fatal(err)
	}
	if status != 0 || endTime <= 0 {
		t.Fatalf("实时告警未标记为已处理: status=%d endTime=%d", status, endTime)
	}
}

func TestUserEquipmentPermissionControllerAddFindDelete(t *testing.T) {
	db := newMigratedControllerDB(t)
	statements := []string{
		`ATTACH DATABASE ':memory:' AS public`,
		`CREATE TABLE public.users (id INTEGER PRIMARY KEY, username TEXT UNIQUE)`,
		`CREATE TABLE equipment (id INTEGER PRIMARY KEY, concern_user_ids TEXT, control_user_ids TEXT)`,
		`INSERT INTO public.users(id, username) VALUES (7, 'operator01')`,
		`INSERT INTO equipment(id, concern_user_ids, control_user_ids) VALUES (1, '{}', '{}'), (2, '{}', '{}')`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("准备设备授权数据失败: %v", err)
		}
	}

	controller := authcontroller.UserEquipmentPermissionCtrler{}
	added := callMigratedController(t, http.MethodPost, "/auth/business/userconcernequipment/add", map[string]any{
		"grantedUser": "operator01", "equipmentIds": []int{2, 1, 2, 0},
	}, controller.AddConcern)
	requireMigratedSuccess(t, added)

	found := callMigratedController(t, http.MethodPost, "/auth/business/userconcernequipment/find", map[string]any{
		"grantedUser": "operator01",
	}, controller.FindConcern)
	requireMigratedSuccess(t, found)
	ids, _ := found.Return["data"].([]any)
	if len(ids) != 2 || ids[0] != float64(1) || ids[1] != float64(2) {
		t.Fatalf("关注设备查询结果错误: %#v", ids)
	}

	deleted := callMigratedController(t, http.MethodPost, "/auth/business/userconcernequipment/delete", map[string]any{
		"grantedUser": "operator01", "equipmentIds": []int{1},
	}, controller.DeleteConcern)
	requireMigratedSuccess(t, deleted)

	var values pq.Int64Array
	if err := db.Table("equipment").Select("concern_user_ids").Where("id = ?", 1).Row().Scan(&values); err != nil {
		t.Fatal(err)
	}
	if len(values) != 0 {
		t.Fatalf("取消关注后仍残留用户: %#v", values)
	}
}
