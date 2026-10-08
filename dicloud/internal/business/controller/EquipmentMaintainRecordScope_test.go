package controller

import (
	"net/http/httptest"
	"testing"

	usermodel "dicloud/models/model"
	pq "gitee.com/opengauss/openGauss-connector-go-pq"
	"github.com/gin-gonic/gin"
)

// 没有真实部门的普通用户不能从部门 0 展开组织树；此测试不初始化数据库或缓存。
func TestMaintainUserScopeRejectsInvalidIdentityAndSkipsEmptyDepartments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, value := range []any{nil, "invalid", usermodel.User{}} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Set("userInfo", value)
		if _, _, ok := maintainUserScope(ctx); ok {
			t.Fatalf("accepted invalid identity: %+v", value)
		}
	}
	for _, ids := range []pq.Int64Array{nil, {0}, {-1, 0}} {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Set("userInfo", usermodel.User{ID: 1, CorpIds: ids})
		_, scope, ok := maintainUserScope(ctx)
		if !ok || len(scope) != 0 {
			t.Fatalf("unexpected scope for %v: %v, %v", ids, scope, ok)
		}
	}
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("userInfo", usermodel.User{ID: 1, Isadmin: 1})
	user, scope, ok := maintainUserScope(ctx)
	if !ok || user.Isadmin != 1 || len(scope) != 0 {
		t.Fatal("administrator scope changed")
	}
}
