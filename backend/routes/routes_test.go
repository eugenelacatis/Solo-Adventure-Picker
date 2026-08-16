package routes

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/eugenelacatis/solo-adventure-picker/config"
	"github.com/eugenelacatis/solo-adventure-picker/models"
	"github.com/eugenelacatis/solo-adventure-picker/services"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5433/solo_adventure_picker?sslmode=disable"
	}
	db := config.InitDB(dsn)
	t.Cleanup(func() {
		db.Exec(`TRUNCATE TABLE sessions, journal_entries, visited_adventures, user_achievements, users, adventures RESTART IDENTITY CASCADE`)
		db.Close()
	})
	return db
}

func insertTestAdventure(t *testing.T, db *sql.DB, adv models.Adventure) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO adventures (name, type, region, effort, lat, lng) VALUES ($1, $2, $3, $4, $5, $6)`,
		adv.Name, adv.Type, adv.Region, adv.Effort, adv.Lat, adv.Lng,
	)
	if err != nil {
		t.Fatalf("failed to insert test adventure: %v", err)
	}
}

// signUpTestUser signs up a fresh account (email derived from the given
// label) and returns its session cookie, so gated-route tests can act as
// that user across multiple requests without re-signing-up. The account's
// server-assigned userId no longer matches label — signup mints its own
// random ID — so callers needing the actual userId should read it from the
// signup response instead of assuming it.
func signUpTestUser(t *testing.T, mux *http.ServeMux, label string) *http.Cookie {
	t.Helper()
	signupBody, _ := json.Marshal(map[string]string{
		"email":    label + "@example.com",
		"password": "hunter2",
	})
	signupReq := httptest.NewRequest(http.MethodPost, "/auth/signup", bytes.NewReader(signupBody))
	signupRec := httptest.NewRecorder()
	mux.ServeHTTP(signupRec, signupReq)
	if signupRec.Code != http.StatusOK {
		t.Fatalf("signUpTestUser: signup for %q failed with status %d: %s", label, signupRec.Code, signupRec.Body.String())
	}

	for _, c := range signupRec.Result().Cookies() {
		if c.Name == "session" {
			return c
		}
	}
	t.Fatalf("signUpTestUser: no session cookie returned for %q", label)
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

func TestRandomHandler_NoSession_Returns401(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/random?region=bay-area", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
}

func TestRandomHandler_ReturnsMatchingRegion(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})
	insertTestAdventure(t, db, models.Adventure{Name: "Some Other Place", Type: "hike", Region: "north-bay", Lat: 38.0, Lng: -122.7})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodGet, "/random?region=bay-area", nil, cookie)
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

func TestRandomHandler_ReturnsMatchingTypeAndEffort(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Effort: "hard", Lat: 37.9, Lng: -122.5})
	insertTestAdventure(t, db, models.Adventure{Name: "Easy Stroll", Type: "hike", Region: "bay-area", Effort: "easy", Lat: 37.8, Lng: -122.4})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodGet, "/random?type=hike&effort=easy", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var adv models.Adventure
	if err := json.Unmarshal(rec.Body.Bytes(), &adv); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if adv.Name != "Easy Stroll" || adv.Effort != "easy" {
		t.Errorf("adv = %+v, want Name=Easy Stroll Effort=easy", adv)
	}
}

func TestRandomHandler_EnforcesDailyRerollLimit(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")

	// dailyRerollAllowance is 5; exhaust it.
	for i := 0; i < 5; i++ {
		req := authedRequest(http.MethodGet, "/random", nil, cookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("reroll %d: status = %d, want %d; body: %s", i+1, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	req := authedRequest(http.MethodGet, "/random", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusTooManyRequests, rec.Body.String())
	}
}

func TestRandomHandler_AllowsRerollAfterResetWindowPasses(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	var userId string
	if err := db.QueryRow(`SELECT user_id FROM sessions WHERE token = $1`, cookie.Value).Scan(&userId); err != nil {
		t.Fatalf("failed to resolve userId from session: %v", err)
	}

	for i := 0; i < 5; i++ {
		req := authedRequest(http.MethodGet, "/random", nil, cookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("reroll %d: status = %d, want %d; body: %s", i+1, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	// Simulate the daily reset window having already passed.
	if _, err := db.Exec(`UPDATE users SET reroll_reset_at = now() - interval '1 minute' WHERE user_id = $1`, userId); err != nil {
		t.Fatalf("failed to backdate reroll_reset_at: %v", err)
	}

	req := authedRequest(http.MethodGet, "/random", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestRerollStatusHandler_ReflectsRemainingTokens(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")

	statusReq := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", statusRec.Code, http.StatusOK, statusRec.Body.String())
	}
	var before struct {
		RerollTokens int `json:"rerollTokens"`
	}
	if err := json.Unmarshal(statusRec.Body.Bytes(), &before); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if before.RerollTokens != 5 {
		t.Errorf("RerollTokens = %d, want 5 before any reroll", before.RerollTokens)
	}

	rerollReq := authedRequest(http.MethodGet, "/random", nil, cookie)
	rerollRec := httptest.NewRecorder()
	mux.ServeHTTP(rerollRec, rerollReq)
	if rerollRec.Code != http.StatusOK {
		t.Fatalf("setup: reroll failed with status %d: %s", rerollRec.Code, rerollRec.Body.String())
	}

	statusReq2 := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec2 := httptest.NewRecorder()
	mux.ServeHTTP(statusRec2, statusReq2)
	var after struct {
		RerollTokens int `json:"rerollTokens"`
	}
	if err := json.Unmarshal(statusRec2.Body.Bytes(), &after); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if after.RerollTokens != 4 {
		t.Errorf("RerollTokens = %d, want 4 after one reroll", after.RerollTokens)
	}
}

func TestRandomHandler_NoMatchingRegion_Returns404(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9, Lng: -122.5})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodGet, "/random?region=nowhere", nil, cookie)
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

// insertTestAdventureWithXP inserts an adventure with an explicit xp_value,
// returning its row ID, for tests that need POST /visited to award a known
// amount looked up server-side (rather than trusting a client-supplied
// amount, which is exactly what this endpoint stopped doing).
func insertTestAdventureWithXP(t *testing.T, db *sql.DB, name string, xpValue int, lat, lng float64) int64 {
	t.Helper()
	var id int64
	err := db.QueryRow(
		`INSERT INTO adventures (name, type, region, xp_value, lat, lng) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		name, "hike", "bay-area", xpValue, lat, lng,
	).Scan(&id)
	if err != nil {
		t.Fatalf("failed to insert test adventure: %v", err)
	}
	return id
}

