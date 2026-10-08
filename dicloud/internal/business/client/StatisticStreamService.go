package client

import (
	"dicloud/internal/common/dto"
	requestutil "dicloud/util/http"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// 统计池数据添加
func requestStatisticStreamDataAdd(statisticDataList []dto.StatisticData, equipmentId int) {
	if len(statisticDataList) == 0 {
		return
	}
	statisticDataList[0].Notes = fmt.Sprint(equipmentId)
	// 默认统计池数据
	timeStr := time.Now().Format("2006-01")
	t, _ := time.Parse("2006-01", timeStr)
	timeStr = t.Format(time.DateTime)
	for k, v := range dto.DP_DefaultStatisticDataMap {
		var statisticData = dto.StatisticData{
			GatewayID:  statisticDataList[0].GatewayID,
			Point:      k,
			StatsName:  v,
			Type:       2,
			StartTime:  timeStr,
			DeviceType: "in",
			Notes:      fmt.Sprint(equipmentId),
		}

		statisticDataList = append(statisticDataList, statisticData)
	}

	// 访问接口添加
	dataStr, err := json.Marshal(statisticDataList)
	if err != nil {
		log.Println("[GRPC-TolinkClient] statistic Stream data format fail")
		return
	}
	resBytes, err := requestutil.HttpPost("http://localhost:80/go/stats/accumulate/create", string(dataStr))
	if err != nil {
		log.Println("[GRPC-TolinkClient] statistic Stream data add fail")
	} else {
		log.Println("[GRPC-TolinkClient] statistic Stream data add success: ", string(resBytes))
	}

}
