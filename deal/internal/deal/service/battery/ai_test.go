package battery

import (
	"encoding/json"
	"testing"
)

func TestRemainingDischargeTimeValueKeepsLegacySemantics(t *testing.T) {
	tests := []struct {
		name  string
		input interface{}
		want  float64
	}{
		{name: "float value", input: float64(35.5), want: 35.5},
		{name: "not discharging text", input: "未放电", want: unavailableRemainingDischargeTime},
		// The old project treats every string as a state marker, including numeric text.
		{name: "numeric text", input: "35", want: unavailableRemainingDischargeTime},
		{name: "missing value", input: nil, want: unavailableRemainingDischargeTime},
		{name: "unexpected integer type", input: int(35), want: unavailableRemainingDischargeTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := remainingDischargeTimeValue(tt.input); got != tt.want {
				t.Fatalf("remainingDischargeTimeValue(%#v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRemainingDischargeTimeValueFromAIJSON(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want float64
	}{
		{name: "number", raw: `[{"remainingDischargeTime":35.5}]`, want: 35.5},
		{name: "numeric text", raw: `[{"remainingDischargeTime":"35.5"}]`, want: unavailableRemainingDischargeTime},
		{name: "state text", raw: `[{"remainingDischargeTime":"未放电"}]`, want: unavailableRemainingDischargeTime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result []AIResult
			if err := json.Unmarshal([]byte(tt.raw), &result); err != nil {
				t.Fatal(err)
			}
			if got := remainingDischargeTimeValue(result[0].RemainingDischargeTime); got != tt.want {
				t.Fatalf("remainingDischargeTimeValue(%s) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestAlgorithmResultsFromAISupportsDynamicCellCount(t *testing.T) {
	var results []AIResult
	raw := `[{
		"soc":73.25,
		"soh":91.5,
		"remainingDischargeTime":42.5,
		"thermalRunawayRisk":"low",
		"batteryEvaluationList":[
			{"cellNo":"1","soc":72.1},
			{"cellNo":"24","soc":74.8},
			{"cellNo":"24","soc":1},
			{"cellNo":"25","soc":1}
		]
	}]`
	if err := json.Unmarshal([]byte(raw), &results); err != nil {
		t.Fatal(err)
	}

	values := algorithmResultsFromAI(DcToAIData{CellNum: 24, Soc: 100, DCZRL: 200}, results)
	assertAlgorithmNumber(t, values, "alg_soc", 73.25)
	// The current battery contract intentionally exposes SOH as 100 until the
	// platform enables the algorithm-provided SOH value.
	assertAlgorithmNumber(t, values, "alg_soh", 100)
	assertAlgorithmNumber(t, values, "alg_cell01_soc", 72.1)
	assertAlgorithmNumber(t, values, "alg_cell24_soc", 74.8)
	if _, ok := values["alg_cell25_soc"]; ok {
		t.Fatal("AI result outside the requested 24-cell range was accepted")
	}
}

func assertAlgorithmNumber(t *testing.T, values map[string]algorithmResult, key string, want float64) {
	t.Helper()
	value, ok := values[key]
	if !ok || value.Number == nil || *value.Number != want {
		t.Fatalf("algorithm result %s = %+v, want %v", key, value, want)
	}
}
