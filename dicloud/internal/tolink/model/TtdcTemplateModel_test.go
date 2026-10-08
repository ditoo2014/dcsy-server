package model

import (
	"strings"
	"testing"
)

func TestTtdcSchemaRuntimeUsable(t *testing.T) {
	for _, status := range []string{TtdcSchemaConfig, TtdcSchemaReady, " configuring "} {
		if !TtdcSchemaRuntimeUsable(status) {
			t.Fatalf("runtime status %q was rejected", status)
		}
	}
	for _, status := range []string{"", TtdcSchemaPreparing, TtdcSchemaFailed, "unknown"} {
		if TtdcSchemaRuntimeUsable(status) {
			t.Fatalf("non-runtime status %q was accepted", status)
		}
	}
}

func TestNormalizeTtdcADR(t *testing.T) {
	for _, value := range []string{"00", "0a", "FE", "ff"} {
		if normalized, ok := NormalizeTtdcADR(value); !ok || normalized != strings.ToUpper(value) {
			t.Fatalf("NormalizeTtdcADR(%q) = (%q, %t)", value, normalized, ok)
		}
	}
	for _, value := range []string{"", "0", "000", "GG"} {
		if _, ok := NormalizeTtdcADR(value); ok {
			t.Fatalf("invalid ADR accepted: %q", value)
		}
	}
}
