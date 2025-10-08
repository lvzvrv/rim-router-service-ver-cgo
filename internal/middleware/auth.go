package middleware

import (
	"context"
	"net/http"
	"strings"

	"your-app/internal/utils"
)

type contextKey string

const (
	UserContextKey contextKey = "user"
)

// AuthMiddleware проверяет JWT токен
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, `{"code": 401, "message": "Authorization header required"}`, http.StatusUnauthorized)
			return
		}

		// Формат: Bearer <token>
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(w, `{"code": 401, "message": "Invalid authorization format"}`, http.StatusUnauthorized)
			return
		}

		tokenString := parts[1]
		claims, err := utils.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, `{"code": 401, "message": "Invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Добавляем claims в контекст
		ctx := context.WithValue(r.Context(), UserContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RoleMiddleware проверяет роль пользователя
func RoleMiddleware(requiredRole int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := r.Context().Value(UserContextKey).(*utils.Claims)
			if !ok {
				http.Error(w, `{"code": 401, "message": "Authentication required"}`, http.StatusUnauthorized)
				return
			}

			if claims.Role < requiredRole {
				http.Error(w, `{"code": 403, "message": "Insufficient permissions"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r.WithContext(r.Context()))
		})
	}
}

// GetUserFromContext извлекает пользователя из контекста
func GetUserFromContext(ctx context.Context) *utils.Claims {
	claims, ok := ctx.Value(UserContextKey).(*utils.Claims)
	if !ok {
		return nil
	}
	return claims
}