func TestPostVisitedHandler_AwardsAdventuresStoredXPNotClientSuppliedXP(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)

	// The client only sends adventureId; any xp/lat/lng fields it might
	// include are ignored, since the amount is looked up server-side.
	visitBody, _ := json.Marshal(map[string]interface{}{
		"adventureId": strconv.FormatInt(advId, 10),
		"xp":          999999,
	})

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
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
	if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if body.TotalXp != 150 {
		t.Errorf("TotalXp = %d, want 150 (the adventure's stored xp_value, not the client-supplied 999999)", body.TotalXp)
	}
}

func TestPostVisitedHandler_DuplicateVisit_DoesNotAwardXPTwice(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	visitBody, _ := json.Marshal(map[string]string{"adventureId": strconv.FormatInt(advId, 10)})

	cookie := signUpTestUser(t, mux, "user-1")

	firstReq := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	firstRec := httptest.NewRecorder()
	mux.ServeHTTP(firstRec, firstReq)
	if firstRec.Code != http.StatusOK {
		t.Fatalf("first visit status = %d, want %d; body: %s", firstRec.Code, http.StatusOK, firstRec.Body.String())
	}

	secondReq := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	secondRec := httptest.NewRecorder()
	mux.ServeHTTP(secondRec, secondReq)
	if secondRec.Code != http.StatusOK {
		t.Fatalf("second visit status = %d, want %d; body: %s", secondRec.Code, http.StatusOK, secondRec.Body.String())
	}

	var secondBody struct {
		TotalXp        int  `json:"totalXp"`
		AlreadyVisited bool `json:"alreadyVisited"`
	}
	if err := json.Unmarshal(secondRec.Body.Bytes(), &secondBody); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !secondBody.AlreadyVisited {
		t.Error("alreadyVisited = false on second visit, want true")
	}
	if secondBody.TotalXp != 150 {
		t.Errorf("TotalXp after duplicate visit = %d, want 150 (unchanged, not double-counted)", secondBody.TotalXp)
	}

	var visitCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM visited_adventures WHERE user_id = (SELECT user_id FROM sessions WHERE token = $1)`, cookie.Value).Scan(&visitCount); err != nil {
		t.Fatalf("failed to count visited_adventures rows: %v", err)
	}
	if visitCount != 1 {
		t.Errorf("visited_adventures rows = %d, want 1 (duplicate visit must not insert a second row)", visitCount)
	}
}

func TestPostVisitedHandler_UnknownAdventure_Returns404(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	visitBody, _ := json.Marshal(map[string]string{"adventureId": "99999"})

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusNotFound, rec.Body.String())
	}
}

func TestJournalHandler_SubmitsEntryAndAwardsBonusXP(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	journalBody, _ := json.Marshal(map[string]string{
		"adventureId": strconv.FormatInt(advId, 10),
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

func TestJournalHandler_AwardsBonusRerollToken(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	journalBody, _ := json.Marshal(map[string]string{
		"adventureId": strconv.FormatInt(advId, 10),
		"text":        "Great hike today!",
	})

	cookie := signUpTestUser(t, mux, "user-1")

	// Consume the daily quest bonus via an unrelated visit first, so the
	// journal submission below is isolated to its own per-entry bonus
	// rather than also picking up the one-time daily-quest bonus (see
	// TestQuestHandler_DoesNotDoubleAwardOnSecondActivityInSameDay for the
	// same pattern).
	visitBody, _ := json.Marshal(map[string]string{"adventureId": strconv.FormatInt(advId, 10)})
	visitReq := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	visitRec := httptest.NewRecorder()
	mux.ServeHTTP(visitRec, visitReq)
	if visitRec.Code != http.StatusOK {
		t.Fatalf("mark visited failed with status %d: %s", visitRec.Code, visitRec.Body.String())
	}

	statusReq := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	var before struct {
		RerollTokens int `json:"rerollTokens"`
	}
	json.Unmarshal(statusRec.Body.Bytes(), &before)

	req := authedRequest(http.MethodPost, "/journal", journalBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	statusReq2 := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec2 := httptest.NewRecorder()
	mux.ServeHTTP(statusRec2, statusReq2)
	var after struct {
		RerollTokens int `json:"rerollTokens"`
	}
	json.Unmarshal(statusRec2.Body.Bytes(), &after)

	if after.RerollTokens != before.RerollTokens+journalBonusRerollTokens {
		t.Errorf("RerollTokens = %d, want %d (before %d + bonus %d)", after.RerollTokens, before.RerollTokens+journalBonusRerollTokens, before.RerollTokens, journalBonusRerollTokens)
	}
}

func TestGetJournalHandler_ReturnsSubmittedEntries(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	journalBody, _ := json.Marshal(map[string]string{
		"adventureId": strconv.FormatInt(advId, 10),
		"text":        "Great hike today!",
	})

	cookie := signUpTestUser(t, mux, "user-1")
	postReq := authedRequest(http.MethodPost, "/journal", journalBody, cookie)
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("setup: journal submission failed with status %d: %s", postRec.Code, postRec.Body.String())
	}

	getReq := authedRequest(http.MethodGet, "/journal", nil, cookie)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", getRec.Code, http.StatusOK, getRec.Body.String())
	}

	var entries []models.JournalEntryView
	if err := json.Unmarshal(getRec.Body.Bytes(), &entries); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	advIdStr := strconv.FormatInt(advId, 10)
	if len(entries) != 1 || entries[0].AdventureId != advIdStr || entries[0].AdventureName != "Mount Tam" || entries[0].Text != "Great hike today!" {
		t.Errorf("entries = %+v, want one entry with adventureId=%s adventureName=Mount Tam text='Great hike today!'", entries, advIdStr)
	}
}

func TestGetJournalHandler_NoEntries_ReturnsEmptyArray(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "never-journaled-user")
	req := authedRequest(http.MethodGet, "/journal", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if rec.Body.String() != "[]\n" {
		t.Errorf("body = %q, want empty JSON array", rec.Body.String())
	}
}

func TestAchievementsHandler_ReflectsVisitedAndJournalCounts(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")

	// signUpTestUser's account gets a server-generated userId, not the
	// "user-1" label passed in, so look up the real id via the session
	// before seeding a row against it directly.
	var userId string
	if err := db.QueryRow(`SELECT user_id FROM sessions WHERE token = $1`, cookie.Value).Scan(&userId); err != nil {
		t.Fatalf("failed to resolve userId from session: %v", err)
	}

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9, -122.5)
	_, err := db.Exec(`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES ($1, $2, $3, $4)`,
		userId, advId, 37.9, -122.5)
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

func TestAchievementsHandler_PersistsUnlockedAchievementOnce(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	var userId string
	if err := db.QueryRow(`SELECT user_id FROM sessions WHERE token = $1`, cookie.Value).Scan(&userId); err != nil {
		t.Fatalf("failed to resolve userId from session: %v", err)
	}

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9, -122.5)
	if _, err := db.Exec(`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES ($1, $2, $3, $4)`,
		userId, advId, 37.9, -122.5); err != nil {
		t.Fatalf("failed to seed visited adventure: %v", err)
	}

	for i := 0; i < 2; i++ {
		req := authedRequest(http.MethodGet, "/achievements", nil, cookie)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("call %d: status = %d, want %d; body: %s", i+1, rec.Code, http.StatusOK, rec.Body.String())
		}
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM user_achievements WHERE user_id = $1 AND achievement_id = $2`,
		userId, "first-adventure").Scan(&count); err != nil {
		t.Fatalf("failed to count persisted achievements: %v", err)
	}
	if count != 1 {
		t.Errorf("user_achievements rows for first-adventure = %d, want 1 (no duplicate on repeat call)", count)
	}
}

