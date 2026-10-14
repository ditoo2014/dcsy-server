package common

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/dgrijalva/jwt-go"
)

var jwtKey = []byte("a_secret_crect")

type Claims struct {
	UserName string
	jwt.StandardClaims
}

func ReleaseToken(UserName string) (string, time.Duration, error) {

	timeDuration := 1 * 24 * time.Hour //1天有效时间
	//timeDuration := 30 * time.Minute //30分钟有效时间

	// 同一秒重新登录也生成不同令牌，避免改密清理会话后复用旧 token。
	var tokenID [16]byte
	if _, err := rand.Read(tokenID[:]); err != nil {
		return "", timeDuration, fmt.Errorf("生成登录令牌标识失败: %w", err)
	}
	now := time.Now()
	claims := &Claims{
		UserName: UserName,
		StandardClaims: jwt.StandardClaims{
			// 调整为30分钟的时间
			ExpiresAt: now.Add(timeDuration).Unix(),
			IssuedAt:  now.Unix(),
			Id:        hex.EncodeToString(tokenID[:]),
			Issuer:    "ditoo.com.cn",
			Subject:   "user token",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)

	if err != nil {
		return "", timeDuration, err
	}

	return tokenString, timeDuration, nil
}

func ParseToken(tokenString string) (*jwt.Token, *Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (i interface{}, err error) {
		// 只接受当前签发算法，拒绝换算法验证；现有 HS256 令牌仍可使用。
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("登录令牌签名算法不受支持")
		}
		return jwtKey, nil
	})

	return token, claims, err
}
