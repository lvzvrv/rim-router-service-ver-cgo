package utils

import (
	"os"
	"testing"
	"time"

	"rim-router-service-ver-cgo/internal/config"
	"rim-router-service-ver-cgo/internal/models"

	"github.com/golang-jwt/jwt"
)

func TestGenerateAndValidateAccessToken(t *testing.T) {
	// 🔧 Подготовка: задаём тестовый секрет через переменную окружения
	os.Setenv("JWT_SECRET", "testsecretkey")

	user := &models.User{
		ID:       42,
		Username: "tester",
		Role:     1,
	}

	tokenString, err := GenerateAccessToken(user)
	if err != nil {
		t.Fatalf("Ошибка при генерации токена: %v", err)
	}

	// ✅ Проверяем, что токен не пустой
	if tokenString == "" {
		t.Fatal("Ожидался непустой JWT токен, получили пустую строку")
	}

	// ✅ Проверяем валидацию
	claims, err := ValidateToken(tokenString)
	if err != nil {
		t.Fatalf("Ошибка при валидации токена: %v", err)
	}

	// 🔍 Проверяем совпадение данных
	if claims.UserID != user.ID {
		t.Errorf("UserID mismatch: ожидалось %d, получили %d", user.ID, claims.UserID)
	}
	if claims.Username != user.Username {
		t.Errorf("Username mismatch: ожидалось %s, получили %s", user.Username, claims.Username)
	}
	if claims.Role != user.Role {
		t.Errorf("Role mismatch: ожидалось %d, получили %d", user.Role, claims.Role)
	}

	// 🕓 Проверяем время жизни
	jwtCfg := config.GetJWTConfig()
	expectedExp := time.Now().Add(jwtCfg.AccessExpiration)
	actualExp := time.Unix(claims.ExpiresAt, 0)

	if actualExp.Sub(expectedExp) > time.Minute {
		t.Errorf("Время жизни токена не совпадает с ожидаемым: %v vs %v", actualExp, expectedExp)
	}
}

func TestValidateToken_InvalidSignature(t *testing.T) {
	os.Setenv("JWT_SECRET", "real-secret")

	claims := &Claims{
		UserID:   1,
		Username: "baduser",
		Role:     0,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(5 * time.Minute).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("wrong-secret"))

	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Error("Ожидалась ошибка при неправильной подписи токена, но её не возникло")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	os.Setenv("JWT_SECRET", "exp-secret")

	expiredClaims := &Claims{
		UserID:   5,
		Username: "expireduser",
		Role:     0,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(-time.Minute).Unix(),
			IssuedAt:  time.Now().Add(-2 * time.Minute).Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	tokenString, _ := token.SignedString([]byte("exp-secret"))

	_, err := ValidateToken(tokenString)
	if err == nil {
		t.Error("Ожидалась ошибка при истёкшем токене, но валидация прошла")
	}
}

func TestGenerateSecureToken(t *testing.T) {
	tok1, err1 := GenerateSecureToken()
	tok2, err2 := GenerateSecureToken()

	if err1 != nil || err2 != nil {
		t.Fatalf("Ошибка при генерации secure токена: %v, %v", err1, err2)
	}

	if len(tok1) != 64 {
		t.Errorf("Ожидалась длина 64, получили %d", len(tok1))
	}
	if tok1 == tok2 {
		t.Error("Ожидалось, что два secure токена будут различны, но они совпали")
	}
}
