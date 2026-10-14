package util

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestInitViperFileOnlyIgnoresEnvironment(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)

	workDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("获取当前工作目录失败: %v", err)
	}
	tempDir := t.TempDir()
	t.Cleanup(func() {
		if err := os.Chdir(workDir); err != nil {
			t.Errorf("恢复工作目录失败: %v", err)
		}
	})

	configDir := filepath.Join(tempDir, "configs")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("创建配置目录失败: %v", err)
	}
	configFile := filepath.Join(configDir, "auth.yml")
	if err := os.WriteFile(configFile, []byte("sms:\n  login:\n    simulate: false\n"), 0600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("切换测试工作目录失败: %v", err)
	}
	t.Setenv("AUTH_SMS_LOGIN_SIMULATE", "true")

	if err := InitViperFileOnly("auth"); err != nil {
		t.Fatalf("读取配置失败: %v", err)
	}
	if viper.GetBool("sms.login.simulate") {
		t.Fatal("配置文件中的 false 被环境变量覆盖")
	}
	if used := filepath.Clean(viper.ConfigFileUsed()); used != filepath.Clean(configFile) {
		t.Fatalf("读取了错误的配置文件: %s", used)
	}
}
