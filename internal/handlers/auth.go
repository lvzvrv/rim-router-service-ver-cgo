// internal/handlers/auth.go
package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"your-app/internal/models"
	"your-app/internal/utils"

	"your-app/internal/config"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserRepo  *models.UserRepository
	TokenRepo *models.TokenRepository
}

func NewAuthHandler(userRepo *models.UserRepository, tokenRepo *models.TokenRepository) *AuthHandler {
	return &AuthHandler{UserRepo: userRepo, TokenRepo: tokenRepo}
}

type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	Role        int    `json:"role"`
}

// ---------- Register ----------
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	if len(req.Username) < 3 || len(req.Username) > 20 {
		sendJSON(w, http.StatusBadRequest, "Username must be 3-20 characters", nil)
		return
	}
	if len(req.Password) < 6 {
		sendJSON(w, http.StatusBadRequest, "Password must be at least 6 characters", nil)
		return
	}
	if matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", req.Username); !matched {
		sendJSON(w, http.StatusBadRequest, "Username can only contain letters, numbers and underscores", nil)
		return
	}

	exists, err := h.UserRepo.UserExists(req.Username)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Database error", nil)
		return
	}
	if exists {
		sendJSON(w, http.StatusConflict, "User already exists", nil)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Password hashing failed", nil)
		return
	}

	if err := h.UserRepo.CreateUser(req.Username, string(hashedPassword), 0); err != nil {
		sendJSON(w, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}

	sendJSON(w, http.StatusCreated, "User registered successfully", nil)
}

// ---------- Login ----------
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	user, err := h.UserRepo.GetUserByUsername(req.Username)
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		sendJSON(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	access, err := utils.GenerateAccessToken(user)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Token generation failed", nil)
		return
	}

	cfg := config.GetJWTConfig()

	refresh, err := utils.GenerateSecureToken()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Refresh token generation failed", nil)
		return
	}

	// Удаляем старые токены перед выдачей нового
	_ = h.TokenRepo.DeleteAllForUser(user.ID)

	if err := h.TokenRepo.SaveRefreshToken(user.ID, refresh, time.Now().Add(cfg.RefreshExpiration)); err != nil {
		sendJSON(w, http.StatusInternalServerError, "Failed to persist refresh token", nil)
		return
	}

	// HttpOnly cookie (локально, без Secure)
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		// Secure: false // по умолчанию false; оставляем локально
		MaxAge: int(cfg.RefreshExpiration / time.Second),
	})

	resp := AuthResponse{AccessToken: access, Role: user.Role}
	sendJSON(w, http.StatusOK, "Login successful", resp)
}

// ---------- Refresh ----------
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		sendJSON(w, http.StatusUnauthorized, "Missing refresh token", nil)
		return
	}

	token := cookie.Value
	userID, expiresAt, err := h.TokenRepo.GetRefreshToken(token)
	if err != nil || time.Now().After(expiresAt) {
		_ = h.TokenRepo.DeleteRefreshToken(token) // на всякий случай чистим
		sendJSON(w, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
		return
	}

	user, err := h.UserRepo.GetUserByID(userID)
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, "User not found", nil)
		return
	}

	// Ротация refresh-токена: удалить старый, создать новый
	_ = h.TokenRepo.DeleteRefreshToken(token)

	cfg := config.GetJWTConfig()

	newRefresh, err := utils.GenerateSecureToken()
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Refresh token generation failed", nil)
		return
	}
	if err := h.TokenRepo.SaveRefreshToken(user.ID, newRefresh, time.Now().Add(cfg.RefreshExpiration)); err != nil {
		sendJSON(w, http.StatusInternalServerError, "Failed to persist refresh token", nil)
		return
	}

	// Устанавливаем новый cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(cfg.RefreshExpiration / time.Second),
	})

	// Новый access
	access, err := utils.GenerateAccessToken(user)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Token generation failed", nil)
		return
	}

	sendJSON(w, http.StatusOK, "Token refreshed", map[string]string{
		"access_token": access,
	})
}

// ---------- Logout ----------
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		// Пытаемся найти user_id по токену
		userID, _, _ := h.TokenRepo.GetRefreshToken(cookie.Value)
		if userID != 0 {
			// Удаляем все токены пользователя (включая этот)
			_ = h.TokenRepo.DeleteAllForUser(userID)
		} else {
			// Если не нашли — просто чистим по значению
			_ = h.TokenRepo.DeleteRefreshToken(cookie.Value)
		}
	}

	// Удаляем cookie у клиента
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	sendJSON(w, http.StatusOK, "Logged out", nil)
}
