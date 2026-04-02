package api

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
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

func RequireConsoleAuth(next http.HandlerFunc, token, consolePassword string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Bearer header still works for scripted access.
		if token != "" && r.Header.Get("Authorization") == "Bearer "+token {
			next(w, r)
			return
		}
		// If no console password configured, console remains open in MVP mode.
		if consolePassword == "" {
			next(w, r)
			return
		}
		c, err := r.Cookie("console_session")
		if err != nil || c.Value != sessionValue(consolePassword) {
			http.Redirect(w, r, "/console/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func sessionValue(password string) string {
	sum := sha256.Sum256([]byte("wsm-console:" + password))
	return hex.EncodeToString(sum[:])
}
