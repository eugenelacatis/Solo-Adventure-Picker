package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func signup(t *testing.T, mux *http.ServeMux, email, password, anonymousUserId string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"email":           email,
		"password":        password,
		"anonymousUserId": anonymousUserId,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func login(t *testing.T, mux *http.ServeMux, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func sessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()
	for _, c := range rec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatalf("no session cookie found in response")
	return nil
}

func TestSignupHandler_CreatesAccountAndReturnsSessionCookie(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	rec := signup(t, mux, "hiker@example.com", "hunter2", "anon-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	sessionCookie(t, rec)
}

func TestSignupHandler_DuplicateEmail_Returns409(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signup(t, mux, "hiker@example.com", "hunter2", "anon-1")
	rec := signup(t, mux, "hiker@example.com", "different-password", "anon-2")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

func TestSignupHandler_RekeysAnonymousData(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	// All XP/journal/visited routes require a session, so there is no live
	// HTTP path for an anonymous userId to accrue data pre-signup today.
	// Seed a row directly to model that case at the DB layer (e.g. legacy
	// data, or a future anonymous-write path) and confirm signup still
	// binds it to the new account correctly.
	_, err := db.Exec(
		`INSERT INTO users (user_id, total_xp) VALUES (?, ?)`,
		"anon-1", 150,
	)
	if err != nil {
		t.Fatalf("setup: failed to seed anonymous user XP: %v", err)
	}

	signupRec := signup(t, mux, "hiker@example.com", "hunter2", "anon-1")
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signup status = %d, want %d; body: %s", signupRec.Code, http.StatusOK, signupRec.Body.String())
	}
	cookie := sessionCookie(t, signupRec)

	getReq := httptest.NewRequest(http.MethodGet, "/xp", nil)
	getReq.AddCookie(cookie)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", getRec.Code, http.StatusOK, getRec.Body.String())
	}
	var body struct {
		TotalXp int `json:"totalXp"`
	}
	json.Unmarshal(getRec.Body.Bytes(), &body)
	if body.TotalXp != 150 {
		t.Errorf("TotalXp = %d, want 150 (re-keyed from anonymous user)", body.TotalXp)
	}
}

func TestLoginHandler_CorrectCredentials_ReturnsSessionCookie(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signup(t, mux, "hiker@example.com", "hunter2", "anon-1")

	rec := login(t, mux, "hiker@example.com", "hunter2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	sessionCookie(t, rec)
}

func TestLoginHandler_WrongPassword_Returns401(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signup(t, mux, "hiker@example.com", "hunter2", "anon-1")

	rec := login(t, mux, "hiker@example.com", "wrong-password")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestLoginHandler_UnknownEmail_Returns401(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	rec := login(t, mux, "nobody@example.com", "hunter2")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestAuthMeHandler_ValidSession_ReturnsUser(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signupRec := signup(t, mux, "hiker@example.com", "hunter2", "anon-1")
	cookie := sessionCookie(t, signupRec)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var body struct {
		Email string `json:"email"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	if body.Email != "hiker@example.com" {
		t.Errorf("Email = %q, want %q", body.Email, "hiker@example.com")
	}
}

func TestAuthMeHandler_NoSession_Returns401(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestLogoutHandler_InvalidatesSession(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signupRec := signup(t, mux, "hiker@example.com", "hunter2", "anon-1")
	cookie := sessionCookie(t, signupRec)

	logoutReq := httptest.NewRequest(http.MethodPost, "/auth/logout", nil)
	logoutReq.AddCookie(cookie)
	logoutRec := httptest.NewRecorder()
	mux.ServeHTTP(logoutRec, logoutReq)
	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, want %d; body: %s", logoutRec.Code, http.StatusOK, logoutRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/auth/me", nil)
	meReq.AddCookie(cookie)
	meRec := httptest.NewRecorder()
	mux.ServeHTTP(meRec, meReq)
	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("status after logout = %d, want %d", meRec.Code, http.StatusUnauthorized)
	}
}
