package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/eugenelacatis/solo-adventure-picker/models"
	"github.com/eugenelacatis/solo-adventure-picker/services"
	"github.com/eugenelacatis/solo-adventure-picker/utils"
)

// allowedOrigin is the production CORS allowlist. When unset (e.g. local
// dev), withCORS falls back to echoing back whatever Origin the browser
// sends, matching the previous unconditional-echo behavior.
var allowedOrigin = os.Getenv("ALLOWED_ORIGIN")

// withCORS wraps a handler so every route (including preflight OPTIONS
// requests browsers send before non-simple methods like POST with a JSON
// body) gets the same CORS headers. Registering headers per-handler and
// missing OPTIONS handling on POST routes causes browser preflight requests
// to fail even though a direct (non-browser) request would succeed.
//
// Session cookies require credentialed requests, and browsers reject
// "Access-Control-Allow-Origin: *" on credentialed requests outright — the
// origin must be echoed back explicitly, paired with
// Access-Control-Allow-Credentials: true.
func withCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && (allowedOrigin == "" || origin == allowedOrigin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func RegisterRoutes(mux *http.ServeMux, db *sql.DB) {

	mux.HandleFunc("/random", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		region := r.URL.Query().Get("region")

		query := `SELECT id, name, type, region, scenery, effort, duration, description, xp_value, lat, lng FROM adventures`
		args := []interface{}{}
		if region != "" {
			query += ` WHERE region = $1`
			args = append(args, region)
		}
		query += ` ORDER BY RANDOM() LIMIT 1`

		row := db.QueryRow(query, args...)

		var adv models.Adventure
		var advType, scenery, effort, duration, description sql.NullString
		var xpValue sql.NullInt64
		var lat, lng sql.NullFloat64

		err := row.Scan(&adv.ID, &adv.Name, &advType, &adv.Region, &scenery, &effort, &duration, &description, &xpValue, &lat, &lng)
		if err != nil {
			utils.WriteJSONError(w, http.StatusNotFound, utils.APIError{
				Error:   "No matching adventure found.",
				Code:    1001,
				Details: "Region either has no adventures or the database is down. Womp womp."})
			return
		}

		adv.Type = advType.String
		adv.Scenery = scenery.String
		adv.Effort = effort.String
		adv.Duration = duration.String
		adv.Description = description.String
		adv.XPValue = int(xpValue.Int64)
		adv.Lat = lat.Float64
		adv.Lng = lng.Float64

		json.NewEncoder(w).Encode(adv)
	})))

	mux.HandleFunc("/xp", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		totalXp, err := getTotalXp(db, userId)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch XP.",
				Code:  1003,
			})
			return
		}

		writeXpResponse(w, totalXp, false)
	})))

	mux.HandleFunc("/visited", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		switch r.Method {
		case http.MethodPost:
			postVisited(w, r, db, userId)
		default:
			getVisited(w, r, db, userId)
		}
	})))

	mux.HandleFunc("/journal", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		var body struct {
			AdventureId string `json:"adventureId"`
			Text        string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Text == "" {
			utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
				Error:   "Journal entry text is required.",
				Code:    1007,
				Details: "You have to actually write something. Womp womp.",
			})
			return
		}

		adventureId, err := strconv.ParseInt(body.AdventureId, 10, 64)
		if err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
				Error:   "Invalid adventureId.",
				Code:    1007,
				Details: "adventureId must be a valid adventure ID.",
			})
			return
		}

		_, err = db.Exec(
			`INSERT INTO journal_entries (user_id, adventure_id, text, created_at) VALUES ($1, $2, $3, now())`,
			userId, adventureId, body.Text,
		)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to save journal entry.",
				Code:  1008,
			})
			return
		}

		if err := addXp(db, userId, journalBonusXp); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to award journal bonus XP.",
				Code:  1005,
			})
			return
		}

		totalXp, err := getTotalXp(db, userId)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch updated XP.",
				Code:  1003,
			})
			return
		}

		writeXpResponse(w, totalXp, false)
	})))

	mux.HandleFunc("/achievements", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		var visitedCount, journalCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM visited_adventures WHERE user_id = $1`, userId).Scan(&visitedCount); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch achievements.",
				Code:  1009,
			})
			return
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM journal_entries WHERE user_id = $1`, userId).Scan(&journalCount); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch achievements.",
				Code:  1009,
			})
			return
		}

		json.NewEncoder(w).Encode(services.ComputeAchievements(visitedCount, journalCount))
	})))

	registerAuthRoutes(mux, db)
}

