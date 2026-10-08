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
// 如果 Redis 中没有该会话，或会话所属用户与 token 中用户不一致，则认为登录已超时。
func Touch(ctx *gin.Context, username string, token string) bool {
	if username == "" || token == "" {
		return false
	}
	key := sessionKey(token)
	storedUsername, err := cache.Get(ctx, key)
	if err != nil || storedUsername != username {
		return false
	}
	cache.ExpireKey(ctx, key, IdleTimeout)
	return true
}

// Clear 在退出登录时删除 Redis 登录会话。
func Clear(ctx *gin.Context, token string) error {
	if token == "" {
		return nil
	}
	return cache.Delete(ctx, sessionKey(token))
}
