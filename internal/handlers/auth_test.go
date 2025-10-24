package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"rim-router-service-ver-cgo/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// вспомогательная функция
func setupAuthHandler(t *testing.T) (*AuthHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка создания mock DB: %v", err)
	}
	userRepo := models.NewUserRepository(db)
	tokenRepo := models.NewTokenRepository(db)
	handler := NewAuthHandler(userRepo, tokenRepo)
	return handler, mock, func() { db.Close() }
}

// ========== TEST: Register ==========

func TestRegister_Success(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("newuser").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	mock.ExpectExec("INSERT INTO users").
		WithArgs("newuser", sqlmock.AnyArg(), 0).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"username":"newuser","password":"strongpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "User registered successfully", resp.Message)
}

func TestRegister_UserExists(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("john").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	body := `{"username":"john","password":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestRegister_InvalidUsername(t *testing.T) {
	h, _, cleanup := setupAuthHandler(t)
	defer cleanup()

	body := `{"username":"!bad!","password":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_ShortPassword(t *testing.T) {
	h, _, cleanup := setupAuthHandler(t)
	defer cleanup()

	body := `{"username":"okuser","password":"123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRegister_DBError(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT EXISTS").
		WithArgs("alex").
		WillReturnError(sql.ErrConnDone)

	body := `{"username":"alex","password":"1234567"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/register", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Register(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ========== TEST: Login ==========

func TestLogin_Success(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)

	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "created_at"}).
		AddRow(1, "tester", string(hashed), 0, time.Now())

	mock.ExpectQuery("SELECT id, username").
		WithArgs("tester").
		WillReturnRows(rows)

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WithArgs(int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	body := `{"username":"tester","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestLogin_UserNotFound(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, username").
		WithArgs("ghost").
		WillReturnError(sql.ErrNoRows)

	body := `{"username":"ghost","password":"123456"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestLogin_InvalidPassword(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	hashed, _ := bcrypt.GenerateFromPassword([]byte("realpass"), bcrypt.DefaultCost)

	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "created_at"}).
		AddRow(1, "john", string(hashed), 0, time.Now())

	mock.ExpectQuery("SELECT id, username").
		WithArgs("john").
		WillReturnRows(rows)

	body := `{"username":"john","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/login", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	h.Login(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========== TEST: Refresh ==========

func TestRefresh_Success(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	now := time.Now().Add(10 * time.Minute)

	mock.ExpectQuery("SELECT user_id, expires_at").
		WithArgs("refresh123").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at"}).AddRow(1, now))

	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "created_at"}).
		AddRow(1, "user", "hash", 0, time.Now())

	mock.ExpectQuery("SELECT id, username").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	mock.ExpectExec("DELETE FROM refresh_tokens").
		WithArgs("refresh123").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectExec("INSERT INTO refresh_tokens").
		WithArgs(int64(1), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh123"})
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRefresh_InvalidToken(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT user_id, expires_at").
		WithArgs("expired").
		WillReturnError(sql.ErrNoRows)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired"})
	w := httptest.NewRecorder()

	h.Refresh(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ========== TEST: Logout ==========

func TestLogout_Success(t *testing.T) {
	h, mock, cleanup := setupAuthHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT user_id, expires_at").
		WithArgs("ref123").
		WillReturnRows(sqlmock.NewRows([]string{"user_id", "expires_at"}).AddRow(5, time.Now().Add(1*time.Hour)))

	mock.ExpectExec("DELETE FROM refresh_tokens WHERE user_id = ?").
		WithArgs(int64(5)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/logout", nil)
	req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "ref123"})
	w := httptest.NewRecorder()

	h.Logout(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