func TestAchievementsHandler_AwardsRerollTokensOnlyForNewlyUnlocked(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	var userId string
	if err := db.QueryRow(`SELECT user_id FROM sessions WHERE token = $1`, cookie.Value).Scan(&userId); err != nil {
		t.Fatalf("failed to resolve userId from session: %v", err)
	}

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9, -122.5)
	if _, err := db.Exec(`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES ($1, $2, $3, $4)`,
		userId, advId, 37.9, -122.5); err != nil {
		t.Fatalf("failed to seed visited adventure: %v", err)
	}

	// First call: first-adventure unlocks for the first time.
	req := authedRequest(http.MethodGet, "/achievements", nil, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var afterFirst struct {
		RerollTokens int `json:"rerollTokens"`
	}
	statusReq := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	json.Unmarshal(statusRec.Body.Bytes(), &afterFirst)

	if afterFirst.RerollTokens != dailyRerollAllowance+achievementRerollTokens {
		t.Errorf("RerollTokens after first unlock = %d, want %d", afterFirst.RerollTokens, dailyRerollAllowance+achievementRerollTokens)
	}

	// Second call: same achievement, already persisted — no additional tokens.
	req2 := authedRequest(http.MethodGet, "/achievements", nil, cookie)
	rec2 := httptest.NewRecorder()
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec2.Code, http.StatusOK, rec2.Body.String())
	}

	var afterSecond struct {
		RerollTokens int `json:"rerollTokens"`
	}
	statusReq2 := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec2 := httptest.NewRecorder()
	mux.ServeHTTP(statusRec2, statusReq2)
	json.Unmarshal(statusRec2.Body.Bytes(), &afterSecond)

	if afterSecond.RerollTokens != afterFirst.RerollTokens {
		t.Errorf("RerollTokens after repeat call = %d, want unchanged %d", afterSecond.RerollTokens, afterFirst.RerollTokens)
	}
}

