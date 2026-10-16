package authsession

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"dicloud/pkg/cache"

	"github.com/gin-gonic/gin"
)

const (
	// IdleTimeout 是 Redis 登录会话的空闲超时时间。
	// 用户登录后会写入 Redis；每次有效请求都会重新续期，超过该时间未请求则登录态失效。
	IdleTimeout = 30 * time.Minute

	// keyPrefix 使用 token 哈希作为 Redis key，避免直接把原始 token 暴露在 key 中。
	keyPrefix = "auth:session:%s"
)

func sessionKey(token string) string {
	sum := sha256.Sum256([]byte(token))
	return fmt.Sprintf(keyPrefix, hex.EncodeToString(sum[:]))
}

// Activate 在用户登录成功后创建 Redis 登录会话，并设置空闲过期时间。
func Activate(ctx *gin.Context, username string, token string) error {
	if username == "" || token == "" {
		return fmt.Errorf("empty username or token")
	}
	key := sessionKey(token)
	return cache.SetWithExpiration(ctx, key, username, IdleTimeout)
}

// Touch 校验 Redis 登录会话是否仍然存在；存在则刷新空闲过期时间。
// 会话不存在或用户不匹配时返回 false；Redis 故障通过 error 单独返回，不能误判为过期。
func Touch(ctx *gin.Context, username string, token string) (bool, error) {
	if username == "" || token == "" {
		return false, nil
	}
	key := sessionKey(token)
	return cache.MatchAndExpire(ctx, key, username, IdleTimeout)
}

// Clear 在退出登录时删除 Redis 登录会话。
func Clear(ctx *gin.Context, token string) error {
	if token == "" {
		return nil
	}
	return cache.Delete(ctx, sessionKey(token))
}
