package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
		allowed, _, err := consumeReroll(db, userId)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to check reroll allowance.",
				Code:  1012,
			})
			return
		}
		if !allowed {
			utils.WriteJSONError(w, http.StatusTooManyRequests, utils.APIError{
				Error:   "Daily reroll limit reached.",
				Code:    1013,
				Details: "You're out of rerolls for today. Come back tomorrow, or earn more.",
			})
			return
		}

		region := r.URL.Query().Get("region")
		typeFilter := r.URL.Query().Get("type")
		effortFilter := r.URL.Query().Get("effort")

		query := `SELECT id, name, type, region, scenery, effort, duration, description, xp_value, lat, lng FROM adventures`
		conditions := []string{}
		args := []interface{}{}
		if region != "" {
			args = append(args, region)
			conditions = append(conditions, fmt.Sprintf("region = $%d", len(args)))
		}
		if typeFilter != "" {
			args = append(args, typeFilter)
			conditions = append(conditions, fmt.Sprintf("type = $%d", len(args)))
		}
		if effortFilter != "" {
			args = append(args, effortFilter)
			conditions = append(conditions, fmt.Sprintf("effort = $%d", len(args)))
		}
		if len(conditions) > 0 {
			query += ` WHERE ` + strings.Join(conditions, " AND ")
		}
		query += ` ORDER BY RANDOM() LIMIT 1`

		row := db.QueryRow(query, args...)

		var adv models.Adventure
		var advType, scenery, effort, duration, description sql.NullString
		var xpValue sql.NullInt64
		var lat, lng sql.NullFloat64

		err = row.Scan(&adv.ID, &adv.Name, &advType, &adv.Region, &scenery, &effort, &duration, &description, &xpValue, &lat, &lng)
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

	mux.HandleFunc("/reroll-status", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		tokens, resetAt, err := getRerollStatus(db, userId)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch reroll status.",
				Code:  1014,
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"rerollTokens":  tokens,
			"rerollResetAt": resetAt,
		})
	})))

	mux.HandleFunc("/quest", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		completedToday, err := getQuestStatus(db, userId)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch quest status.",
				Code:  1018,
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"description":    "Do 1 new thing today",
			"completedToday": completedToday,
		})
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
		switch r.Method {
		case http.MethodPost:
			postJournal(w, r, db, userId)
		default:
			getJournal(w, r, db, userId)
		}
	})))

	mux.HandleFunc("/trail", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		switch r.Method {
		case http.MethodPost:
			postTrail(w, r, db, userId)
		default:
			getTrail(w, r, db, userId)
		}
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

		achievements := services.ComputeAchievements(visitedCount, journalCount)
		newlyUnlocked, err := persistNewAchievements(db, userId, achievements)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch achievements.",
				Code:  1009,
			})
			return
		}
		if len(newlyUnlocked) > 0 {
			if err := creditRerollTokens(db, userId, len(newlyUnlocked)*achievementRerollTokens); err != nil {
				utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
					Error: "Failed to award achievement reroll tokens.",
					Code:  1016,
				})
				return
			}
		}

		json.NewEncoder(w).Encode(achievements)
	})))

	registerAuthRoutes(mux, db)
}

const journalBonusXp = 25
const journalBonusRerollTokens = 1
const achievementRerollTokens = 2
const dailyRerollAllowance = 5

// consumeReroll atomically resets userId's reroll tokens if their reset
// window has passed, then spends one token if any are available. The reset
// and spend happen in a single UPDATE (via a CTE computing the post-reset
// balance once) so concurrent requests can't both read a stale token count
// and double-spend the last one. allowed reports whether a token was
// actually spent; remaining is the token count after this call either way.
func consumeReroll(db *sql.DB, userId string) (allowed bool, remaining int, err error) {
	_, err = db.Exec(
		`INSERT INTO users (user_id, reroll_tokens, reroll_reset_at) VALUES ($1, $2, now() + interval '1 day')
		 ON CONFLICT (user_id) DO NOTHING`,
		userId, dailyRerollAllowance,
	)
	if err != nil {
		return false, 0, err
	}

	err = db.QueryRow(`
		WITH current AS (
			SELECT
				CASE WHEN reroll_reset_at <= now() THEN $2 ELSE reroll_tokens END AS balance,
				CASE WHEN reroll_reset_at <= now() THEN now() + interval '1 day' ELSE reroll_reset_at END AS reset_at
			FROM users WHERE user_id = $1
		)
		UPDATE users
		SET
			reroll_tokens = GREATEST(current.balance - 1, 0),
			reroll_reset_at = current.reset_at
		FROM current
		WHERE users.user_id = $1
		RETURNING users.reroll_tokens, current.balance > 0`,
		userId, dailyRerollAllowance,
	).Scan(&remaining, &allowed)
	if err != nil {
		return false, 0, err
	}

	return allowed, remaining, nil
}

