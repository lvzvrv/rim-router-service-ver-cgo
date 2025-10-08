package handlers

import (
	"encoding/json"
	"net/http"
	"regexp"

	"your-app/internal/models"
	"your-app/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserRepo *models.UserRepository
}

func NewAuthHandler(userRepo *models.UserRepository) *AuthHandler {
	return &AuthHandler{UserRepo: userRepo}
}

// RegisterRequest - запрос на регистрацию
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest - запрос на вход
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// AuthResponse - ответ с токеном
type AuthResponse struct {
	Token string `json:"token"`
	Role  int    `json:"role"`
}

// Register - регистрация нового пользователя
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	// Валидация
	if len(req.Username) < 3 || len(req.Username) > 20 {
		sendJSON(w, http.StatusBadRequest, "Username must be 3-20 characters", nil)
		return
	}

	if len(req.Password) < 6 {
		sendJSON(w, http.StatusBadRequest, "Password must be at least 6 characters", nil)
		return
	}

	// Проверка формата username (только буквы, цифры, подчеркивание)
	if matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", req.Username); !matched {
		sendJSON(w, http.StatusBadRequest, "Username can only contain letters, numbers and underscores", nil)
		return
	}

	// Проверяем, не существует ли пользователь
	exists, err := h.UserRepo.UserExists(req.Username)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Database error", nil)
		return
	}
	if exists {
		sendJSON(w, http.StatusConflict, "User already exists", nil)
		return
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Password hashing failed", nil)
		return
	}

	// Создаем пользователя (по умолчанию роль - обычный пользователь)
	err = h.UserRepo.CreateUser(req.Username, string(hashedPassword), 0)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Failed to create user", nil)
		return
	}

	sendJSON(w, http.StatusCreated, "User registered successfully", nil)
}

// Login - аутентификация пользователя
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, http.StatusBadRequest, "Invalid request", nil)
		return
	}

	// Ищем пользователя
	user, err := h.UserRepo.GetUserByUsername(req.Username)
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Проверяем пароль
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password))
	if err != nil {
		sendJSON(w, http.StatusUnauthorized, "Invalid credentials", nil)
		return
	}

	// Генерируем JWT токен
	token, err := utils.GenerateToken(user)
	if err != nil {
		sendJSON(w, http.StatusInternalServerError, "Token generation failed", nil)
		return
	}

	response := AuthResponse{
		Token: token,
		Role:  user.Role,
	}

	sendJSON(w, http.StatusOK, "Login successful", response)
}
