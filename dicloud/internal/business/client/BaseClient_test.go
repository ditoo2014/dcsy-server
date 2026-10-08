package client

import (
	"context"
	"dicloud/internal/common/grpc/tolink_grpc"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 不启动真实服务，确认缺少连接配置时能安全返回错误。
func TestTolinkMissingConnection(t *testing.T) {
	client := tolinkClientForConnection(nil)
	resp, err := client.FindGeneralTrigger(context.Background(), &tolink_grpc.GeneralTriggerConfig{})
	if resp != nil || status.Code(err) != codes.Unavailable {
		t.Fatalf("期望服务不可用，实际响应=%v 错误=%v", resp, err)
	}
}