// getRerollStatus reports userId's current reroll token balance and reset
// time without spending a token or writing to the database. A user with no
// row yet (never called /random) is reported as having the full daily
// allowance, matching what consumeReroll would give them on their first call.
func getRerollStatus(db *sql.DB, userId string) (tokens int, resetAt time.Time, err error) {
	err = db.QueryRow(`
		SELECT
			CASE WHEN reroll_reset_at <= now() THEN $2 ELSE reroll_tokens END,
			CASE WHEN reroll_reset_at <= now() THEN now() + interval '1 day' ELSE reroll_reset_at END
		FROM users WHERE user_id = $1`,
		userId, dailyRerollAllowance,
	).Scan(&tokens, &resetAt)
	if err == sql.ErrNoRows {
		return dailyRerollAllowance, time.Now(), nil
	}
	return tokens, resetAt, err
}

// persistNewAchievements records any of the given (currently-unlocked)
// achievements that userId doesn't already have a user_achievements row
// for, and returns just the ones that were newly inserted this call. The
// achievements themselves stay computed on demand (services.ComputeAchievements
// is still the source of truth for what's unlocked); this only tracks
// *when* each one was first seen, so reward logic (e.g. reroll tokens) can
// tell a first-time unlock from a re-computation on every page load.
func persistNewAchievements(db *sql.DB, userId string, achievements []services.Achievement) ([]services.Achievement, error) {
	newlyUnlocked := []services.Achievement{}
	for _, a := range achievements {
		res, err := db.Exec(
			`INSERT INTO user_achievements (user_id, achievement_id) VALUES ($1, $2)
			 ON CONFLICT (user_id, achievement_id) DO NOTHING`,
			userId, a.Id,
		)
		if err != nil {
			return nil, err
		}
		rowsAffected, err := res.RowsAffected()
		if err != nil {
			return nil, err
		}
		if rowsAffected > 0 {
			newlyUnlocked = append(newlyUnlocked, a)
		}
	}
	return newlyUnlocked, nil
}

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

