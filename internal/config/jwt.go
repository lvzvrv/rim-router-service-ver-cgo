package config

import (
	"os"
	"time"
)

type JWTConfig struct {
	Secret            string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

func GetJWTConfig() JWTConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-default-super-secret-key-change-in-production" // Заменить в продакшене!
	}

	return JWTConfig{
		Secret:            secret,
		AccessExpiration:  15 * time.Minute,   // access token: 15 минут
		RefreshExpiration: 7 * 24 * time.Hour, // refresh token: 7 дней
	}
}
