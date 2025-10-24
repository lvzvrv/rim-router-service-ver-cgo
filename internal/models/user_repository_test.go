package models

import (
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock, *UserRepository) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Не удалось создать mock DB: %v", err)
	}
	repo := NewUserRepository(db)
	return db, mock, repo
}

func TestCreateUser(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)")).
		WithArgs("alice", "hashed_pass", 1).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.CreateUser("alice", "hashed_pass", 1)
	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByUsername(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "username", "password_hash", "role", "created_at"}).
		AddRow(1, "bob", "hash123", 0, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?")).
		WithArgs("bob").
		WillReturnRows(rows)

	user, err := repo.GetUserByUsername("bob")
	assert.NoError(t, err)
	assert.Equal(t, int64(1), user.ID)
	assert.Equal(t, "bob", user.Username)
	assert.Equal(t, 0, user.Role)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetUserByUsername_NoRows(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?")).
		WithArgs("ghost").
		WillReturnError(sql.ErrNoRows)

	user, err := repo.GetUserByUsername("ghost")
	assert.Nil(t, user)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUserExists(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)")).
		WithArgs("john").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

	exists, err := repo.UserExists("john")
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestAdminExists(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT EXISTS(SELECT 1 FROM users WHERE role = 1)")).
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

	exists, err := repo.AdminExists()
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestUpdateUserRole(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("UPDATE users SET role = ? WHERE id = ?")).
		WithArgs(1, 2).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err := repo.UpdateUserRole(2, 1)
	assert.NoError(t, err)
}

func TestGetAllUsers(t *testing.T) {
	db, mock, repo := setupMockDB(t)
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id", "username", "role", "created_at"}).
		AddRow(1, "admin", 1, time.Now()).
		AddRow(2, "user", 0, time.Now())

	mock.ExpectQuery(regexp.QuoteMeta(
		"SELECT id, username, role, created_at FROM users ORDER BY id ASC")).
		WillReturnRows(rows)

	users, err := repo.GetAllUsers()
	assert.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "admin", users[0].Username)
	assert.Equal(t, 1, users[0].Role)
	assert.Equal(t, "user", users[1].Username)
	assert.Equal(t, 0, users[1].Role)
}