// creditRerollTokens adds amount to userId's reroll token balance, as a
// reward for journaling or unlocking an achievement (on top of whatever
// their daily allowance from consumeReroll already gave them). If userId
// has no row yet, one is created at the schema default (dailyRerollAllowance)
// first, so the UPDATE below always has a row to add amount onto.
func creditRerollTokens(db *sql.DB, userId string, amount int) error {
	if _, err := db.Exec(
		`INSERT INTO users (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userId,
	); err != nil {
		return err
	}
	_, err := db.Exec(
		`UPDATE users SET reroll_tokens = reroll_tokens + $2 WHERE user_id = $1`,
		userId, amount,
	)
	return err
}

const dailyQuestRerollTokens = 3

// completeDailyQuestIfFirst marks userId's daily quest ("do 1 new thing")
// complete and awards a reroll token bonus, but only the first time this is
// called for a given calendar date — the UPDATE only touches rows where
// last_quest_completed_at isn't already today, and RETURNING tells the
// caller whether this call was the one that flipped it. Both postVisited
// and postJournal call this, so whichever the user does first each day
// completes the quest.
func completeDailyQuestIfFirst(db *sql.DB, userId string) (justCompleted bool, err error) {
	if _, err := db.Exec(
		`INSERT INTO users (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING`,
		userId,
	); err != nil {
		return false, err
	}

	res, err := db.Exec(
		`UPDATE users SET last_quest_completed_at = CURRENT_DATE
		 WHERE user_id = $1
		   AND last_quest_completed_at IS DISTINCT FROM CURRENT_DATE`,
		userId,
	)
	if err != nil {
		return false, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	if rowsAffected == 0 {
		return false, nil
	}

	if err := creditRerollTokens(db, userId, dailyQuestRerollTokens); err != nil {
		return false, err
	}
	return true, nil
}

// getQuestStatus reports whether userId has completed today's daily quest.
func getQuestStatus(db *sql.DB, userId string) (completedToday bool, err error) {
	err = db.QueryRow(
		`SELECT last_quest_completed_at = CURRENT_DATE FROM users WHERE user_id = $1`,
		userId,
	).Scan(&completedToday)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return completedToday, err
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

// postJournal saves a journal entry for userId and awards the flat journal
// bonus XP.
func postJournal(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
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

	if err := creditRerollTokens(db, userId, journalBonusRerollTokens); err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to award journal bonus reroll token.",
			Code:  1015,
		})
		return
	}

	if _, err := completeDailyQuestIfFirst(db, userId); err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to update daily quest.",
			Code:  1017,
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
}

// awardVisit records userId visiting adventureId at (lat, lng) and awards
// xpValue XP, unless they've already visited it (the unique index on
// (user_id, adventure_id) makes the insert a no-op on repeat visits, so XP
// is only ever awarded once per adventure per user). Shared by postVisited
// (manual visit) and the trail proximity-match path (POST /trail) so XP
// math isn't duplicated between the two ways a visit can be recorded.
func awardVisit(db *sql.DB, userId string, adventureId int64, lat, lng float64, xpValue int) (alreadyVisited bool, err error) {
	result, err := db.Exec(
		`INSERT INTO visited_adventures (user_id, adventure_id, lat, lng, created_at)
		 VALUES ($1, $2, $3, $4, now())
		 ON CONFLICT (user_id, adventure_id) DO NOTHING`,
		userId, adventureId, lat, lng,
	)
	if err != nil {
		return false, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	alreadyVisited = rowsAffected == 0

	if !alreadyVisited {
		if err := addXp(db, userId, xpValue); err != nil {
			return false, err
		}
		if _, err := completeDailyQuestIfFirst(db, userId); err != nil {
			return false, err
		}
	}

	return alreadyVisited, nil
}

// trailMatchRadiusMiles is how close a trail point must be to an adventure
// to auto-mark it visited — ~150ft, deliberately tight so driving past on a
// nearby road doesn't silently complete a hike. Separate from any
// dashboard fog-reveal visual radius, which can be more generous.
const trailMatchRadiusMiles = 150.0 / 5280.0

// postTrail stores userId's uploaded trail points and, for each one, checks
// every adventure for proximity within trailMatchRadiusMiles. Any adventure
// close enough that userId hasn't already visited is marked visited via the
// same awardVisit path postVisited uses, so XP math isn't duplicated.
func postTrail(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
	var body struct {
		Points []models.TrailPoint `json:"points"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
			Error: "Invalid trail points payload.",
			Code:  1019,
		})
		return
	}

	for _, p := range body.Points {
		if _, err := db.Exec(
			`INSERT INTO trail_points (user_id, lat, lng, recorded_at) VALUES ($1, $2, $3, $4)`,
			userId, p.Lat, p.Lng, p.RecordedAt,
		); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to save trail points.",
				Code:  1020,
			})
			return
		}
	}

	rows, err := db.Query(`SELECT id, name, type, region, scenery, effort, duration, description, xp_value, lat, lng FROM adventures`)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to check trail proximity.",
			Code:  1021,
		})
		return
	}
	type candidate struct {
		adv models.Adventure
	}
	var candidates []candidate
	for rows.Next() {
		var adv models.Adventure
		var advType, scenery, effort, duration, description sql.NullString
		var xpValue sql.NullInt64
		var lat, lng sql.NullFloat64
		if err := rows.Scan(&adv.ID, &adv.Name, &advType, &adv.Region, &scenery, &effort, &duration, &description, &xpValue, &lat, &lng); err != nil {
			rows.Close()
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to check trail proximity.",
				Code:  1021,
			})
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
		candidates = append(candidates, candidate{adv: adv})
	}
	rows.Close()

	newlyVisited := []models.Adventure{}
	matched := map[int64]bool{}
	for _, p := range body.Points {
		for _, c := range candidates {
			if matched[c.adv.ID] {
				continue
			}
			if services.HaversineMiles(p.Lat, p.Lng, c.adv.Lat, c.adv.Lng) > trailMatchRadiusMiles {
				continue
			}
			alreadyVisited, err := awardVisit(db, userId, c.adv.ID, c.adv.Lat, c.adv.Lng, c.adv.XPValue)
			if err != nil {
				utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
					Error: "Failed to award trail visit.",
					Code:  1022,
				})
				return
			}
			matched[c.adv.ID] = true
			if !alreadyVisited {
				newlyVisited = append(newlyVisited, c.adv)
			}
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"newlyVisited": newlyVisited,
	})
}

