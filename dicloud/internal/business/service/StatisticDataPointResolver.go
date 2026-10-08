package service

import (
	"dicloud/internal/tolink/dao"
	tolinkmodel "dicloud/internal/tolink/model"
	"log"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// statisticDataPoint describes a physical gateway data point and the
// canonical statistic code used by the statistic tables.
type statisticDataPoint struct {
	DataID string
	Code   string
}

// statisticDataPointResolver resolves gateway data IDs once per statistic run.
// Gateway data IDs may contain a channel/address prefix, so the fixed legacy
// IDs cannot be used for every battery.
type statisticDataPointResolver struct {
	ctx   *gin.Context
	cache map[string][]tolinkmodel.GatewayData
}

// newStatisticDataPointResolver creates a resolver bound to the current Gin context.
func newStatisticDataPointResolver(ctx *gin.Context) *statisticDataPointResolver {
	return &statisticDataPointResolver{ctx: ctx, cache: make(map[string][]tolinkmodel.GatewayData)}
}

var statisticPointAliases = map[string][]string{
	"tt_fangdiandianliang":  {"累计放电量", "累计放电容量"},
	"tt_chongdiancishu":     {"循环次数", "充电次数"},
	"tt_chongdiandianliang": {"累计充电量", "累计充电容量"},
	"alg_soh":               {"SOH"},
}

// resolve returns every physical candidate for each canonical statistic code.
// The caller selects the candidate with the latest valid reading because a
// gateway can retain data points for more than one channel/address.
func (r *statisticDataPointResolver) resolve(gatewayID string, legacyPoints []statisticDataPoint) []statisticDataPoint {
	if strings.TrimSpace(gatewayID) == "" {
		return nil
	}

	resolved := make([]statisticDataPoint, 0, len(legacyPoints))
	points, found := r.gatewayData(gatewayID)
	if !found || len(points) == 0 {
		log.Printf("[Business-Statistic] gateway data points unavailable, use legacy IDs gateway=%s", gatewayID)
		for _, point := range legacyPoints {
			resolved = append(resolved, point)
		}
		return resolved
	}

	for _, legacy := range legacyPoints {
		candidates := resolveStatisticPoints(points, legacy.DataID)
		if len(candidates) == 0 {
			log.Printf("[Business-Statistic] statistic data point not found gateway=%s code=%s legacyID=%s", gatewayID, legacy.Code, legacy.DataID)
			continue
		}
		for _, candidate := range candidates {
			resolved = append(resolved, statisticDataPoint{DataID: candidate, Code: legacy.Code})
		}
	}
	return resolved
}

func (r *statisticDataPointResolver) gatewayData(gatewayID string) ([]tolinkmodel.GatewayData, bool) {
	if points, ok := r.cache[gatewayID]; ok {
		return points, len(points) > 0
	}
	gatewayDao := dao.NewGatewayDAO(r.ctx)
	gateway, found := gatewayDao.FindGatewayDatas(tolinkmodel.Gateway{GatewayId: gatewayID})
	if !found || len(gateway.GatewayDatas) == 0 {
		r.cache[gatewayID] = nil
		return nil, false
	}
	r.cache[gatewayID] = gateway.GatewayDatas
	return gateway.GatewayDatas, true
}

func resolveStatisticPoint(points []tolinkmodel.GatewayData, legacyID string) string {
	candidates := resolveStatisticPoints(points, legacyID)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func resolveStatisticPoints(points []tolinkmodel.GatewayData, legacyID string) []string {
	legacyID = strings.ToLower(strings.TrimSpace(legacyID))
	type candidate struct {
		dataID string
		score  int
	}
	candidates := make([]candidate, 0)
	aliases := statisticPointAliases[legacyID]
	for _, point := range points {
		dataID := strings.TrimSpace(point.DataId)
		if dataID == "" {
			continue
		}
		id := strings.ToLower(dataID)
		score := -1
		switch {
		case id == legacyID:
			score = 0
		case strings.HasSuffix(id, "_"+legacyID):
			score = 1
		}
		name := strings.TrimSpace(point.Name)
		for _, alias := range aliases {
			if name == alias {
				if score < 0 || score > 2 {
					score = 2
				}
				break
			}
			if strings.HasSuffix(name, "_"+alias) {
				if score < 0 || score > 3 {
					score = 3
				}
				break
			}
		}
		if score >= 0 {
			candidates = append(candidates, candidate{dataID: dataID, score: score})
		}
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score < candidates[j].score
		}
		return candidates[i].dataID < candidates[j].dataID
	})
	dataIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		dataIDs = append(dataIDs, candidate.dataID)
	}
	return dataIDs
}
