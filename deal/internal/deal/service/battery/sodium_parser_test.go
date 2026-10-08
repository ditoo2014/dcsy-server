package battery

import (
	"testing"
	"time"

	"deal/internal/tolink/model"
)

func TestParseSodiumCID42Packet(t *testing.T) {
	moduleCacheMu.Lock()
	moduleCache = map[string]moduleSnapshot{
		"862288088847155": {
			Modules: []model.GatewayModuleTtdc{{
				GatewayId: "862288088847155",
				TtdcConfig: model.ModuleTtdcConfig{
					ModuleSoi: "~", ModuleVer: "20", ModuleAdr: "02",
					ModuleCid1: "4A", ModuleCid2: "42_钠电",
				},
				ModuleDatas: []model.ModuleTtdcData{
					{DataId: "m", Name: "M（单体电池数量）", DataType: 1, Transport: "4 1 3", Formula: "x"},
					{DataId: "cell01", Name: "单体电池 1 电压", DataType: 1, Transport: "6 2 1", Formula: "x / 1000"},
					{DataId: "n", Name: "N（电芯温度数量）", DataType: 1, Transport: "70 1 3", Formula: "x"},
					{DataId: "temp01", Name: "电芯温度 1 数据", DataType: 1, Transport: "72 2 1", Formula: "(x - 2731) / 10"},
					{DataId: "current", Name: "电池电流数据", DataType: 1, Transport: "112 2 1", Formula: "x / 100"},
					{DataId: "voltage", Name: "电池总压数据", DataType: 1, Transport: "116 2 3", Formula: "x / 100"},
				},
			}},
		},
	}
	moduleCacheMu.Unlock()
	t.Cleanup(func() {
		moduleCacheMu.Lock()
		moduleCache = map[string]moduleSnapshot{}
		moduleCacheMu.Unlock()
	})

	packet := "~20024A420000FDA1\r~20024A0020860002100CAB0CAB0CA70CAB0CAA0CB00CAB0CAA0CA90CB00CAF0CB00CAE0CAE0CB00CA2080BC30BAF0BAF0BB90BB90BAF0BAF0BC30BEB0BE1FE961446231223F0000200DF0F"
	receivedAt := time.Unix(1787025260, 987654321)
	parsed, err := ParsePacketAt("/DCBMS4G/SodiumBattery/PUB/862288088847155", packet, receivedAt)
	if err != nil {
		t.Fatal(err)
	}

	values := make(map[string]float64, len(parsed.Points))
	for _, point := range parsed.Points {
		values[point.DataID] = point.Value
		if point.TS.Unix() != receivedAt.Unix() || point.TS.Nanosecond() != 0 {
			t.Fatalf("数据点 %s 的时间戳不正确: %v", point.DataID, point.TS)
		}
	}
	want := map[string]float64{
		"m": 16, "n": 8, "cell01": 3.243, "temp01": 28, "current": -3.62, "voltage": 51.9,
	}
	for dataID, expected := range want {
		if values[dataID] != expected {
			t.Fatalf("数据点 %s = %v, 期望 %v", dataID, values[dataID], expected)
		}
	}
}
