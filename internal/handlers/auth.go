package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"time"

	"rim-router-service-ver-cgo/internal/config"
	"rim-router-service-ver-cgo/internal/models"
	"rim-router-service-ver-cgo/internal/utils"

	"github.com/rs/zerolog/log"
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

	logger := log.With().Str("module", "auth").Str("user", req.Username).Logger()

	exists, err := h.UserRepo.UserExists(req.Username)
	if err != nil {
		(&logger).Error().Err(err).Msg("Database error during registration")
		sendJSON(w, http.StatusInternalServerError, "Database error", nil)
		return
	}
	if exists {
		(&logger).Warn().Msg("Attempt to register existing user")
		sendJSON(w, http.StatusConflict, "User already exists", nil)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		(&logger).Error().Msg("Password hashing failed")
		sendJSON(w, http.StatusInternalServerError, "Password hashing failed", nil)
		return
	}

	if err := h.UserRepo.CreateUser(req.Username, string(hashedPassword), 0); err != nil {
		(&logger).Error().Msg("Failed to create user in database")
		sendJSON(w, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}

	(&logger).Info().Msg("User registered successfully")
	sendJSON(w, http.StatusCreated, "User registered successfully", nil)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	logger := log.With().Str("module", "auth").Str("user", req.Username).Logger()

	user, err := h.UserRepo.GetUserByUsername(req.Username)
	if err != nil {
		(&logger).Warn().Msg("User not found during login")
		sendJSON(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		(&logger).Warn().Msg("Invalid password attempt")
		sendJSON(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	access, err := utils.GenerateAccessToken(user)
	if err != nil {
		(&logger).Error().Msg("Access token generation failed")
		sendJSON(w, http.StatusInternalServerError, "Token generation failed", nil)
		return
	}

	cfg := config.GetJWTConfig()
	refresh, err := utils.GenerateSecureToken()
	if err != nil {
		(&logger).Error().Msg("Refresh token generation failed")
		sendJSON(w, http.StatusInternalServerError, "Refresh token generation failed", nil)
		return
	}

	_ = h.TokenRepo.DeleteAllForUser(user.ID)
	if err := h.TokenRepo.SaveRefreshToken(user.ID, refresh, time.Now().Add(cfg.RefreshExpiration)); err != nil {
		(&logger).Error().Msg("Failed to persist refresh token")
		sendJSON(w, http.StatusInternalServerError, "Failed to persist refresh token", nil)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    refresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(cfg.RefreshExpiration / time.Second),
	})

	(&logger).Info().Msg("User logged in successfully")

	resp := AuthResponse{AccessToken: access, Role: user.Role}
	sendJSON(w, http.StatusOK, "Login successful", resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil || cookie.Value == "" {
		sendJSON(w, http.StatusUnauthorized, "Missing refresh token", nil)
		return
	}

	token := cookie.Value
	userID, expiresAt, err := h.TokenRepo.GetRefreshToken(token)
	if err != nil || time.Now().After(expiresAt) {
		_ = h.TokenRepo.DeleteRefreshToken(token)
		logger := log.With().Str("module", "auth").Logger()
		(&logger).Warn().Msg("Invalid or expired refresh token")
		sendJSON(w, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
		return
	}

	user, err := h.UserRepo.GetUserByID(userID)
	if err != nil {
		logger := log.With().Str("module", "auth").Logger()
		(&logger).Warn().Msg("User not found for refresh")
		sendJSON(w, http.StatusUnauthorized, "User not found", nil)
		return
	}

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

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    newRefresh,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(cfg.RefreshExpiration / time.Second),
	})

	access, err := utils.GenerateAccessToken(user)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Token generation failed", nil)
		return
	}

	logger := log.With().Str("module", "auth").Str("user", user.Username).Logger()
	(&logger).Debug().Msg("Refresh token rotated")

	sendJSON(w, http.StatusOK, "Token refreshed", map[string]string{
		"access_token": access,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		userID, _, _ := h.TokenRepo.GetRefreshToken(cookie.Value)
		if userID != 0 {
			_ = h.TokenRepo.DeleteAllForUser(userID)
		} else {
			_ = h.TokenRepo.DeleteRefreshToken(cookie.Value)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	})

	logger := log.With().Str("module", "auth").Logger()
	(&logger).Info().Msg("User logged out")

	sendJSON(w, http.StatusOK, "Logged out", nil)
}