// getTrail returns userId's uploaded trail points in chronological capture
// order (by recorded_at, not upload order) so the dashboard can render the
// path correctly even if batches arrived late or out of order.
func getTrail(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
	rows, err := db.Query(
		`SELECT lat, lng, recorded_at FROM trail_points WHERE user_id = $1 ORDER BY recorded_at ASC`,
		userId,
	)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to fetch trail points.",
			Code:  1023,
		})
		return
	}
	defer rows.Close()

	points := []models.TrailPoint{}
	for rows.Next() {
		var p models.TrailPoint
		var recordedAt time.Time
		if err := rows.Scan(&p.Lat, &p.Lng, &recordedAt); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch trail points.",
				Code:  1023,
			})
			return
		}
		p.RecordedAt = recordedAt.Format(time.RFC3339)
		points = append(points, p)
	}

	json.NewEncoder(w).Encode(points)
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

	alreadyVisited, err := awardVisit(db, userId, adventureId, lat.Float64, lng.Float64, int(xpValue.Int64))
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to record visited adventure.",
			Code:  1006,
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

	writeXpResponse(w, totalXp, alreadyVisited)
}

func getVisited(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
	rows, err := db.Query(`
		SELECT v.adventure_id, a.name, a.region, v.lat, v.lng, v.created_at
		FROM visited_adventures v
		LEFT JOIN adventures a ON a.id = v.adventure_id
		WHERE v.user_id = $1
		ORDER BY v.created_at DESC`, userId)
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
		var name, region sql.NullString
		if err := rows.Scan(&adventureId, &name, &region, &entry.Lat, &entry.Lng, &entry.CreatedAt); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch visited adventures.",
				Code:  1010,
			})
			return
		}
		entry.AdventureId = strconv.FormatInt(adventureId, 10)
		entry.Name = name.String
		entry.Region = region.String
		visited = append(visited, entry)
	}

	json.NewEncoder(w).Encode(visited)
}

// getJournal returns the requesting user's journal entries, newest first,
// with the adventure name joined in the same way getVisited joins it.
func getJournal(w http.ResponseWriter, r *http.Request, db *sql.DB, userId string) {
	rows, err := db.Query(`
		SELECT j.adventure_id, a.name, j.text, j.created_at
		FROM journal_entries j
		LEFT JOIN adventures a ON a.id = j.adventure_id
		WHERE j.user_id = $1
		ORDER BY j.created_at DESC`, userId)
	if err != nil {
		utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
			Error: "Failed to fetch journal entries.",
			Code:  1011,
		})
		return
	}
	defer rows.Close()

	entries := []models.JournalEntryView{}
	for rows.Next() {
		var entry models.JournalEntryView
		var adventureId int64
		var name sql.NullString
		if err := rows.Scan(&adventureId, &name, &entry.Text, &entry.CreatedAt); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch journal entries.",
				Code:  1011,
			})
			return
		}
		entry.AdventureId = strconv.FormatInt(adventureId, 10)
		entry.AdventureName = name.String
		entries = append(entries, entry)
	}

	json.NewEncoder(w).Encode(entries)
}