func TestPersistNewAchievements_ReturnsOnlyNewlyUnlocked(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	cookie := signUpTestUser(t, mux, "user-1")
	var userId string
	if err := db.QueryRow(`SELECT user_id FROM sessions WHERE token = $1`, cookie.Value).Scan(&userId); err != nil {
		t.Fatalf("failed to resolve userId from session: %v", err)
	}

	first := []services.Achievement{{Id: "first-adventure", Name: "First Adventure"}}
	newlyUnlocked, err := persistNewAchievements(db, userId, first)
	if err != nil {
		t.Fatalf("persistNewAchievements: %v", err)
	}
	if len(newlyUnlocked) != 1 || newlyUnlocked[0].Id != "first-adventure" {
		t.Fatalf("newlyUnlocked = %+v, want one entry with id first-adventure", newlyUnlocked)
	}

	both := []services.Achievement{
		{Id: "first-adventure", Name: "First Adventure"},
		{Id: "five-adventures", Name: "Five Adventures"},
	}
	newlyUnlocked, err = persistNewAchievements(db, userId, both)
	if err != nil {
		t.Fatalf("persistNewAchievements: %v", err)
	}
	if len(newlyUnlocked) != 1 || newlyUnlocked[0].Id != "five-adventures" {
		t.Fatalf("newlyUnlocked = %+v, want only five-adventures (first-adventure already persisted)", newlyUnlocked)
	}
}

