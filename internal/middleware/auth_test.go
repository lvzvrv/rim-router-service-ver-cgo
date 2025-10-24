package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"rim-router-service-ver-cgo/internal/utils"

	"github.com/stretchr/testify/assert"
)

// =============================
//   Вспомогательные функции
// =============================

func makeHandlerCalledFlag() (http.Handler, *bool) {
	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	return handler, &called
}

// =============================
//   Тест AuthMiddleware
// =============================

func TestAuthMiddleware_NoHeader(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler, _ := makeHandlerCalledFlag()
	middleware := AuthMiddleware(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authorization header required")
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "TokenOnlyWithoutBearer")
	w := httptest.NewRecorder()

	handler, _ := makeHandlerCalledFlag()
	middleware := AuthMiddleware(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid authorization format")
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	oldValidate := utils.ValidateTokenFunc
	defer func() { utils.ValidateTokenFunc = oldValidate }()

	utils.ValidateTokenFunc = func(token string) (*utils.Claims, error) {
		return nil, assert.AnError
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	w := httptest.NewRecorder()

	handler, called := makeHandlerCalledFlag()
	middleware := AuthMiddleware(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid token")
	assert.False(t, *called)
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	oldValidate := utils.ValidateTokenFunc
	defer func() { utils.ValidateTokenFunc = oldValidate }()

	utils.ValidateTokenFunc = func(token string) (*utils.Claims, error) {
		return &utils.Claims{UserID: 1, Username: "admin", Role: 1}, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid_token")
	w := httptest.NewRecorder()

	handler, called := makeHandlerCalledFlag()
	middleware := AuthMiddleware(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *called)
}

// =============================
//   Тест RoleMiddleware
// =============================

func TestRoleMiddleware_NoUserInContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler, _ := makeHandlerCalledFlag()
	middleware := RoleMiddleware(1)(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Contains(t, w.Body.String(), "Authentication required")
}

func TestRoleMiddleware_InsufficientRole(t *testing.T) {
	claims := &utils.Claims{UserID: 1, Username: "user", Role: 0}
	ctx := context.WithValue(context.Background(), UserContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler, called := makeHandlerCalledFlag()
	middleware := RoleMiddleware(1)(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "Insufficient permissions")
	assert.False(t, *called)
}

func TestRoleMiddleware_SufficientRole(t *testing.T) {
	claims := &utils.Claims{UserID: 1, Username: "admin", Role: 2}
	ctx := context.WithValue(context.Background(), UserContextKey, claims)
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	w := httptest.NewRecorder()

	handler, called := makeHandlerCalledFlag()
	middleware := RoleMiddleware(1)(handler)
	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, *called)
}

// =============================
//   Тест GetUserFromContext
// =============================

func TestGetUserFromContext(t *testing.T) {
	claims := &utils.Claims{UserID: 42, Username: "tester", Role: 1}

	t.Run("valid context", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), UserContextKey, claims)
		got := GetUserFromContext(ctx)
		assert.Equal(t, claims, got)
	})

	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		got := GetUserFromContext(ctx)
		assert.Nil(t, got)
	})
}
