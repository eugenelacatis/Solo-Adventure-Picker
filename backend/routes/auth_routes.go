package routes

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/eugenelacatis/solo-adventure-picker/services"
	"github.com/eugenelacatis/solo-adventure-picker/utils"
)

const sessionCookieName = "session"

func setSessionCookie(w http.ResponseWriter, db *sql.DB, userId string) error {
	token, expiresAt, err := services.CreateSession(db, userId)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// resolveUserFromSession returns the userId bound to the request's session
// cookie, or an error if there is no valid session.
func resolveUserFromSession(db *sql.DB, r *http.Request) (string, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", err
	}
	return services.ResolveSession(db, cookie.Value)
}

// requireAuth resolves the caller's identity from their session cookie and
// hands it to next, instead of letting the caller assert an identity via a
// URL path parameter (the previously-fixed access-control gap).
func requireAuth(db *sql.DB, next func(w http.ResponseWriter, r *http.Request, userId string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userId, err := resolveUserFromSession(db, r)
		if err != nil {
			utils.WriteJSONError(w, http.StatusUnauthorized, utils.APIError{
				Error: "Authentication required.",
				Code:  1011,
			})
			return
		}
		next(w, r, userId)
	}
}

func registerAuthRoutes(mux *http.ServeMux, db *sql.DB) {
	mux.HandleFunc("/auth/signup", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email           string `json:"email"`
			Password        string `json:"password"`
			AnonymousUserId string `json:"anonymousUserId"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Email == "" || body.Password == "" {
			utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
				Error: "Email and password are required.",
				Code:  1012,
			})
			return
		}

		var exists int
		db.QueryRow(`SELECT COUNT(*) FROM users WHERE email = ?`, body.Email).Scan(&exists)
		if exists > 0 {
			utils.WriteJSONError(w, http.StatusConflict, utils.APIError{
				Error: "An account with that email already exists.",
				Code:  1013,
			})
			return
		}

		passwordHash, err := services.HashPassword(body.Password)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to create account.",
				Code:  1014,
			})
			return
		}

		userId := body.AnonymousUserId

		tx, err := db.Begin()
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to create account.",
				Code:  1014,
			})
			return
		}
		defer tx.Rollback()

		_, err = tx.Exec(
			`INSERT INTO users (user_id, email, password_hash) VALUES (?, ?, ?)
			 ON CONFLICT(user_id) DO UPDATE SET email = excluded.email, password_hash = excluded.password_hash`,
			userId, body.Email, passwordHash,
		)
		if err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to create account.",
				Code:  1014,
			})
			return
		}

		if err := tx.Commit(); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to create account.",
				Code:  1014,
			})
			return
		}

		if err := setSessionCookie(w, db, userId); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Account created but failed to start session.",
				Code:  1015,
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"userId": userId, "email": body.Email})
	}))

	mux.HandleFunc("/auth/login", withCORS(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			utils.WriteJSONError(w, http.StatusBadRequest, utils.APIError{
				Error: "Invalid request body.",
				Code:  1016,
			})
			return
		}

		var userId, passwordHash string
		err := db.QueryRow(`SELECT user_id, password_hash FROM users WHERE email = ?`, body.Email).Scan(&userId, &passwordHash)
		if err != nil || services.CheckPassword(passwordHash, body.Password) != nil {
			utils.WriteJSONError(w, http.StatusUnauthorized, utils.APIError{
				Error: "Invalid email or password.",
				Code:  1017,
			})
			return
		}

		if err := setSessionCookie(w, db, userId); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to start session.",
				Code:  1015,
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"userId": userId, "email": body.Email})
	}))

	mux.HandleFunc("/auth/logout", withCORS(func(w http.ResponseWriter, r *http.Request) {
		if cookie, err := r.Cookie(sessionCookieName); err == nil {
			db.Exec(`DELETE FROM sessions WHERE token = ?`, cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))

	mux.HandleFunc("/auth/me", withCORS(requireAuth(db, func(w http.ResponseWriter, r *http.Request, userId string) {
		var email string
		if err := db.QueryRow(`SELECT email FROM users WHERE user_id = ?`, userId).Scan(&email); err != nil {
			utils.WriteJSONError(w, http.StatusInternalServerError, utils.APIError{
				Error: "Failed to fetch account.",
				Code:  1018,
			})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"userId": userId, "email": email})
	})))
}