func TestQuestHandler_CompletesOnFirstVisitOfTheDay(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	cookie := signUpTestUser(t, mux, "user-1")

	questReq := authedRequest(http.MethodGet, "/quest", nil, cookie)
	questRec := httptest.NewRecorder()
	mux.ServeHTTP(questRec, questReq)
	var before struct {
		CompletedToday bool `json:"completedToday"`
	}
	json.Unmarshal(questRec.Body.Bytes(), &before)
	if before.CompletedToday {
		t.Fatalf("quest should not be completed before any activity")
	}

	statusReq := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	var tokensBefore struct {
		RerollTokens int `json:"rerollTokens"`
	}
	json.Unmarshal(statusRec.Body.Bytes(), &tokensBefore)

	visitBody, _ := json.Marshal(map[string]string{"adventureId": strconv.FormatInt(advId, 10)})
	visitReq := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	visitRec := httptest.NewRecorder()
	mux.ServeHTTP(visitRec, visitReq)
	if visitRec.Code != http.StatusOK {
		t.Fatalf("mark visited failed with status %d: %s", visitRec.Code, visitRec.Body.String())
	}

	questReq2 := authedRequest(http.MethodGet, "/quest", nil, cookie)
	questRec2 := httptest.NewRecorder()
	mux.ServeHTTP(questRec2, questReq2)
	var after struct {
		CompletedToday bool `json:"completedToday"`
	}
	json.Unmarshal(questRec2.Body.Bytes(), &after)
	if !after.CompletedToday {
		t.Errorf("quest should be completed after a new visit")
	}

	statusReq2 := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec2 := httptest.NewRecorder()
	mux.ServeHTTP(statusRec2, statusReq2)
	var tokensAfter struct {
		RerollTokens int `json:"rerollTokens"`
	}
	json.Unmarshal(statusRec2.Body.Bytes(), &tokensAfter)
	if tokensAfter.RerollTokens != tokensBefore.RerollTokens+dailyQuestRerollTokens {
		t.Errorf("RerollTokens = %d, want %d (before %d + quest bonus %d)", tokensAfter.RerollTokens, tokensBefore.RerollTokens+dailyQuestRerollTokens, tokensBefore.RerollTokens, dailyQuestRerollTokens)
	}
}

