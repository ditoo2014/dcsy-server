package main

import (
	"encoding/json"
	"testing"

	"github.com/spf13/viper"
)

func TestBuildBatteryTtdcMessagePreservesPacket(t *testing.T) {
	topic := "/DCBMS4G/SodiumBattery/PUB/862288088847155"
	payload := "~20024A420000FDA1\r\n~20024A0020860002100CAB0CAB0CA70DF0F"
	body, messageID, err := buildBatteryTtdcMessage(topic, payload, "862288088847155", 1787025260)
	if err != nil {
		t.Fatal(err)
	}
	var message batteryTtdcMQTTMessage
	if err := json.Unmarshal([]byte(body), &message); err != nil {
		t.Fatal(err)
	}
	if messageID == "" || message.MessageID != messageID || message.Topic != topic || message.Payload != payload {
		t.Fatalf("消息包装结果不正确: %+v", message)
	}
}

func TestParseBatteryCandidateTopicSupportsConfiguredSegments(t *testing.T) {
	viper.Set("battery.candidateRoutingMode", "allowlist")
	viper.Set("battery.candidateTopicSegments", []string{"SodiumBattery", "LithiumSulfurBattery"})
	t.Cleanup(viper.Reset)

	for _, segment := range []string{"SodiumBattery", "LithiumSulfurBattery"} {
		topic := "/DCBMS4G/" + segment + "/PUB/860000000000001"
		actualSegment, gatewayID, ok := parseBatteryCandidateTopic(topic)
		if !ok || actualSegment != segment || gatewayID != "860000000000001" {
			t.Fatalf("topic=%s segment=%q gatewayID=%q ok=%t", topic, actualSegment, gatewayID, ok)
		}
	}
	if _, _, ok := parseBatteryCandidateTopic("/DCBMS4G/OtherBattery/PUB/860000000000001"); ok {
		t.Fatal("白名单之外的主题不应进入处理队列")
	}
}

func TestValidateBatteryRoutingConfig(t *testing.T) {
	viper.Set("battery.enabled", true)
	viper.Set("battery.subscribeFilter", "/DCBMS4G/+/PUB/+")
	viper.Set("battery.candidateRoutingMode", "allowlist")
	viper.Set("battery.candidateTopicSegments", []string{"SodiumBattery", "LithiumSulfurBattery"})
	t.Cleanup(viper.Reset)

	if err := validateBatteryRoutingConfig(); err != nil {
		t.Fatalf("合法配置被拒绝: %v", err)
	}
	viper.Set("battery.candidateTopicSegments", []string{})
	if err := validateBatteryRoutingConfig(); err == nil {
		t.Fatal("空白名单不应通过校验")
	}
}
