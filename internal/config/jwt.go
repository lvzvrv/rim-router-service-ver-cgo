package config

import (
	"os"
	"time"
)

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
}

func GetJWTConfig() JWTConfig {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "your-default-super-secret-key-change-in-production" // Заменить в продакшене!
	}

	return JWTConfig{
		Secret:     secret,
		Expiration: 24 * time.Hour, // 24 часа
	}
}