func TestQuestHandler_DoesNotDoubleAwardOnSecondActivityInSameDay(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	cookie := signUpTestUser(t, mux, "user-1")

	visitBody, _ := json.Marshal(map[string]string{"adventureId": strconv.FormatInt(advId, 10)})
	visitReq := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	visitRec := httptest.NewRecorder()
	mux.ServeHTTP(visitRec, visitReq)
	if visitRec.Code != http.StatusOK {
		t.Fatalf("mark visited failed with status %d: %s", visitRec.Code, visitRec.Body.String())
	}

	statusReq := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec := httptest.NewRecorder()
	mux.ServeHTTP(statusRec, statusReq)
	var afterVisit struct {
		RerollTokens int `json:"rerollTokens"`
	}
	json.Unmarshal(statusRec.Body.Bytes(), &afterVisit)

	journalBody, _ := json.Marshal(map[string]string{
		"adventureId": strconv.FormatInt(advId, 10),
		"text":        "Second thing today",
	})
	journalReq := authedRequest(http.MethodPost, "/journal", journalBody, cookie)
	journalRec := httptest.NewRecorder()
	mux.ServeHTTP(journalRec, journalReq)
	if journalRec.Code != http.StatusOK {
		t.Fatalf("journal submission failed with status %d: %s", journalRec.Code, journalRec.Body.String())
	}

	statusReq2 := authedRequest(http.MethodGet, "/reroll-status", nil, cookie)
	statusRec2 := httptest.NewRecorder()
	mux.ServeHTTP(statusRec2, statusReq2)
	var afterJournal struct {
		RerollTokens int `json:"rerollTokens"`
	}
	json.Unmarshal(statusRec2.Body.Bytes(), &afterJournal)

	// Journal still awards its own per-entry bonus, but the quest itself
	// should not grant a second dailyQuestRerollTokens bonus today.
	if afterJournal.RerollTokens != afterVisit.RerollTokens+journalBonusRerollTokens {
		t.Errorf("RerollTokens after second activity = %d, want %d (only journal bonus, no repeat quest bonus)", afterJournal.RerollTokens, afterVisit.RerollTokens+journalBonusRerollTokens)
	}
}

func TestGetVisitedHandler_ReturnsRecordedVisits(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	advId := insertTestAdventureWithXP(t, db, "Mount Tam", 150, 37.9235, -122.5965)
	visitBody, _ := json.Marshal(map[string]string{"adventureId": strconv.FormatInt(advId, 10)})

	cookie := signUpTestUser(t, mux, "user-1")
	req := authedRequest(http.MethodPost, "/visited", visitBody, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup: mark visited failed with status %d: %s", rec.Code, rec.Body.String())
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
	advIdStr := strconv.FormatInt(advId, 10)
	if visited[0].AdventureId != advIdStr || visited[0].Name != "Mount Tam" || visited[0].Region != "bay-area" || visited[0].Lat != 37.9235 || visited[0].Lng != -122.5965 {
		t.Errorf("visited[0] = %+v, want adventureId=%s name=Mount Tam region=bay-area lat=37.9235 lng=-122.5965", visited[0], advIdStr)
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

func TestPostTrail_PointNearAdventure_MarksVisitedAndAwardsXp(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9235, Lng: -122.5965})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	cookie := signUpTestUser(t, mux, "trail-match")

	// ~30ft from the adventure — well within the 150ft match radius.
	body, _ := json.Marshal(map[string]interface{}{
		"points": []map[string]interface{}{
			{"lat": 37.92353, "lng": -122.5965, "recordedAt": "2026-08-13T10:00:00Z"},
		},
	})
	req := authedRequest(http.MethodPost, "/trail", body, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		NewlyVisited []models.Adventure `json:"newlyVisited"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(resp.NewlyVisited) != 1 || resp.NewlyVisited[0].Name != "Mount Tam" {
		t.Fatalf("newlyVisited = %+v, want [Mount Tam]", resp.NewlyVisited)
	}

	var visitedCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM visited_adventures WHERE user_id = (SELECT user_id FROM sessions WHERE token = $1)`, cookie.Value).Scan(&visitedCount); err != nil {
		t.Fatalf("query visited_adventures: %v", err)
	}
	if visitedCount != 1 {
		t.Errorf("visited_adventures count = %d, want 1", visitedCount)
	}
}

func TestPostTrail_PointFarFromAdventure_DoesNotMatch(t *testing.T) {
	db := newTestDB(t)
	insertTestAdventure(t, db, models.Adventure{Name: "Mount Tam", Type: "hike", Region: "bay-area", Lat: 37.9235, Lng: -122.5965})

	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	cookie := signUpTestUser(t, mux, "trail-nomatch")

	// ~5 miles away — outside the 150ft match radius.
	body, _ := json.Marshal(map[string]interface{}{
		"points": []map[string]interface{}{
			{"lat": 37.87, "lng": -122.53, "recordedAt": "2026-08-13T10:00:00Z"},
		},
	})
	req := authedRequest(http.MethodPost, "/trail", body, cookie)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		NewlyVisited []models.Adventure `json:"newlyVisited"`
	}
	json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.NewlyVisited) != 0 {
		t.Errorf("newlyVisited = %+v, want empty", resp.NewlyVisited)
	}
}

