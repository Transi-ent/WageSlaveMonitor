package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"wageslavemonitor/server/internal/store"
)

func ConfigHandler(db *store.SQLiteStore, defaultInterval int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/clients/")
		clientID = strings.TrimSuffix(clientID, "/config")
		clientID = strings.Trim(clientID, "/")
		if clientID == "" {
			writeError(w, http.StatusBadRequest, "missing client id")
			return
		}
		switch r.Method {
		case http.MethodGet:
			cfg, err := db.GetClientConfig(r.Context(), clientID, defaultInterval)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"client_id": cfg.ClientID, "capture_interval_seconds": cfg.CaptureInterval, "updated_at": cfg.UpdatedAt})
		case http.MethodPut:
			var payload struct {
				CaptureIntervalSeconds int `json:"capture_interval_seconds"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeError(w, http.StatusBadRequest, "invalid json")
				return
			}
			if payload.CaptureIntervalSeconds < 5 {
				writeError(w, http.StatusBadRequest, "interval must be >= 5")
				return
			}
			err := db.UpsertClientConfig(r.Context(), store.ClientConfig{
				ClientID:        clientID,
				CaptureInterval: payload.CaptureIntervalSeconds,
				UpdatedAt:       time.Now().UTC(),
			})
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}
}
