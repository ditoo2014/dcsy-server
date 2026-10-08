package service

import (
	"deal/internal/deal/service/battery"

	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/gin-gonic/gin"
)

func InitBatteryTtdc() {
	battery.Init()
}

func DealBatteryTtdc(ctx *gin.Context, msgOrg *primitive.MessageExt) error {
	return battery.Deal(ctx, msgOrg)
}

func IsBatteryErrorRetryable(err error) bool { return battery.IsRetryable(err) }
