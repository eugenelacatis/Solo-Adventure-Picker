package services

import (
	"testing"
	"time"

	"github.com/eugenelacatis/solo-adventure-picker/config"
)

func TestHashPassword_CheckPassword_RoundTrip(t *testing.T) {
	hash, err := HashPassword("correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "correct-horse-battery-staple" {
		t.Fatalf("HashPassword did not hash the input")
	}

	if err := CheckPassword(hash, "correct-horse-battery-staple"); err != nil {
		t.Errorf("CheckPassword failed on correct password: %v", err)
	}
	if err := CheckPassword(hash, "wrong-password"); err == nil {
		t.Errorf("CheckPassword succeeded on incorrect password, want error")
	}
}

func TestCreateSession_ResolveSession_RoundTrip(t *testing.T) {
	db := config.InitDB(":memory:")
	defer db.Close()

	token, expiresAt, err := CreateSession(db, "user-1")
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if token == "" {
		t.Fatalf("CreateSession returned empty token")
	}
	if !expiresAt.After(time.Now()) {
		t.Errorf("expiresAt = %v, want a future time", expiresAt)
	}

	userId, err := ResolveSession(db, token)
	if err != nil {
		t.Fatalf("ResolveSession returned error for valid token: %v", err)
	}
	if userId != "user-1" {
		t.Errorf("ResolveSession userId = %q, want %q", userId, "user-1")
	}
}

func TestResolveSession_UnknownToken_ReturnsError(t *testing.T) {
	db := config.InitDB(":memory:")
	defer db.Close()

	if _, err := ResolveSession(db, "does-not-exist"); err == nil {
		t.Errorf("ResolveSession succeeded for unknown token, want error")
	}
}

func TestResolveSession_ExpiredToken_ReturnsError(t *testing.T) {
	db := config.InitDB(":memory:")
	defer db.Close()

	_, err := db.Exec(
		`INSERT INTO sessions (token, user_id, expires_at) VALUES (?, ?, ?)`,
		"expired-token", "user-1", time.Now().Add(-time.Hour),
	)
	if err != nil {
		t.Fatalf("failed to seed expired session: %v", err)
	}

	if _, err := ResolveSession(db, "expired-token"); err == nil {
		t.Errorf("ResolveSession succeeded for expired token, want error")
	}
}
