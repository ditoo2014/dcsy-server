package controller

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"dicloud/internal/tolink/model"

	"github.com/gin-gonic/gin"
)

func TestHistoryTableCheckResponses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, function := range []string{"countdata", "historydata", "newdata"} {
		for _, state := range []string{"已存在", "空库", "其他网关", "查询失败"} {
			t.Run(function+"/"+state, func(t *testing.T) {
				recorder := httptest.NewRecorder()
				ctx, _ := gin.CreateTestContext(recorder)
				point := model.GatewayData{GatewayId: "GW01", DataId: "00_voltage"}
				query := func() ([]string, error) {
					switch state {
					case "已存在":
						return []string{"gw01"}, nil
					case "其他网关":
						return []string{"GW02"}, nil
					case "查询失败":
						return nil, errors.New("数据库连接中断")
					default:
						return []string{}, nil
					}
				}
				proceed := checkHistoryTable(ctx, function, point, query)
				if state == "已存在" {
					if !proceed || recorder.Body.Len() != 0 {
						t.Fatal("已有历史表应继续查询，不提前写响应")
					}
					return
				}
				if proceed || recorder.Code != 200 {
					t.Fatal("无表或查询失败应终止后续查询并保持原 HTTP 协议")
				}
				var body struct {
					Code   int                    `json:"code"`
					Msg    string                 `json:"msg"`
					Return map[string]interface{} `json:"return"`
				}
				if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if state == "查询失败" {
					if body.Code != 400 || !strings.Contains(body.Msg, "查询历史数据表失败") || body.Return != nil {
						t.Fatalf("数据库故障不应伪装成无数据：%s", recorder.Body.String())
					}
					return
				}
				if body.Code != 200 || body.Return["data"] != point.DataId {
					t.Fatalf("空数据协议错误：%s", recorder.Body.String())
				}
				if function != "newdata" && body.Return["count"] != float64(0) {
					t.Fatal("未建表时应明确返回零条")
				}
				if function != "countdata" {
					for _, key := range []string{"times", "values", "status"} {
						array, ok := body.Return[key].([]interface{})
						if !ok || len(array) != 0 {
							t.Fatalf("%s 应返回空数组", key)
						}
					}
				}
			})
		}
	}
}
