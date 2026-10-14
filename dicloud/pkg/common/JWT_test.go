package common

import (
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
)

func TestRepeatedLoginTokensAreDistinct(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		token, _, err := ReleaseToken("same-user")
		if err != nil {
			t.Fatal(err)
		}
		if seen[token] {
			t.Fatal("同一用户连续登录不能复用已失效的令牌")
		}
		seen[token] = true
		parsed, claims, err := ParseToken(token)
		if err != nil || !parsed.Valid || claims.Id == "" || claims.UserName != "same-user" {
			t.Fatalf("新令牌应保留现有身份信息和随机标识: %+v, %v", claims, err)
		}
	}
}

func TestSigningAlgorithmAndLegacyTokenCompatibility(t *testing.T) {
	claims := &Claims{UserName: "old-user", StandardClaims: jwt.StandardClaims{ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	legacy, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	if parsed, _, err := ParseToken(legacy); err != nil || !parsed.Valid {
		t.Fatalf("升级前未带 jti 的 HS256 登录令牌仍应通过签名验证: %v", err)
	}
	otherAlgorithm, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString(jwtKey)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseToken(otherAlgorithm); err == nil {
		t.Fatal("不能接受其他签名算法的令牌")
	}
}
