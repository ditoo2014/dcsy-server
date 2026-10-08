package service

import (
	"dicloud/internal/common/dao"
	"dicloud/internal/common/model"
	bloomutil "dicloud/util/bloom"
	"strings"

	"github.com/gin-gonic/gin"
)

// HandleGeneralStringTriggerData evaluates string-valued general triggers.
// The legacy string handler only reads trigger_config; TTDC public rules are
// generated into general_trigger_config, so they need this parallel adapter.
func HandleGeneralStringTriggerData(ctx *gin.Context, gatewayID, dataID, value string, now int64) {
	marktrigger := strings.TrimSpace(gatewayID) + " " + strings.TrimSpace(dataID)
	if gatewayID == "" || dataID == "" || !bloomutil.BusinessGeneralTriggerBloomFilter.Test([]byte(marktrigger)) {
		return
	}

	triggerDAO := dao.NewGeneralTriggerDao(ctx)
	configs, ok := triggerDAO.FindByMarktrigger(marktrigger)
	if !ok {
		return
	}

	for _, config := range configs {
		if config.TriggerModel != 1 || len(config.StrMatchs) == 0 {
			continue
		}
		for _, mark := range config.Marktriggers {
			if mark.Marktrigger != marktrigger {
				continue
			}

			result, replacement, replace := matchGeneralStringTrigger(value, config.StrMatchs)
			unit := config
			if replace {
				unit.Code = replacement
				unit.AlarmUser.AlarmCode = replacement
				if unit.StatisticConfig.StatisticCode != "" {
					unit.StatisticConfig.StatisticCode = replacement
				}
			}
			normalSingleOrCycleGeneralJudge(ctx, unit, mark, int64(result), now, value, triggerDAO)
		}
	}
}

func matchGeneralStringTrigger(value string, matches []model.GeneralRealtimeStrMatch) (result int, replacement string, replace bool) {
	for _, item := range matches {
		if item.MatchStr == "" || value != item.MatchStr {
			continue
		}
		if item.ReplaceCodeFlag {
			return 1, item.ReplaceCode, true
		}
		return 1, "", false
	}
	return 0, "", false
}
