package battery

import "testing"

func TestParseTopicSupportsLithiumSulfurAndSodium(t *testing.T) {
	tests := []struct {
		topic string
		kind  TopicKind
	}{
		{topic: "/DCBMS4G/LithiumSulfurBattery/PUB/862288088847155", kind: LithiumSulfurTopic},
		{topic: "/DCBMS4G/SodiumBattery/PUB/862288088847155", kind: SodiumTopic},
	}
	for _, test := range tests {
		kind, gatewayID, ok := ParseTopic(test.topic)
		if !ok || kind != test.kind || gatewayID != "862288088847155" {
			t.Fatalf("topic=%s kind=%q gatewayID=%q ok=%t", test.topic, kind, gatewayID, ok)
		}
	}
	if _, _, ok := ParseTopic("/DCBMS4G/OtherBattery/PUB/862288088847155"); ok {
		t.Fatal("不支持的电池主题不应进入解析链路")
	}
}

func TestCID2MatchesSodiumLegacyConfiguration(t *testing.T) {
	if !cid2Matches(SodiumTopic, "42", "42_钠电") {
		t.Fatal("钠电 CID42 配置应能匹配")
	}
	if cid2Matches(SodiumTopic, "42", "42_锂硫电池") {
		t.Fatal("钠电主题不应匹配锂硫配置")
	}
}
