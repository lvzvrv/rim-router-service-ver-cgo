package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"rim-router-service-ver-cgo/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
)

// AdminHandler — обработчик админских запросов
type AdminHandler struct {
	UserRepo *models.UserRepository
	logger   zerolog.Logger
}

// NewAdminHandler создаёт отдельный логгер для api.log
func NewAdminHandler(userRepo *models.UserRepository) *AdminHandler {
	// создаём директорию логов, если её нет
	_ = os.MkdirAll("logs", 0o755)

	logFile := filepath.Join("logs", "api.log")
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		panic("cannot open api.log: " + err.Error())
	}

	logger := zerolog.New(file).With().
		Timestamp().
		Str("module", "admin").
		Logger()

	return &AdminHandler{UserRepo: userRepo, logger: logger}
}

// GET /api/v1/admin/users — список всех пользователей
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	h.logger.Info().
		Str("endpoint", "/api/v1/admin/users").
		Msg("Admin requested user list")

	users, err := h.UserRepo.GetAllUsers()
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to fetch user list")
		sendJSON(w, http.StatusInternalServerError, "Database error", nil)
		return
	}

	h.logger.Info().
		Int("count", len(users)).
		Msg("Fetched user list successfully")

	sendJSON(w, http.StatusOK, "OK", users)
}

// POST /api/v1/admin/users/{id}/role — изменить роль пользователя
func (h *AdminHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		h.logger.Warn().Str("user_id", idStr).Msg("Invalid user ID")
		sendJSON(w, http.StatusBadRequest, "Invalid user ID", nil)
		return
	}

	var body struct {
		Role int `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		h.logger.Warn().Int("user_id", id).Msg("Invalid JSON payload")
		sendJSON(w, http.StatusBadRequest, "Invalid JSON", nil)
		return
	}

	if body.Role < 0 || body.Role > 2 {
		h.logger.Warn().Int("role", body.Role).Msg("Invalid role value")
		sendJSON(w, http.StatusBadRequest, "Invalid role value", nil)
		return
	}

	if err := h.UserRepo.UpdateUserRole(id, body.Role); err != nil {
		h.logger.Error().Err(err).Int("user_id", id).Msg("Failed to update user role")
		sendJSON(w, http.StatusInternalServerError, "Failed to update role", nil)
		return
	}

	h.logger.Info().
		Int("user_id", id).
		Int("new_role", body.Role).
		Time("ts", time.Now()).
		Msg("User role updated successfully")

	sendJSON(w, http.StatusOK, "Role updated successfully", nil)
}
