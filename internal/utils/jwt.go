package utils

import (
	"time"

	"your-app/internal/config"
	"your-app/internal/models"

	"github.com/golang-jwt/jwt"
)

type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.StandardClaims
}

func GenerateToken(user *models.User) (string, error) {
	jwtConfig := config.GetJWTConfig()

	expirationTime := time.Now().Add(jwtConfig.Expiration)
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Role:     user.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtConfig.Secret))
}

func ValidateToken(tokenString string) (*Claims, error) {
	jwtConfig := config.GetJWTConfig()

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
