package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"your-app/internal/utils"
)

// TestAuthMiddlewareLogic тестирует только логику middleware без реальной JWT валидации
func TestAuthMiddlewareLogic(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "No authorization header",
			authHeader:     "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format - no bearer",
			authHeader:     "token",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid format - multiple parts",
			authHeader:     "Bearer token extra",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Empty token",
			authHeader:     "Bearer ",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			rr := httptest.NewRecorder()

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Error("Next handler should not be called when auth fails")
			})

			middleware := AuthMiddleware(nextHandler)
			middleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestRoleMiddlewareLogic тестирует логику проверки ролей
func TestRoleMiddlewareLogic(t *testing.T) {
	tests := []struct {
		name           string
		userRole       int
		requiredRole   int
		expectedStatus int
	}{
		{
			name:           "Admin can access admin endpoint",
			userRole:       1,
			requiredRole:   1,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "User cannot access admin endpoint",
			userRole:       0,
			requiredRole:   1,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "User can access user endpoint",
			userRole:       0,
			requiredRole:   0,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/endpoint", nil)

			// Set user in context directly
			claims := &utils.Claims{
				UserID:   1,
				Username: "testuser",
				Role:     tt.userRole,
			}
			req = req.WithContext(context.WithValue(req.Context(), UserContextKey, claims))

			rr := httptest.NewRecorder()

			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			middleware := RoleMiddleware(tt.requiredRole)(nextHandler)
			middleware.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

// TestExtractToken тестирует функцию извлечения токена
func TestExtractToken(t *testing.T) {
	tests := []struct {
		header   string
		expected string
		valid    bool
	}{
		{"Bearer token123", "token123", true},
		{"Bearer", "", false},
		{"token123", "", false},
		{"", "", false},
		{"Bearer token1 token2", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}

			token := extractTokenFromHeader(req)
			if tt.valid {
				if token != tt.expected {
					t.Errorf("Expected token '%s', got '%s'", tt.expected, token)
				}
			} else {
				if token != "" {
					t.Errorf("Expected empty token, got '%s'", token)
				}
			}
		})
	}
}

// Вспомогательная функция для извлечения токена (добавьте её в auth.go если нужно)
func extractTokenFromHeader(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}
