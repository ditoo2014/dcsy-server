package battery

import (
	"deal/internal/batterylog"

	"github.com/spf13/viper"
)

func debugLogf(format string, args ...interface{}) {
	if viper.GetBool("battery.debugLog") {
		batterylog.Printf(format, args...)
	}
}
