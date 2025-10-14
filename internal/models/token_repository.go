package models

import (
	"database/sql"
	"time"
)

type TokenRepository struct {
	DB *sql.DB
}

func NewTokenRepository(db *sql.DB) *TokenRepository {
	return &TokenRepository{DB: db}
}

func (r *TokenRepository) SaveRefreshToken(userID int64, token string, expiresAt time.Time) error {
	_, err := r.DB.Exec(`
        INSERT INTO refresh_tokens (user_id, token, expires_at) VALUES (?, ?, ?)
    `, userID, token, expiresAt.UTC())
	return err
}

func (r *TokenRepository) GetRefreshToken(token string) (userID int64, expiresAt time.Time, err error) {
	err = r.DB.QueryRow(`
        SELECT user_id, expires_at FROM refresh_tokens WHERE token = ?
    `, token).Scan(&userID, &expiresAt)
	return
}

func (r *TokenRepository) DeleteRefreshToken(token string) error {
	_, err := r.DB.Exec(`DELETE FROM refresh_tokens WHERE token = ?`, token)
	return err
}

func (r *TokenRepository) DeleteAllForUser(userID int64) error {
	_, err := r.DB.Exec(`DELETE FROM refresh_tokens WHERE user_id = ?`, userID)
	return err
}
