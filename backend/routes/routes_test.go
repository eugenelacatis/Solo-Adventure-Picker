package routes

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/EugeneL97/solo-adventure-picker/config"
	"github.com/EugeneL97/solo-adventure-picker/models"
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

func TestGetXPHandler_NewUser_ReturnsZero(t *testing.T) {
	db := newTestDB(t)
	mux := http.NewServeMux()
	RegisterRoutes(mux, db)

	req := httptest.NewRequest(http.MethodGet, "/xp/new-user-1", nil)
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

	req := httptest.NewRequest(http.MethodPost, "/xp/user-1/add", bytes.NewReader(awardBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	// Verify it persisted by fetching again.
	getReq := httptest.NewRequest(http.MethodGet, "/xp/user-1", nil)
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

	req := httptest.NewRequest(http.MethodPost, "/journal/user-1", bytes.NewReader(journalBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/xp/user-1", nil)
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

	_, err := db.Exec(`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES (?, ?, ?, ?)`,
		"user-1", "adv-1", 37.9, -122.5)
	if err != nil {
		t.Fatalf("failed to seed visited adventure: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/achievements/user-1", nil)
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
	req := httptest.NewRequest(http.MethodPost, "/xp/user-1/add", bytes.NewReader(awardBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("setup: award XP failed with status %d: %s", rec.Code, rec.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/visited/user-1", nil)
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

	req := httptest.NewRequest(http.MethodGet, "/visited/never-visited-user", nil)
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

	req := httptest.NewRequest(http.MethodPost, "/journal/user-1", bytes.NewReader(journalBody))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
