package schedule

import (
	"dicloud/internal/business/client"
	"dicloud/internal/business/service"
	"dicloud/internal/common/grpc/tolink_grpc"
	"dicloud/internal/common/model"
	commonservice "dicloud/internal/common/service"
	"encoding/json"
	"log"
	"strings"

	"github.com/gin-gonic/gin"
)

// RefreshDatasource refreshes generated general-trigger mark bindings.
func RefreshDatasource() {
	// Synchronize platform TTDC alarm definitions before refreshing the legacy
	// label-name based trigger bindings. Generated TTDC triggers use an internal
	// label name and therefore remain outside the legacy matching semantics.
	refreshTtdcAlarmRules()

	ctx := &gin.Context{}
	doRefresh(ctx)

	dbNameDatas := commonservice.LoadingAllPlat()
	for _, dbTem := range dbNameDatas {
		ctxTem := &gin.Context{}
		ctxTem.Set("dbName", dbTem)
		doRefresh(ctxTem)
	}
}

func doRefresh(ctx *gin.Context) {
	datas, err := client.FindAllGeneralTriggerMarkBinds(ctx, &model.GeneralTriggerConfig{})
	if err != nil || datas == nil {
		log.Println("[BUSINESS] schedule refresh generaltrigger fail: ", err)
		return
	}

	for _, trigger := range datas.Triggers {
		if trigger == nil {
			continue
		}
		syncGeneralTriggerMarktriggers(ctx, trigger)
	}
}

func syncGeneralTriggerMarktriggers(ctx *gin.Context, trigger *tolink_grpc.GeneralTriggerConfig) bool {
	if isGeneratedTTDCTrigger(trigger) {
		return true
	}
	gmts := service.GetGeneralMarktriggerListByLabelName(ctx, trigger.EquipmentIds, trigger.LabelName)
	if len(gmts) == 0 {
		return true
	}

	if len(trigger.Marktriggers) == 0 {
		return createGeneralMarktriggers(ctx, trigger, gmts)
	}

	success := true
	updatedEquipmentIds := map[int64]bool{}
	for _, mark := range trigger.Marktriggers {
		if mark == nil {
			continue
		}
		if updatedEquipmentIds[mark.EquipmentId] {
			log.Printf("[BUSINESS] GeneralTrigger duplicate marktrigger skipped, triggerId=%d equipmentId=%d marktriggerId=%d", trigger.Id, mark.EquipmentId, mark.Id)
			continue
		}
		updatedEquipmentIds[mark.EquipmentId] = true

		current, exist := findGeneratedGeneralMarktrigger(gmts, mark.EquipmentId)
		if !exist {
			log.Printf("[BUSINESS] GeneralTrigger keep unmatched marktrigger, triggerId=%d equipmentId=%d marktriggerId=%d", trigger.Id, mark.EquipmentId, mark.Id)
			continue
		}

		next := baseGeneralMarktriggerDataFormat(current, trigger.Id)
		next.Id = mark.Id
		if !generalMarktriggerBaseInfoChanged(mark, &next) {
			continue
		}
		if !updateGeneralMarktrigger(ctx, trigger, &next) {
			success = false
		}
	}

	for _, generated := range gmts {
		if generatedGeneralMarktriggerExists(trigger.Marktriggers, generated.EquipmentId) {
			continue
		}

		next := baseGeneralMarktriggerDataFormat(generated, trigger.Id)
		if !createGeneralMarktrigger(ctx, trigger, &next) {
			success = false
		}
	}

	return success
}

func isGeneratedTTDCTrigger(trigger *tolink_grpc.GeneralTriggerConfig) bool {
	if trigger == nil {
		return false
	}
	return strings.HasPrefix(strings.TrimSpace(trigger.LabelName), ttdcGeneratedTriggerPrefix)
}

func createGeneralMarktriggers(ctx *gin.Context, trigger *tolink_grpc.GeneralTriggerConfig, gmts []model.GeneralMarktrigger) bool {
	param := &tolink_grpc.GeneralTriggerConfig{
		DbName: trigger.DbName,
	}
	for idx := range gmts {
		next := baseGeneralMarktriggerDataFormat(gmts[idx], trigger.Id)
		param.Marktriggers = append(param.Marktriggers, &next)
	}

	resp, err := client.CreateMarkTrigger(ctx, param)
	if err != nil || resp == nil || resp.Code != 200 {
		log.Println("[BUSINESS] GeneralTrigger sync batch create marktrigger fail:", err)
		return false
	}
	return true
}

func updateGeneralMarktrigger(ctx *gin.Context, trigger *tolink_grpc.GeneralTriggerConfig, mark *tolink_grpc.GeneralMarktrigger) bool {
	resp, err := client.UpdateMarkTrigger(ctx, &tolink_grpc.GeneralTriggerConfig{
		DbName:       trigger.DbName,
		Marktriggers: []*tolink_grpc.GeneralMarktrigger{mark},
	})
	if err != nil || resp == nil || resp.Code != 200 {
		log.Println("[BUSINESS] GeneralTrigger sync update marktrigger fail:", err)
		return false
	}
	return true
}

func createGeneralMarktrigger(ctx *gin.Context, trigger *tolink_grpc.GeneralTriggerConfig, mark *tolink_grpc.GeneralMarktrigger) bool {
	resp, err := client.CreateMarkTrigger(ctx, &tolink_grpc.GeneralTriggerConfig{
		DbName:       trigger.DbName,
		Marktriggers: []*tolink_grpc.GeneralMarktrigger{mark},
	})
	if err != nil || resp == nil || resp.Code != 200 {
		log.Println("[BUSINESS] GeneralTrigger sync create marktrigger fail:", err)
		return false
	}
	return true
}

func findGeneratedGeneralMarktrigger(gmts []model.GeneralMarktrigger, equipmentID int64) (model.GeneralMarktrigger, bool) {
	for _, item := range gmts {
		if int64(item.EquipmentId) == equipmentID {
			return item, true
		}
	}
	return model.GeneralMarktrigger{}, false
}

func generatedGeneralMarktriggerExists(marks []*tolink_grpc.GeneralMarktrigger, equipmentID int) bool {
	for _, mark := range marks {
		if mark != nil && mark.EquipmentId == int64(equipmentID) {
			return true
		}
	}
	return false
}

func generalMarktriggerBaseInfoChanged(current, next *tolink_grpc.GeneralMarktrigger) bool {
	if current == nil || next == nil {
		return false
	}

	return current.SiteId != next.SiteId ||
		current.EquipmentId != next.EquipmentId ||
		current.Marktrigger != next.Marktrigger
}

func baseGeneralMarktriggerDataFormat(gtc model.GeneralMarktrigger, triggerId int64) (dataTem tolink_grpc.GeneralMarktrigger) {
	dataStr, err := json.Marshal(gtc)
	if err != nil {
		log.Println("[GRPC-Schedule] A transform data fail, may have fields ignored")
	}

	err = json.Unmarshal(dataStr, &dataTem)
	if err != nil {
		log.Println("[GRPC-Schedule] B transform data fail, may have fields ignored")
		return
	}

	dataTem.TriggerId = triggerId
	return
}
