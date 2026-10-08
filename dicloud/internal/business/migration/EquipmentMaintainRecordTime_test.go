package migration

import "testing"

func TestMaintainRecordTimeJSONConversionValidation(t *testing.T) {
	for _, raw := range []string{"0", "1726862400", `"1726862400"`, "null", "-1", "9223372036854775807"} {
		if err := validateMaintainRecordTimeJSON(raw); err != nil {
			t.Fatalf("valid value %s rejected: %v", raw, err)
		}
	}
	for _, raw := range []string{"{}", "[]", "true", `"bad"`, "12.5", "1e3", "9223372036854775808", `"12\u0033"`, "", `""`} {
		if err := validateMaintainRecordTimeJSON(raw); err == nil {
			t.Fatalf("unsafe value %s accepted", raw)
		}
	}
}
