package handlers

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

	"rim-router-service-ver-cgo/internal/models"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

// helper: создаёт mock DB и AdminHandler
func setupAdminHandler(t *testing.T) (*AdminHandler, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка создания mock DB: %v", err)
	}
	repo := models.NewUserRepository(db)
	handler := NewAdminHandler(repo)
	cleanup := func() { db.Close() }
	return handler, mock, cleanup
}

// ====== TEST: ListUsers ======
func TestListUsers_Success(t *testing.T) {
	h, mock, cleanup := setupAdminHandler(t)
	defer cleanup()

	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "username", "role", "created_at"}).
		AddRow(1, "admin", 1, now).
		AddRow(2, "user", 0, now)

	mock.ExpectQuery(regexp.QuoteMeta("SELECT id, username, role, created_at FROM users ORDER BY id ASC")).
		WillReturnRows(rows)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "OK", resp.Message)

	// Data может быть сериализован в map — проверяем наличие данных
	assert.NotNil(t, resp.Data)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestListUsers_DBError(t *testing.T) {
	h, mock, cleanup := setupAdminHandler(t)
	defer cleanup()

	mock.ExpectQuery("SELECT id, username").WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users", nil)
	w := httptest.NewRecorder()

	h.ListUsers(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 500, resp.Code)
	assert.Equal(t, "Database error", resp.Message)
}

// ====== TEST: UpdateUserRole ======

func TestUpdateUserRole_Success(t *testing.T) {
	h, mock, cleanup := setupAdminHandler(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET role = ? WHERE id = ?")).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	body := `{"role":1}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/2/role", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	// правильный способ добавить chi.RouteContext
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "2")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUserRole(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "Role updated successfully", resp.Message)
}

func TestUpdateUserRole_InvalidID(t *testing.T) {
	h, _, cleanup := setupAdminHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/abc/role", bytes.NewBufferString(`{"role":1}`))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "abc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUserRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserRole_InvalidJSON(t *testing.T) {
	h, _, cleanup := setupAdminHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/1/role", bytes.NewBufferString(`{bad-json}`))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUserRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserRole_InvalidRoleValue(t *testing.T) {
	h, _, cleanup := setupAdminHandler(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/1/role", bytes.NewBufferString(`{"role":5}`))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUserRole(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUpdateUserRole_DBError(t *testing.T) {
	h, mock, cleanup := setupAdminHandler(t)
	defer cleanup()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET role = ? WHERE id = ?")).
		WithArgs(2, 1).
		WillReturnError(sql.ErrConnDone)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/users/1/role", bytes.NewBufferString(`{"role":2}`))
	w := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateUserRole(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
