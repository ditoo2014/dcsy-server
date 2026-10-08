package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// 使用真实路由验证匿名上传被提前拒绝，不访问数据库、不创建上传文件。
func TestImageUploadRequiresLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := ConnectRoute(gin.New())
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/go/images/upload", nil))
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"msg"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Code != 400 || result.Message != "auth header格式错误" {
		t.Fatalf("匿名上传未在登录校验处拦截：%s", recorder.Body.String())
	}
}
