package models

import (
	"database/sql"
	"time"
)

type User struct {
	ID           int64     `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"`
	Role         int       `json:"role"` // 0=user, 1=admin
	CreatedAt    time.Time `json:"created_at"`
}

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{DB: db}
}

// CreateUser создает нового пользователя
func (r *UserRepository) CreateUser(username, passwordHash string, role int) error {
	_, err := r.DB.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		username, passwordHash, role,
	)
	return err
}

// GetUserByUsername находит пользователя по имени
func (r *UserRepository) GetUserByUsername(username string) (*User, error) {
	var user User
	err := r.DB.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UserExists проверяет существование пользователя
func (r *UserRepository) UserExists(username string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)",
		username,
	).Scan(&exists)
	return exists, err
}
