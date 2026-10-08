package battery

import "strings"

type TopicKind string

const (
	LithiumSulfurTopic TopicKind = "lithium_sulfur"
	SodiumTopic        TopicKind = "sodium"
)

const (
	LithiumSulfurTopicPrefix = "/DCBMS4G/LithiumSulfurBattery/PUB/"
	SodiumTopicPrefix        = "/DCBMS4G/SodiumBattery/PUB/"
)

func ParseTopic(topic string) (TopicKind, string, bool) {
	segment, gatewayID, ok := ParseBatteryTopic(topic)
	if !ok {
		return "", "", false
	}
	switch segment {
	case "LithiumSulfurBattery":
		return LithiumSulfurTopic, gatewayID, true
	case "SodiumBattery":
		return SodiumTopic, gatewayID, true
	default:
		return "", "", false
	}
}

// ParseBatteryTopic recognizes the fixed four-part battery topic structure.
// The second segment is configured by public templates and is not hardcoded.
func ParseBatteryTopic(topic string) (topicSegment, gatewayID string, ok bool) {
	normalized := strings.Trim(strings.TrimSpace(topic), "/")
	parts := strings.Split(normalized, "/")
	if len(parts) != 4 || parts[0] != "DCBMS4G" || parts[2] != "PUB" {
		return "", "", false
	}
	topicSegment = strings.TrimSpace(parts[1])
	gatewayID = strings.TrimSpace(parts[3])
	if topicSegment == "" || !ValidateGatewayID(gatewayID) || strings.ContainsAny(topicSegment+gatewayID, " \t\r\n") {
		return "", "", false
	}
	return topicSegment, gatewayID, true
}

func ParseLegacyTopic(topic string) (TopicKind, string, bool) {
	normalized := strings.TrimSuffix(strings.TrimSpace(topic), "/")
	for _, item := range []struct {
		prefix string
		kind   TopicKind
	}{
		{prefix: LithiumSulfurTopicPrefix, kind: LithiumSulfurTopic},
		{prefix: SodiumTopicPrefix, kind: SodiumTopic},
	} {
		if !strings.HasPrefix(normalized, item.prefix) {
			continue
		}
		gatewayID := strings.TrimPrefix(normalized, item.prefix)
		if gatewayID == "" || strings.Contains(gatewayID, "/") {
			return "", "", false
		}
		return item.kind, gatewayID, true
	}
	return "", "", false
}

func IsBatteryTopicPrefix(prefix string) bool {
	return prefix == LithiumSulfurTopicPrefix || prefix == SodiumTopicPrefix
}
