package models

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// вспомогательная функция для мока
func setupTokenRepo(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *TokenRepository) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Ошибка создания mock DB: %v", err)
	}
	repo := NewTokenRepository(db)
	return db, mock, repo
}

func TestSaveRefreshToken(t *testing.T) {
	db, mock, repo := setupTokenRepo(t)
	defer db.Close()

	expires := time.Now().Add(24 * time.Hour).UTC()

	mock.ExpectExec(regexp.QuoteMeta(
		"INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES (?, ?, ?)")).
		WithArgs(int64(1), "token123", expires).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.SaveRefreshToken(1, "token123", expires)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetRefreshToken(t *testing.T) {
	db, mock, repo := setupTokenRepo(t)
	defer db.Close()

	expires := time.Now().Add(7 * 24 * time.Hour)
	rows := sqlmock.NewRows([]string{"user_id", "expires_at"}).
		AddRow(2, expires)

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT user_id, expires_at FROM refresh_tokens WHERE token = ?")).
		WithArgs("abc123").
		WillReturnRows(rows)

	userID, exp, err := repo.GetRefreshToken("abc123")
	assert.NoError(t, err)
	assert.Equal(t, int64(2), userID)
	assert.WithinDuration(t, expires, exp, time.Second)
}

func TestGetRefreshToken_NotFound(t *testing.T) {
	db, mock, repo := setupTokenRepo(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT user_id, expires_at FROM refresh_tokens WHERE token = ?")).
		WithArgs("missing").
		WillReturnError(sql.ErrNoRows)

	userID, exp, err := repo.GetRefreshToken("missing")
	assert.ErrorIs(t, err, sql.ErrNoRows)
	assert.Zero(t, userID)
	assert.True(t, exp.IsZero())
}

func TestDeleteRefreshToken(t *testing.T) {
	db, mock, repo := setupTokenRepo(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM refresh_tokens WHERE token = ?")).
		WithArgs("todelete").
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.DeleteRefreshToken("todelete")
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestDeleteAllForUser(t *testing.T) {
	db, mock, repo := setupTokenRepo(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM refresh_tokens WHERE user_id = ?")).
		WithArgs(int64(7)).
		WillReturnResult(sqlmock.NewResult(0, 3))

	err := repo.DeleteAllForUser(7)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
