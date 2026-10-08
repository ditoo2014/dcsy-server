package controller

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"dicloud/internal/auth/dto"
	"dicloud/models/model"
	"dicloud/pkg/cache"
	"dicloud/util/response"
)

const (
	LOGIN_FAILURE_PREFIX = "loginfail:%s"
	LOGIN_LOCK_PREFIX    = "loginlock:%s"

	defaultLoginMaxFailures          = 5
	defaultLoginLockMinutes          = 15
	defaultLoginFailureWindowMinutes = 15
	minCompliantLoginLockMinutes     = 11
)

type loginSecurityConfig struct {
	maxFailures  int
	lockDuration time.Duration
	failWindow   time.Duration
}

func getLoginSecurityConfig() loginSecurityConfig {
	return normalizeLoginSecurityConfig(
		viper.GetInt("security.login.maxFailures"),
		viper.GetInt("security.login.lockMinutes"),
		viper.GetInt("security.login.failureWindowMinutes"),
	)
}

func normalizeLoginSecurityConfig(maxFailures, lockMinutes, failureWindowMinutes int) loginSecurityConfig {
	if maxFailures <= 0 || maxFailures > defaultLoginMaxFailures {
		maxFailures = defaultLoginMaxFailures
	}
	if lockMinutes < minCompliantLoginLockMinutes {
		lockMinutes = defaultLoginLockMinutes
	}
	if failureWindowMinutes <= 0 {
		failureWindowMinutes = lockMinutes
	}

	return loginSecurityConfig{
		maxFailures:  maxFailures,
		lockDuration: time.Duration(lockMinutes) * time.Minute,
		failWindow:   time.Duration(failureWindowMinutes) * time.Minute,
	}
}

func loginIdentifierFromDTO(req dto.LoginDTO) string {
	if identifier := strings.TrimSpace(req.Username); len(identifier) > 0 {
		return identifier
	}
	return strings.TrimSpace(req.Phone)
}

func loginIdentifierFromUser(dbUser model.User, fallback string) string {
	if identifier := strings.TrimSpace(dbUser.Username); len(identifier) > 0 {
		return identifier
	}
	return strings.TrimSpace(fallback)
}

func ensureLoginAllowed(ctx *gin.Context, identifier string) bool {
	identifier = strings.TrimSpace(identifier)
	if len(identifier) == 0 {
		return true
	}

	key := fmt.Sprintf(LOGIN_LOCK_PREFIX, identifier)
	if result, err := cache.Get(ctx, key); err == nil && len(result) > 0 {
		ttl, _ := cache.TTL(ctx, key)
		response.Fail(ctx, buildLoginLockMessage(ttl), gin.H{
			"remainSeconds": remainingSeconds(ttl),
		})
		return false
	}
	return true
}

func failLogin(ctx *gin.Context, identifier string) {
	failLoginWithMessage(ctx, identifier, "用户名或密码错误")
}

func failLoginWithMessage(ctx *gin.Context, identifier string, message string) {
	if recordLoginFailure(ctx, identifier) {
		cfg := getLoginSecurityConfig()
		response.Fail(ctx, buildLoginLockMessage(cfg.lockDuration), gin.H{
			"remainSeconds": int(cfg.lockDuration.Seconds()),
		})
		return
	}
	response.Fail(ctx, message, nil)
}

func recordLoginFailure(ctx *gin.Context, identifier string) bool {
	identifier = strings.TrimSpace(identifier)
	if len(identifier) == 0 {
		return false
	}

	cfg := getLoginSecurityConfig()
	failKey := fmt.Sprintf(LOGIN_FAILURE_PREFIX, identifier)
	lockKey := fmt.Sprintf(LOGIN_LOCK_PREFIX, identifier)

	count, err := cache.Incr(ctx, failKey)
	if err != nil {
		log.Println("[AUTH] Login failure count incr fail:", err)
		return false
	}
	if count == 1 {
		cache.ExpireKey(ctx, failKey, cfg.failWindow)
	} else if ttl, err := cache.TTL(ctx, failKey); err == nil && ttl <= 0 {
		cache.ExpireKey(ctx, failKey, cfg.failWindow)
	}

	if int(count) < cfg.maxFailures {
		return false
	}

	if err := cache.Set(ctx, lockKey, "1"); err != nil {
		log.Println("[AUTH] Login lock set fail:", err)
		return false
	}
	cache.ExpireKey(ctx, lockKey, cfg.lockDuration)
	cache.Del(ctx, failKey)
	return true
}

func clearLoginFailure(ctx *gin.Context, identifier string) {
	identifier = strings.TrimSpace(identifier)
	if len(identifier) == 0 {
		return
	}
	cache.Del(ctx, fmt.Sprintf(LOGIN_FAILURE_PREFIX, identifier))
	cache.Del(ctx, fmt.Sprintf(LOGIN_LOCK_PREFIX, identifier))
}

func buildLoginLockMessage(ttl time.Duration) string {
	if ttl <= 0 {
		return "账号已锁定，请稍后再试"
	}
	return fmt.Sprintf("账号已锁定，请%d分钟后再试", remainingMinutes(ttl))
}

func remainingMinutes(ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}
	return int((ttl + time.Minute - time.Nanosecond) / time.Minute)
}

func remainingSeconds(ttl time.Duration) int {
	if ttl <= 0 {
		return 0
	}
	return int((ttl + time.Second - time.Nanosecond) / time.Second)
}
