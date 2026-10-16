package battery

import (
	"bytes"
	"deal/internal/batterylog"
	"deal/util/pointid"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

const unavailableRemainingDischargeTime float64 = 999

func appendAIResultPoints(result *ParsedData, adr string, req BatteryAIRequest, receivedAt time.Time) {
	if !viper.GetBool("battery.ai.enabled") {
		return
	}
	aiResults, err := postAI(req)
	if err != nil {
		batterylog.Printf("[BATTERY] AI request failed: %v\n", err)
		return
	}
	for _, item := range aiResults {
		// AI results belong to the source packet, not to the time the AI call
		// finished. This keeps retries and delayed consumption idempotent.
		ts := receivedAt
		result.StrPoints = append(result.StrPoints, PointStr{
			DataID: algorithmDataID(adr, pointid.ThermalRunawayRiskLevel),
			Value:  item.ThermalRunawayRisk,
			Status: 1,
			TS:     ts,
		})

		soc := item.Soc
		if req.TotalCapacity != 0 {
			soc = math.Round((req.Soc / req.TotalCapacity) * 100)
		}
		result.Points = append(result.Points,
			Point{DataID: adr + "_alg_soc", Value: soc, Status: 1, TS: ts},
			//Point{DataID: adr + "_alg_soh", Value: item.Soh, Status: 1, TS: ts},
			Point{DataID: adr + "_alg_soh", Value: 100, Status: 1, TS: ts}, //2026.08.11 硬编码100  ,上限就取消上面的注释
			Point{DataID: algorithmDataID(adr, pointid.RemainingDischargeTime), Value: remainingDischargeTimeValue(item.RemainingDischargeTime), Status: 1, TS: ts},
		)

		for _, cell := range item.BatteryEvaluationList {
			cellNo, err := strconv.Atoi(cell.CellNo)
			if err != nil || cellNo < 1 || cellNo > 16 {
				continue
			}
			result.Points = append(result.Points, Point{
				DataID: fmt.Sprintf("%s_alg_cell%02d_soc", adr, cellNo),
				Value:  cell.Soc,
				Status: 1,
				TS:     ts,
			})
		}
	}
}

type algorithmResult struct {
	Number *float64
	Text   *string
}

// evaluateAI returns stable algorithm keys. The caller maps those keys through
// the current template group's actual 42_SOC channel.
func evaluateAI(req BatteryAIRequest) map[string]algorithmResult {
	if !viper.GetBool("battery.ai.enabled") {
		return make(map[string]algorithmResult)
	}
	results, err := postAI(req)
	if err != nil {
		batterylog.Printf("[BATTERY] AI request failed: %v\n", err)

		return make(map[string]algorithmResult)
	}
	if req.TotalCapacity != 0 {
		results[0].Soc = math.Round((req.Soc / req.TotalCapacity) * 100) //2026.08.28临时使用需要
	}

	return algorithmResultsFromAI(req, results)
}

func algorithmResultsFromAI(req BatteryAIRequest, results []AIResult) map[string]algorithmResult {
	values := make(map[string]algorithmResult)
	if len(results) == 0 {
		return values
	}
	if len(results) > 1 {
		batterylog.Printf("[BATTERY-AI] received %d result objects for one battery request; only the first is used\n", len(results))
	}

	item := results[0]
	risk := item.ThermalRunawayRisk
	values[pointid.ThermalRunawayRiskLevel] = algorithmResult{Text: &risk}
	values["alg_soc"] = numericAlgorithmResult(item.Soc)
	//values["alg_soh"] = numericAlgorithmResult(item.Soh)
	values["alg_soh"] = numericAlgorithmResult(100) //2026.08.28临时使用需要
	values[pointid.RemainingDischargeTime] = numericAlgorithmResult(remainingDischargeTimeValue(item.RemainingDischargeTime))

	expectedCellCount := int(req.CellNum)
	seenCells := make(map[int]bool, expectedCellCount)
	for _, cell := range item.BatteryEvaluationList {
		cellNo, parseErr := strconv.Atoi(cell.CellNo)
		if parseErr != nil || cellNo < 1 || cellNo > expectedCellCount {
			batterylog.Printf("[BATTERY-AI] ignored invalid cellNo=%q expectedRange=1..%d\n", cell.CellNo, expectedCellCount)
			continue
		}
		if seenCells[cellNo] {
			batterylog.Printf("[BATTERY-AI] ignored duplicate cellNo=%d\n", cellNo)
			continue
		}
		seenCells[cellNo] = true
		values[fmt.Sprintf("alg_cell%02d_soc", cellNo)] = numericAlgorithmResult(cell.Soc)
	}
	if expectedCellCount > 0 && len(seenCells) != expectedCellCount {
		batterylog.Printf("[BATTERY-AI] incomplete per-cell SOC result expected=%d actual=%d\n", expectedCellCount, len(seenCells))
	}
	return values
}

func numericAlgorithmResult(value float64) algorithmResult { return algorithmResult{Number: &value} }

func postAI(req BatteryAIRequest) ([]AIResult, error) {
	url := viper.GetString("battery.ai.url")
	if url == "" {
		url = "http://172.17.7.191:12024/api/batteryEvaluation"
	}
	timeout := viper.GetDuration("battery.ai.timeout")
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	body, err := json.Marshal([]BatteryAIRequest{req})
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("AI status code: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(respBody) == 0 {
		return nil, fmt.Errorf("AI response body is empty")
	}

	var out []AIResult
	if err := json.Unmarshal(respBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func remainingDischargeTimeValue(value interface{}) float64 {
	// Keep the legacy contract: a string from the AI service means "not discharging"
	// and is stored as the numeric sentinel 999. Do not parse numeric strings here.
	if _, ok := value.(string); ok {
		return unavailableRemainingDischargeTime
	}
	numericValue, ok := value.(float64)
	if !ok {
		return unavailableRemainingDischargeTime
	}
	return numericValue
}
