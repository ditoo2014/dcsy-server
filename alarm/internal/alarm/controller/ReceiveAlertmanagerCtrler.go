package controller

import (
	"alarm/internal/alarm/service"
	"alarm/internal/common/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

/* 接受Alertmanager报警消息 */
type IReceiveAlertmanagerCtrler interface {
	ReceiveAlertmanager(ctx *gin.Context)
}

type ReceiveAlertmanagerCtrler struct {
}

// 根据body定义存储alertmanager的 json 结构体
type Alert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
}

type AlertNotification struct {
	Receiver          string            `json:"receiver"`
	Status            string            `json:"status"`
	Alerts            []Alert           `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string            `json:"externalURL"`
	Version           string            `json:"version"`
	GroupKey          string            `json:"groupKey"`
	TruncatedAlerts   int               `json:"truncatedAlerts"`
}

// 参考: https://blog.csdn.net/nange_nice/article/details/140977347
// 需求: 主要需求, 用户觉得报警消息过多, 可以通过Alertmanager来管理, 过滤; 尝试实现效果
// TODO: 配置自定义规则, 过滤报警消息
func (r ReceiveAlertmanagerCtrler) ReceiveAlertmanager(ctx *gin.Context) {
	fmt.Println("line 18 in check: ")

	var data AlertNotification
	if err := ctx.ShouldBindJSON(&data); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		fmt.Println("[ALARM] receive altermanager msg fail.")
		return
	}

	// 打印请求体中的原始JSON数据
	fmt.Println("Received JSON data: ", data)

	// 取到JSON数据
	if len(data.Alerts) > 0 {
		for idx := range data.Alerts {
			// 根据prometheus配置的规则, 读取数据
			if len(data.Alerts[idx].Annotations) > 0 {
				var jsonData = data.Alerts[idx].Annotations["json"]
				// 获取站点id, 设备id, coding, value等
				jsonData = strings.ReplaceAll(jsonData, "\\", "")
				var mapTem = make(map[string]any)
				err := json.Unmarshal([]byte(jsonData), &mapTem)
				if err != nil {
					fmt.Println("[ALARM] receive altermanager transform msg fail: ", err)
					continue
				}

				siteId, _ := strconv.ParseInt(fmt.Sprint(mapTem["siteId"]), 0, 64)
				equipmentId, _ := strconv.ParseInt(fmt.Sprint(mapTem["equipmentId"]), 0, 64)
				// 存库数据
				// alarmData := model.AlarmDataRecord{
				// 	SiteId:      int(siteId),
				// 	EquipmentId: int(equipmentId),
				// 	AlarmCode:   fmt.Sprint(mapTem["coding"]),
				// 	BeginTime:   time.Now().Unix(),
				// 	EndTime:     1,
				// 	AlarmValue:  fmt.Sprint(mapTem["value"]),
				// }
				msgDataTem := model.AlarmMQModel{
					SiteId:      int(siteId),
					EquipmentId: int(equipmentId),
					AlarmCode:   fmt.Sprint(mapTem["coding"]),
					Begin:       time.Now().Unix(),
					End:         1,
					AlarmValue:  fmt.Sprint(mapTem["value"])}

				curCtx := &gin.Context{}
				if mapTem["value"] != nil {
					curCtx.Set("dbName", fmt.Sprint(mapTem["value"]))
				}
				// 查询报警配置, 获取报警取到, 通知报警人
				service.AliSendSms(curCtx, &msgDataTem)
			}
		}
	}

}
