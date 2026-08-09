package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func signup(t *testing.T, mux *http.ServeMux, email, password string) *httptest.ResponseRecorder {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
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

	rec := signup(t, mux, "hiker@example.com", "hunter2")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	sessionCookie(t, rec)
}

func TestSignupHandler_DuplicateEmail_Returns409(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signup(t, mux, "hiker@example.com", "hunter2")
	rec := signup(t, mux, "hiker@example.com", "different-password")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusConflict, rec.Body.String())
	}
}

// TestSignupHandler_SecondSignupDoesNotOverwriteFirstAccount guards against
// the account-takeover bug this milestone fixed: signup used to accept a
// client-supplied anonymousUserId and upsert onto it, so two signups with
// the same (client-controlled) ID in the same browser session would
// silently overwrite the first account's email and password hash. Signup
// now always mints a fresh server-side user_id, so two signups can never
// collide on identity.
func TestSignupHandler_SecondSignupDoesNotOverwriteFirstAccount(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	firstRec := signup(t, mux, "first@example.com", "first-password")
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first signup status = %d, want %d; body: %s", firstRec.Code, http.StatusOK, firstRec.Body.String())
	}
	var first struct {
		UserId string `json:"userId"`
	}
	json.Unmarshal(firstRec.Body.Bytes(), &first)

	secondRec := signup(t, mux, "second@example.com", "second-password")
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second signup status = %d, want %d; body: %s", secondRec.Code, http.StatusOK, secondRec.Body.String())
	}
	var second struct {
		UserId string `json:"userId"`
	}
	json.Unmarshal(secondRec.Body.Bytes(), &second)

	if first.UserId == second.UserId {
		t.Fatalf("both signups got the same userId %q, want distinct accounts", first.UserId)
	}

	// The first account's credentials must still work.
	loginRec := login(t, mux, "first@example.com", "first-password")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login for first account status = %d, want %d; body: %s", loginRec.Code, http.StatusOK, loginRec.Body.String())
	}
}

func TestLoginHandler_CorrectCredentials_ReturnsSessionCookie(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	signup(t, mux, "hiker@example.com", "hunter2")

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

	signup(t, mux, "hiker@example.com", "hunter2")

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

	signupRec := signup(t, mux, "hiker@example.com", "hunter2")
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

	signupRec := signup(t, mux, "hiker@example.com", "hunter2")
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