func TestPostTrail_NoSession_Returns401(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	body, _ := json.Marshal(map[string]interface{}{"points": []map[string]interface{}{}})
	req := httptest.NewRequest(http.MethodPost, "/trail", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestGetTrail_ReturnsPointsInChronologicalOrder(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	cookie := signUpTestUser(t, mux, "trail-get")

	body, _ := json.Marshal(map[string]interface{}{
		"points": []map[string]interface{}{
			{"lat": 37.1, "lng": -122.1, "recordedAt": "2026-08-13T10:00:00Z"},
			{"lat": 37.2, "lng": -122.2, "recordedAt": "2026-08-13T10:01:00Z"},
		},
	})
	postReq := authedRequest(http.MethodPost, "/trail", body, cookie)
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("seed POST /trail failed: status %d, body %s", postRec.Code, postRec.Body.String())
	}

	getReq := authedRequest(http.MethodGet, "/trail", nil, cookie)
	getRec := httptest.NewRecorder()
	mux.ServeHTTP(getRec, getReq)

	if getRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", getRec.Code, getRec.Body.String())
	}

	var points []models.TrailPoint
	if err := json.NewDecoder(getRec.Body).Decode(&points); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(points) != 2 {
		t.Fatalf("len(points) = %d, want 2", len(points))
	}
	if points[0].Lat != 37.1 || points[1].Lat != 37.2 {
		t.Errorf("points = %+v, want chronological order (37.1 then 37.2)", points)
	}
}

func TestGetTrail_LimitAndSinceParamsBoundResults(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)
	cookie := signUpTestUser(t, mux, "trail-bounds")

	body, _ := json.Marshal(map[string]interface{}{
		"points": []map[string]interface{}{
			{"lat": 37.1, "lng": -122.1, "recordedAt": "2026-08-13T10:00:00Z"},
			{"lat": 37.2, "lng": -122.2, "recordedAt": "2026-08-13T10:01:00Z"},
			{"lat": 37.3, "lng": -122.3, "recordedAt": "2026-08-13T10:02:00Z"},
		},
	})
	postReq := authedRequest(http.MethodPost, "/trail", body, cookie)
	postRec := httptest.NewRecorder()
	mux.ServeHTTP(postRec, postReq)
	if postRec.Code != http.StatusOK {
		t.Fatalf("seed POST /trail failed: status %d, body %s", postRec.Code, postRec.Body.String())
	}

	limitReq := authedRequest(http.MethodGet, "/trail?limit=2", nil, cookie)
	limitRec := httptest.NewRecorder()
	mux.ServeHTTP(limitRec, limitReq)
	if limitRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", limitRec.Code, limitRec.Body.String())
	}
	var limitedPoints []models.TrailPoint
	if err := json.NewDecoder(limitRec.Body).Decode(&limitedPoints); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(limitedPoints) != 2 {
		t.Fatalf("len(limitedPoints) = %d, want 2", len(limitedPoints))
	}
	if limitedPoints[0].Lat != 37.1 || limitedPoints[1].Lat != 37.2 {
		t.Errorf("limitedPoints = %+v, want the 2 oldest points", limitedPoints)
	}

	sinceReq := authedRequest(http.MethodGet, "/trail?since=2026-08-13T10:01:30Z", nil, cookie)
	sinceRec := httptest.NewRecorder()
	mux.ServeHTTP(sinceRec, sinceReq)
	if sinceRec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", sinceRec.Code, sinceRec.Body.String())
	}
	var sincePoints []models.TrailPoint
	if err := json.NewDecoder(sinceRec.Body).Decode(&sincePoints); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(sincePoints) != 1 || sincePoints[0].Lat != 37.3 {
		t.Errorf("sincePoints = %+v, want only the point recorded after the since timestamp", sincePoints)
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
