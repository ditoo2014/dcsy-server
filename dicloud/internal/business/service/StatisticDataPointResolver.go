package service

import (
	"dicloud/internal/tolink/dao"
	tolinkmodel "dicloud/internal/tolink/model"
	"dicloud/util/pointid"
	"log"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
)

// statisticDataPoint 关联实际网关数据点与统计代码，统计表中的代码保持不变。
type statisticDataPoint struct {
	DataID string
	Code   string
}

// statisticDataPointResolver 在一次统计任务中缓存网关配置，支持英文 ID 的通道地址前缀。
type statisticDataPointResolver struct {
	ctx   *gin.Context
	cache map[string][]tolinkmodel.GatewayData
}

// newStatisticDataPointResolver creates a resolver bound to the current Gin context.
func newStatisticDataPointResolver(ctx *gin.Context) *statisticDataPointResolver {
	return &statisticDataPointResolver{ctx: ctx, cache: make(map[string][]tolinkmodel.GatewayData)}
}

var statisticPointAliases = map[string][]string{
	// SOH 保持既有识别规则；循环次数仍只按统一英文 ID 识别。
	"alg_soh": {"SOH"},
}

// 两项累计容量支持已有配置的名称，查询仍使用配置中的真实 ID，不写入或转换存量数据。
var cumulativeCapacityAliases = map[string][]string{
	pointid.TotalDischargeCapacity: {"累计放电量", "累计放电容量"},
	pointid.TotalChargeCapacity:    {"累计充电量", "累计充电容量"},
}

func matchesCumulativeCapacityName(baseID, name string) bool {
	name = strings.TrimSpace(name)
	// 只允许两位十六进制通道地址前缀，防止把月统计等同名派生指标识别为设备累计值。
	if len(name) > 3 && name[2] == '_' && strings.ContainsRune("0123456789abcdefABCDEF", rune(name[0])) && strings.ContainsRune("0123456789abcdefABCDEF", rune(name[1])) {
		name = name[3:]
	}
	for _, alias := range cumulativeCapacityAliases[baseID] {
		if name == alias {
			return true
		}
	}
	return false
}

// resolve returns every physical candidate for each canonical statistic code.
// The caller selects the candidate with the latest valid reading because a
// gateway can retain data points for more than one channel/address.
func (r *statisticDataPointResolver) resolve(gatewayID string, standardPoints []statisticDataPoint) []statisticDataPoint {
	if strings.TrimSpace(gatewayID) == "" {
		return nil
	}

	resolved := make([]statisticDataPoint, 0, len(standardPoints))
	points, found := r.gatewayData(gatewayID)
	if !found || len(points) == 0 {
		log.Printf("[Business-Statistic] 网关数据点配置暂不可用，按标准英文 ID 查询 gateway=%s", gatewayID)
		// 配置暂时不可用时仍只查询标准英文 ID，不回退到历史编码。
		resolved = append(resolved, standardPoints...)
		return resolved
	}

	for _, point := range standardPoints {
		candidates := resolveStatisticPoints(points, point.DataID)
		if len(candidates) == 0 {
			log.Printf("[Business-Statistic] statistic data point not found gateway=%s code=%s dataID=%s", gatewayID, point.Code, point.DataID)
			continue
		}
		for _, candidate := range candidates {
			resolved = append(resolved, statisticDataPoint{DataID: candidate, Code: point.Code})
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

func resolveStatisticPoint(points []tolinkmodel.GatewayData, baseID string) string {
	candidates := resolveStatisticPoints(points, baseID)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func resolveStatisticPoints(points []tolinkmodel.GatewayData, baseID string) []string {
	baseID = strings.ToLower(strings.TrimSpace(baseID))
	type candidate struct {
		dataID string
		score  int
	}
	candidates := make([]candidate, 0)
	aliases := statisticPointAliases[baseID]
	for _, point := range points {
		dataID := strings.TrimSpace(point.DataId)
		if dataID == "" {
			continue
		}
		id := strings.ToLower(dataID)
		score := -1
		switch {
		case id == baseID:
			score = 0
		case pointid.Matches(id, baseID):
			score = 1
		}
		name := strings.TrimSpace(point.Name)
		if score < 0 && matchesCumulativeCapacityName(baseID, name) {
			score = 2
		}
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
		// 累计容量按指标选择英文候选或名称候选，避免两套配置同时参与查询。
		if len(cumulativeCapacityAliases[baseID]) > 0 && candidates[0].score < 2 && candidate.score >= 2 {
			continue
		}
		dataIDs = append(dataIDs, candidate.dataID)
	}
	return dataIDs
}
