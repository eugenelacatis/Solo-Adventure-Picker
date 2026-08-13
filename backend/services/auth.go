package services

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const sessionDuration = 30 * 24 * time.Hour

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func CreateSession(db *sql.DB, userId string) (string, time.Time, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", time.Time{}, err
	}
	token := base64.URLEncoding.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(sessionDuration)

	_, err := db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		token, userId, expiresAt,
	)
	if err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func ResolveSession(db *sql.DB, token string) (string, error) {
	var userId string
	var expiresAt time.Time
	err := db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE token = $1`, token,
	).Scan(&userId, &expiresAt)
	if err != nil {
		return "", err
	}
	if time.Now().After(expiresAt) {
		return "", errors.New("session expired")
	}
	return userId, nil
}
