package common

import (
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

	claims := &Claims{
		UserName: UserName,
		StandardClaims: jwt.StandardClaims{
			// 调整为30分钟的时间
			ExpiresAt: time.Now().Add(timeDuration).Unix(),
			IssuedAt:  time.Now().Unix(),
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
		return jwtKey, nil
	})

	return token, claims, err
}
