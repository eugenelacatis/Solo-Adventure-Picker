package routes

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eugenelacatis/solo-adventure-picker/config"
	"github.com/eugenelacatis/solo-adventure-picker/models"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db := config.InitDB(":memory:")
	t.Cleanup(func() { db.Close() })
	return db
}

func insertTestAdventure(t *testing.T, db *sql.DB, adv models.Adventure) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO adventures (name, type, region, lat, lng) VALUES (?, ?, ?, ?, ?)`,
		adv.Name, adv.Type, adv.Region, adv.Lat, adv.Lng,
	)
	if err != nil {
		t.Fatalf("failed to insert test adventure: %v", err)
	}
}

// signUpTestUser signs up userId as a real account (email derived from
// userId) and returns its session cookie, so gated-route tests can act as
// that user across multiple requests without re-signing-up.
func signUpTestUser(t *testing.T, mux *http.ServeMux, userId string) *http.Cookie {
	t.Helper()
	signupBody, _ := json.Marshal(map[string]string{
		"email":           userId + "@example.com",
		"password":        "hunter2",
		"anonymousUserId": userId,
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupBody))
	signupRec := httptest.NewRecorder()
	mux.ServeHTTP(signupRec, signupReq)
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signUpTestUser: signup for %q failed with status %d: %s", userId, signupRec.Code, signupRec.Body.String())
	}

	for _, c := range signupRec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatalf("signUpTestUser: no session cookie returned for %q", userId)
	return nil
}

// authedRequest builds a request carrying the given session cookie.
func authedRequest(method, path string, body []byte, cookie *http.Cookie) *http.Request {
	var reader *bytes.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.AddCookie(cookie)
	return req
}

func TestWithCORS_CredentialedRequest_EchoesOriginNotWildcard(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodOptions, "/auth/signup", nil)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	gotOrigin := rec.Header().Get("Access-Control-Allow-Origin")
	if gotOrigin != "http://localhost:5173" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the echoed request origin (a wildcard is rejected by browsers on credentialed requests)", gotOrigin)
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want %q", rec.Header().Get("Access-Control-Allow-Credentials"), "true")
	}
}

func TestRandomHandler_ReturnsMatchingRegion(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})
	insertTestAdventure(t, db, models.Adventure{Name: "Some Other Place", Type: "hike", Region: "north-bay", Lat: 38.0, Lng: -122.7})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/random?region=bay-area", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var adv models.Adventure
	if err := json.Unmarshal(rec.Body.Bytes(), &adv); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if adv.Region != "bay-area" {
		t.Errorf("Region = %q, want %q", adv.Region, "bay-area")
	}
	if adv.Name != "Mount Tam" {
		t.Errorf("Name = %q, want %q", adv.Name, "Mount Tam")
	}
}

func TestRandomHandler_NoMatchingRegion_Returns404(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/random?region=nowhere", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestGetXPHandler_NoSession_Returns401(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/xp", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestGetXPHandler_NewUser_ReturnsZero(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "new-user-1")
	req := authedRequest(http.MethodGet, "/xp", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var body struct {
		TotalXp int `json:"totalXp"`
		Level   int `json:"level"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.TotalXp != 0 {
		t.Errorf("TotalXp = %d, want 0", body.TotalXp)
	}
	if body.Level != 1 {
		t.Errorf("Level = %d, want 1", body.Level)
	}
}

func TestAwardXPHandler_AccumulatesAndPersists(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	awardBody, _ := json.Marshal(map[string]interface{}{
		"adventureId": "adv-1",
		"xp":          150,
		"lat":         37.9235,
		"lng":         -122.5965,
	})

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/xp/add", awardBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify it persisted by fetching again.
	getReq := authedRequest(http.MethodGet, "/xp", nil, cookie)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	var body struct {
		TotalXp int `json:"totalXp"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.TotalXp != 150 {
		t.Errorf("TotalXp = %d, want 150 after award", body.TotalXp)
	}
}

func TestJournalHandler_SubmitsEntryAndAwardsBonusXP(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	journalBody, _ := json.Marshal(map[string]string{
		"adventureId": "adv-1",
		"text":        "Great hike today!",
	})

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/journal", journalBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	getReq := authedRequest(http.MethodGet, "/xp", nil, cookie)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	var body struct {
		TotalXp int `json:"totalXp"`
	}
	json.Unmarshal(getRec.Body.Bytes(), &body)
	if body.TotalXp <= 0 {
		t.Errorf("TotalXp = %d, want > 0 after journal submission awards bonus XP", body.TotalXp)
	}
}

func TestAchievementsHandler_ReflectsVisitedAndJournalCounts(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")

	_, err := db.Exec(`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES (?, ?, ?, ?)`,
		"user-1", "adv-1", 37.9, -122.5)
	if err != nil {
		t.Fatalf("failed to seed visited adventure: %v", err)
	}

	req := authedRequest(http.MethodGet, "/achievements", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var achievements []struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &achievements); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(achievements) != 1 || achievements[0].Id != "first-adventure" {
		t.Errorf("achievements = %+v, want one entry with id first-adventure", achievements)
	}
}

func TestGetVisitedHandler_ReturnsRecordedVisits(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	awardBody, _ := json.Marshal(map[string]interface{}{
		"adventureId": "adv-1",
		"xp":          150,
		"lat":         37.9235,
		"lng":         -122.5965,
	})
	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/xp/add", awardBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup: award XP failed with status %d: %s", rec.Code, rec.Body.String())
	}

	getReq := authedRequest(http.MethodGet, "/visited", nil, cookie)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", getRec.Code, http.StatusOK, getRec.Body.String())
	}

	var visited []models.VisitedEntry
	if err := json.Unmarshal(getRec.Body.Bytes(), &visited); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(visited) != 1 {
		t.Fatalf("visited = %+v, want 1 entry", visited)
	}
	if visited[0].AdventureId != "adv-1" || visited[0].Lat != 37.9235 || visited[0].Lng != -122.5965 {
		t.Errorf("visited[0] = %+v, want adventureId=adv-1 lat=37.9235 lng=-122.5965", visited[0])
	}
}

func TestGetVisitedHandler_NoVisits_ReturnsEmptyArray(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "never-visited-user")
	req := authedRequest(http.MethodGet, "/visited", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "[]\n" {
		t.Errorf("body = %q, want empty JSON array", rec.Body.String())
	}
}

func TestJournalHandler_EmptyText_Returns400(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	journalBody, _ := json.Marshal(map[string]string{
		"adventureId": "adv-1",
		"text":        "",
	})

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/journal", journalBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
