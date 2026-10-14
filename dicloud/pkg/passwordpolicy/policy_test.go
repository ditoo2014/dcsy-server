package passwordpolicy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func restoreConfig(t *testing.T) {
	t.Helper()
	original := Current()
	t.Cleanup(func() {
		if err := Configure(original); err != nil {
			t.Fatal(err)
		}
	})
	if err := Configure(Config{Enabled: true, Mode: CalendarQuarter, Timezone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
}

func parseTime(t *testing.T, value string) time.Time {
	t.Helper()
	result, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

// UTC 服务时钟和北京时间必须在同一个时刻进入季度，不按修改后 90 天判断。
func TestCalendarQuarterBoundaries(t *testing.T) {
	restoreConfig(t)
	changedAt := parseTime(t, "2026-09-30T23:59:58+08:00")
	before := Evaluate(&changedAt, false, parseTime(t, "2026-09-30T15:59:59Z"))
	if before.PasswordChangeRequired || before.PasswordChangeReason != "" {
		t.Fatalf("季度边界前不应限制: %+v", before)
	}
	boundary := parseTime(t, "2026-10-01T00:00:00+08:00")
	if before.PasswordExpiresAt == nil || !before.PasswordExpiresAt.Equal(boundary) {
		t.Fatalf("应按北京时间季初到期: %+v", before)
	}
	for _, timestamp := range []string{"2026-09-30T16:00:00Z", "2026-10-01T00:00:00+08:00"} {
		status := Evaluate(&changedAt, false, parseTime(t, timestamp))
		if !status.PasswordChangeRequired || status.PasswordChangeReason != ReasonQuarterExpired {
			t.Fatalf("边界时刻必须受限: %+v", status)
		}
	}
	lastQuarter := parseTime(t, "2026-12-31T23:59:59+08:00")
	status := Evaluate(&lastQuarter, false, parseTime(t, "2027-01-01T00:00:00+08:00"))
	if !status.PasswordChangeRequired || status.PasswordChangeReason != ReasonQuarterExpired {
		t.Fatalf("跨年也必须进入新季度: %+v", status)
	}
	changedOnBoundary := boundary
	if Evaluate(&changedOnBoundary, false, boundary).PasswordChangeRequired {
		t.Fatal("季度第一秒修改成功后不能再次受限")
	}
}

func TestMissingForcedAndFuturePasswordState(t *testing.T) {
	restoreConfig(t)
	now := parseTime(t, "2026-10-09T12:00:00+08:00")
	if result := Evaluate(nil, false, now); !result.PasswordChangeRequired || result.PasswordChangeReason != ReasonPasswordUnset {
		t.Fatalf("存量账号不能伪造修改时间: %+v", result)
	}
	future := now.Add(time.Hour)
	if result := Evaluate(&future, false, now); !result.PasswordChangeRequired || result.PasswordChangeReason != ReasonFutureTime {
		t.Fatalf("异常未来时间不能绕过策略: %+v", result)
	}
	if err := Configure(Config{Enabled: false, Mode: CalendarQuarter, Timezone: "Asia/Shanghai"}); err != nil {
		t.Fatal(err)
	}
	if Evaluate(nil, false, now).PasswordChangeRequired {
		t.Fatal("关闭开关后季度限制应解除")
	}
	if result := Evaluate(&now, true, now); !result.PasswordChangeRequired || result.PasswordChangeReason != ReasonAdminReset {
		t.Fatalf("季度开关不能取消管理员重置限制: %+v", result)
	}
}

// 独立文件配置不能受到其他服务的全局 Viper 或环境变量覆盖。
func TestLoadFileRequiresExplicitConfiguration(t *testing.T) {
	restoreConfig(t)
	t.Setenv("PASSWORD_SECURITY_PASSWORD_ENABLED", "true")
	t.Setenv("AUTH_SECURITY_PASSWORD_ENABLED", "true")
	configPath := filepath.Join(t.TempDir(), "auth.yml")
	if err := os.WriteFile(configPath, []byte("security:\n  password:\n    enabled: false\n    mode: calendar_quarter\n    timezone: Asia/Shanghai\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := LoadFile(configPath); err != nil {
		t.Fatal(err)
	}
	if Current().Enabled {
		t.Fatal("环境变量不能覆盖文件中关闭的配置")
	}
	for _, invalid := range []string{
		"security:\n  password:\n    mode: calendar_quarter\n    timezone: Asia/Shanghai\n",
		"security:\n  password:\n    enabled: true\n    mode: rolling_days\n    timezone: Asia/Shanghai\n",
		"security:\n  password:\n    enabled: true\n    mode: calendar_quarter\n    timezone: UTC\n",
	} {
		if err := os.WriteFile(configPath, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		if err := LoadFile(configPath); err == nil {
			t.Fatal("缺失或错误的安全策略不能静默使用")
		}
	}
	if err := LoadFile(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("配置文件不存在时必须拒绝启动")
	}
}
