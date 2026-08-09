package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/eugenelacatis/solo-adventure-picker/models"
	"github.com/eugenelacatis/solo-adventure-picker/services"
	"github.com/eugenelacatis/solo-adventure-picker/utils"
)

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
		if origin := r.Header.Get("Origin"); origin != "" {
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

	mux.HandleFunc("/random", withCORS(func(w http.ResponseWriter, r *http.Request) {
		region := r.URL.Query().Get("region")

		query := `SELECT id, name, type, region, scenery, effort, duration, description, xp_value, lat, lng FROM adventures`
		args := []interface{}{}
		if region != "" {
			query += ` WHERE region = ?`
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

		// AI Enhancement: Create a basic user profile and enhance the adventure
		enhanceWithAI(&adv)

		json.NewEncoder(w).Encode(adv)
	}))

	mux.HandleFunc("/xp", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		totalXp, err := getTotalXp(db, userId)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch XP.",
				Code:  1003,
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]int{
			"totalXp": totalXp,
			"level":   services.LevelForXp(totalXp),
		})
	})))

	mux.HandleFunc("/xp/add", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		var body struct {
			AdventureId string  `json:"adventureId"`
			Xp          int     `json:"xp"`
			Lat         float64 `json:"lat"`
			Lng         float64 `json:"lng"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
				Error: "Invalid request body.",
				Code:  1004,
			})
			return
		}

		if err := addXp(db, userId, body.Xp); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to award XP.",
				Code:  1005,
			})
			return
		}

		if body.AdventureId != "" {
			_, err := db.Exec(
				`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng) VALUES (?, ?, ?, ?)`,
				userId, body.AdventureId, body.Lat, body.Lng,
			)
			if err != nil {
				utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
					Error: "Failed to record visited adventure.",
					Code:  1006,
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

		json.NewEncoder(w).Encode(map[string]int{
			"totalXp": totalXp,
			"level":   services.LevelForXp(totalXp),
		})
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

		_, err := db.Exec(
			`INSERT INTO journal_entries (user_id, adventure_id, text) VALUES (?, ?, ?)`,
			userId, body.AdventureId, body.Text,
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

		json.NewEncoder(w).Encode(map[string]int{
			"totalXp": totalXp,
			"level":   services.LevelForXp(totalXp),
		})
	})))

	mux.HandleFunc("/visited", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		rows, err := db.Query(`SELECT adventure_id, lat, lng FROM visited_adventures WHERE user_id = ?`, userId)
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
			if err := rows.Scan(&entry.AdventureId, &entry.Lat, &entry.Lng); err != nil {
				utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
					Error: "Failed to fetch visited adventures.",
					Code:  1010,
				})
				return
			}
			visited = append(visited, entry)
		}

		json.NewEncoder(w).Encode(visited)
	})))

	mux.HandleFunc("/achievements", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		var visitedCount, journalCount int
		if err := db.QueryRow(`SELECT COUNT(*) FROM visited_adventures WHERE user_id = ?`, userId).Scan(&visitedCount); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch achievements.",
				Code:  1009,
			})
			return
		}
		if err := db.QueryRow(`SELECT COUNT(*) FROM journal_entries WHERE user_id = ?`, userId).Scan(&journalCount); err != nil {
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
	err := db.QueryRow(`SELECT total_xp FROM users WHERE user_id = ?`, userId).Scan(&totalXp)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return totalXp, err
}

func addXp(db *sql.DB, userId string, xp int) error {
	_, err := db.Exec(
		`INSERT INTO users (user_id, total_xp) VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET total_xp = total_xp + excluded.total_xp`,
		userId, xp,
	)
	return err
}

// enhanceWithAI uses the adventure agent to personalize the adventure description
func enhanceWithAI(adv *models.Adventure) {
	ctx := context.Background()
	agent, err := services.NewAdventureAgent(ctx)
	if err != nil {
		log.Printf("Failed to create adventure agent: %v", err)
		// Set a default description if AI isn't available
		if adv.Description == "" {
			adv.Description = "Embark on this exciting adventure!"
		}
		// Set default XP value
		if adv.XPValue == 0 {
			adv.XPValue = 100
		}
		return
	}
	defer agent.Close()

	// Create a default user profile for now (later you'll get this from user data)
	userProfile := &services.UserProfile{
		Preferences: []string{"outdoor activities", "exploration", "mindfulness"},
		PastTypes:   []string{},
		EnergyLevel: "medium",
	}

	// Generate a basic description if none exists
	basicDescription := adv.Description
	if basicDescription == "" {
		basicDescription = "A wonderful adventure awaits you!"
	}

	// Enhance the description with AI
	enhanced, err := agent.EnhanceDescription(adv.Name, adv.Type, basicDescription, userProfile)
	if err != nil {
		log.Printf("Failed to enhance adventure description: %v", err)
		adv.Description = basicDescription
	} else {
		adv.Description = enhanced
	}

	// Set XP value based on adventure type (you can make this more sophisticated later)
	if adv.XPValue == 0 {
		switch adv.Type {
		case "hike", "outdoor":
			adv.XPValue = 150
		case "food", "cafe":
			adv.XPValue = 75
		case "culture", "museum":
			adv.XPValue = 125
		default:
			adv.XPValue = 100
		}
	}
}
