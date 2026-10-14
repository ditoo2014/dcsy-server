// Package passwordpolicy 统一认证、业务和采集接口的季度改密规则。
package passwordpolicy

import (
	"fmt"
	"sync"
	"time"
	_ "time/tzdata" // 将时区数据编入程序，避免精简服务器镜像缺少北京时间规则。

	"github.com/spf13/viper"
)

const (
	CalendarQuarter      = "calendar_quarter"
	ChangeRequiredCode   = "PASSWORD_CHANGE_REQUIRED"
	ContextStatusKey     = "passwordStatus"
	ReasonQuarterExpired = "quarter_expired"
	ReasonPasswordUnset  = "password_unset"
	ReasonAdminReset     = "admin_reset"
	ReasonFutureTime     = "future_time"
)

// Config 来自各服务配置文件的 security.password，各服务的规则必须保持一致。
type Config struct {
	Enabled  bool   `mapstructure:"enabled"`
	Mode     string `mapstructure:"mode"`
	Timezone string `mapstructure:"timezone"`
}

// Status 可供登录响应、用户信息和受限接口使用，原因编码由页面转换成中文提示。
type Status struct {
	PasswordChangeRequired bool       `json:"passwordChangeRequired"`
	PasswordChangeReason   string     `json:"passwordChangeReason"`
	PasswordExpiresAt      *time.Time `json:"passwordExpiresAt"`
}

var (
	configMu sync.RWMutex
	// 未加载配置时保守地启用季度限制；服务启动仍必须成功读取配置才能运行。
	currentConfig   = Config{Enabled: true, Mode: CalendarQuarter, Timezone: "Asia/Shanghai"}
	currentLocation = time.FixedZone("Asia/Shanghai", 8*60*60)
)

func Current() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	return currentConfig
}

// Configure 校验全部规则，不能将错误配置静默解释为关闭安全限制。
func Configure(config Config) error {
	if config.Mode != CalendarQuarter {
		return fmt.Errorf("密码策略 mode 必须为 %s", CalendarQuarter)
	}
	if config.Timezone != "Asia/Shanghai" {
		return fmt.Errorf("季度密码策略 timezone 必须为 Asia/Shanghai")
	}
	location, err := time.LoadLocation(config.Timezone)
	if err != nil {
		return fmt.Errorf("加载密码策略时区失败: %w", err)
	}
	configMu.Lock()
	currentConfig, currentLocation = config, location
	configMu.Unlock()
	return nil
}

// LoadFile 从当前服务的配置文件读取季度规则，使用独立配置实例。
// 不读取环境变量，也不修改服务已有的数据库、短信等配置。
func LoadFile(path string) error {
	fileConfig := viper.New()
	fileConfig.SetConfigFile(path)
	if err := fileConfig.ReadInConfig(); err != nil {
		return fmt.Errorf("读取服务密码策略配置失败: %w", err)
	}
	for _, key := range []string{"enabled", "mode", "timezone"} {
		if !fileConfig.IsSet("security.password." + key) {
			return fmt.Errorf("服务配置缺少 security.password.%s", key)
		}
	}
	var config Config
	if err := fileConfig.UnmarshalKey("security.password", &config); err != nil {
		return fmt.Errorf("解析服务密码策略失败: %w", err)
	}
	return Configure(config)
}

func quarterStart(value time.Time, location *time.Location) time.Time {
	local := value.In(location)
	month := time.Month((int(local.Month())-1)/3*3 + 1)
	return time.Date(local.Year(), month, 1, 0, 0, 0, 0, location)
}

// ReasonMessage 将内部原因编码转换成前端可显示的中文说明。
func ReasonMessage(reason string) string {
	switch reason {
	case ReasonPasswordUnset:
		return "账号尚无密码修改记录，请先修改密码"
	case ReasonAdminReset:
		return "管理员已重置密码，请先修改为本人设置的新密码"
	case ReasonFutureTime:
		return "密码修改时间异常，请重新修改密码"
	case ReasonQuarterExpired:
		return "已进入新的季度，请先修改账号密码"
	default:
		return "请先修改账号密码"
	}
}

// Evaluate 按自然季度判断，每个季度首次使用前必须改密，不采用滚动 90 天。
func Evaluate(changedAt *time.Time, force bool, now time.Time) Status {
	configMu.RLock()
	config, location := currentConfig, currentLocation
	configMu.RUnlock()
	status := Status{}
	if config.Enabled && changedAt != nil && !changedAt.IsZero() {
		expiresAt := quarterStart(*changedAt, location).AddDate(0, 3, 0)
		status.PasswordExpiresAt = &expiresAt
	}
	if force {
		status.PasswordChangeRequired = true
		status.PasswordChangeReason = ReasonAdminReset
		return status
	}
	if !config.Enabled {
		return status
	}
	status.PasswordChangeRequired = true
	if changedAt == nil || changedAt.IsZero() {
		status.PasswordChangeReason = ReasonPasswordUnset
	} else if changedAt.After(now) {
		status.PasswordChangeReason = ReasonFutureTime
	} else if changedAt.Before(quarterStart(now, location)) {
		status.PasswordChangeReason = ReasonQuarterExpired
	} else {
		status.PasswordChangeRequired = false
	}
	return status
}
