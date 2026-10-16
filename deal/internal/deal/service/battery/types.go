package battery

import "time"

type MQTTMessage struct {
	Version    int    `json:"version"`
	MessageID  string `json:"messageId"`
	GatewayID  string `json:"gatewayId"`
	Topic      string `json:"topic"`
	Payload    string `json:"payload"`
	ReceivedAt int64  `json:"receivedAt"`
}

type Point struct {
	DataID string
	Value  float64
	Status int
	TS     time.Time
}

type PointStr struct {
	DataID string
	Value  string
	Status int
	TS     time.Time
}

type ParsedData struct {
	GatewayID string
	Points    []Point
	StrPoints []PointStr
}

type BatteryAIRequest struct {
	PackNo             string    `json:"packNo"`
	Timestamp          string    `json:"timestamp"`
	CellNum            float64   `json:"cellNum"`
	TepNum             float64   `json:"tepNum"`
	BatteryVoltage     float64   `json:"batteryVoltage"`
	BatteryCurrent     float64   `json:"batteryCurrent"`
	BatteryTemperature float64   `json:"batteryTemperature"`
	BmsMosTemperature  float64   `json:"bmsMosTemperature"`
	Soc                float64   `json:"soc"`
	CellVoltages       []float64 `json:"cellVoltages"`
	CellTemperatures   []float64 `json:"cellTemperatures"`
	TotalCapacity      float64   `json:"DCZRL"`
}

type AIResult struct {
	PackNo                 string      `json:"packNo"`
	Timestamp              string      `json:"timestamp"`
	EvaluationTimestamp    string      `json:"evaluationTimestamp"`
	Soc                    float64     `json:"soc"`
	Soh                    float64     `json:"soh"`
	RemainingDischargeTime interface{} `json:"remainingDischargeTime"`
	ThermalRunawayRisk     string      `json:"thermalRunawayRisk"`
	BatteryEvaluationList  []struct {
		CellNo string  `json:"cellNo"`
		Soc    float64 `json:"soc"`
	} `json:"batteryEvaluationList"`
}