const journalBonusXp = 25

func getTotalXp(db *sql.DB, userId string) (int, error) {
	var totalXp int
	err := db.QueryRow(`SELECT total_xp FROM users WHERE user_id = $1`, userId).Scan(&totalXp)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return totalXp, err
}

func addXp(db *sql.DB, userId string, xp int) error {
	_, err := db.Exec(
		`INSERT INTO users (user_id, total_xp) VALUES ($1, $2)
		 ON CONFLICT(user_id) DO UPDATE SET total_xp = users.total_xp + excluded.total_xp`,
		userId, xp,
	)
	return err
}

// writeXpResponse encodes the standard XP/level payload every XP-awarding
// endpoint returns, so the frontend HUD can update from any of them the
// same way. alreadyVisited is only meaningful for POST /visited; other
// callers pass false.
func writeXpResponse(w http.ResponseWriter, totalXp int, alreadyVisited bool) {
	json.NewEncoder(w).Encode(map[string]interface{}{
		"totalXp":        totalXp,
		"level":          services.LevelForXp(totalXp),
		"nextLevelXp":    services.NextLevelXp(totalXp),
		"alreadyVisited": alreadyVisited,
	})
}

// postVisited records userId visiting an adventure and awards its XP value.
// The client sends only the adventure ID — XP amount and coordinates are
// looked up server-side so they can't be forged. A unique index on
// (user_id, adventure_id) makes the insert a no-op on repeat visits, so XP
// is only awarded once per adventure per user.
func postVisited(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
	var body struct {
		AdventureId string `json:"adventureId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.AdventureId == "" {
		utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
			Error: "adventureId is required.",
			Code:  1004,
		})
		return
	}

	adventureId, err := strconv.ParseInt(body.AdventureId, 10, 64)
	if err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
			Error:   "Invalid adventureId.",
			Code:    1004,
			Details: "adventureId must be a valid adventure ID.",
		})
		return
	}

	var xpValue sql.NullInt64
	var lat, lng sql.NullFloat64
	err = db.QueryRow(`SELECT xp_value, lat, lng FROM adventures WHERE id = $1`, adventureId).
		Scan(&xpValue, &lat, &lng)
	if err == sql.ErrNoRows {
		utils.WriteJSONError(w, http.StatusNotFound, utils.APIError{
			Error: "Adventure not found.",
			Code:  1001,
		})
		return
	}
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to look up adventure.",
			Code:  1006,
		})
		return
	}

	result, err := db.Exec(
		`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng, created_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (user_id, adventure_id) DO NOTHING`,
		userId, adventureId, lat.Float64, lng.Float64,
	)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to record visited adventure.",
			Code:  1006,
		})
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to record visited adventure.",
			Code:  1006,
		})
		return
	}
	alreadyVisited := rowsAffected == 0

	if !alreadyVisited {
		if err := addXp(db, userId, int(xpValue.Int64)); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to award XP.",
				Code:  1005,
			})
			return
		}
	}

	totalXp, err := getTotalXp(db, userId)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to fetch updated XP.",
			Code:  1003,
		})
		return
	}

	writeXpResponse(w, totalXp, alreadyVisited)
}

func getVisited(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
	rows, err := db.Query(`
		SELECT v.adventure_id, a.name, v.lat, v.lng
		FROM visited_adventures v
		LEFT JOIN adventures a ON a.id = v.adventure_id
		WHERE v.user_id = $1`, userId)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to fetch visited adventures.",
			Code:  1010,
		})
		return
	}
	defer rows.Close()

	visited := []models.VisitedEntry{}
	for rows.Next() {
		var entry models.VisitedEntry
		var adventureId int64
		var name sql.NullString
		if err := rows.Scan(&adventureId, &name, &entry.Lat, &entry.Lng); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch visited adventures.",
				Code:  1010,
			})
			return
		}
		entry.AdventureId = strconv.FormatInt(adventureId, 10)
		entry.Name = name.String
		visited = append(visited, entry)
	}

	json.NewEncoder(w).Encode(visited)
}
