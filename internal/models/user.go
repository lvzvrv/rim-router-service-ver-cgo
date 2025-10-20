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

func (r *UserRepository) CreateUser(username, passwordHash string, role int) error {
	_, err := r.DB.Exec(
		"INSERT INTO users (username, password_hash, role) VALUES (?, ?, ?)",
		username, passwordHash, role,
	)
	return err
}

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

func (r *UserRepository) GetUserByID(id int64) (*User, error) {
	var user User
	err := r.DB.QueryRow(
		"SELECT id, username, password_hash, role, created_at FROM users WHERE id = ?",
		id,
	).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) UserExists(username string) (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE username = ?)",
		username,
	).Scan(&exists)
	return exists, err
}

// AdminExists проверяет, есть ли хотя бы один администратор (role = 1)
func (r *UserRepository) AdminExists() (bool, error) {
	var exists bool
	err := r.DB.QueryRow(
		"SELECT EXISTS(SELECT 1 FROM users WHERE role = 1)",
	).Scan(&exists)
	return exists, err
}

// GetAllUsers возвращает список всех пользователей (без паролей)
func (r *UserRepository) GetAllUsers() ([]User, error) {
	rows, err := r.DB.Query(`SELECT id, username, role, created_at FROM users ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Username, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// UpdateUserRole изменяет роль пользователя
func (r *UserRepository) UpdateUserRole(id int, role int) error {
	_, err := r.DB.Exec(`UPDATE users SET role = ? WHERE id = ?`, role, id)
	return err
}
