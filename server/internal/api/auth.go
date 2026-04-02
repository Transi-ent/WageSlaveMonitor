package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"

	"wageslavemonitor/server/internal/store"
)

func RequireBearer(next http.HandlerFunc, token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token == "" {
			next(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer "+token {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		next(w, r)
	}
}

// ConsoleSessionCookie derives a stable cookie value from the stored bcrypt hash so we never
// put plaintext passwords in cookies; changing the password changes the hash and invalidates old sessions.
func ConsoleSessionCookie(passwordBcrypt string) string {
	sum := sha256.Sum256([]byte("wsm-console-cookie:" + passwordBcrypt))
	return hex.EncodeToString(sum[:])
}

func SetConsoleSessionCookie(w http.ResponseWriter, passwordBcrypt string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "console_session",
		Value:    ConsoleSessionCookie(passwordBcrypt),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
}

func RequireConsoleAuth(next http.HandlerFunc, token string, db *store.SQLiteStore, consoleAuthDisabled bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if consoleAuthDisabled {
			next(w, r)
			return
		}
		if token != "" && r.Header.Get("Authorization") == "Bearer "+token {
			next(w, r)
			return
		}
		hashStr, err := db.GetConsolePasswordBcrypt(r.Context())
		if err != nil || hashStr == "" {
			http.Redirect(w, r, "/console/login", http.StatusSeeOther)
			return
		}
		want := ConsoleSessionCookie(hashStr)
		c, err := r.Cookie("console_session")
		if err != nil || c.Value != want {
			http.Redirect(w, r, "/console/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
