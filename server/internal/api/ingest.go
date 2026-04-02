package api

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"wageslavemonitor/server/internal/store"
)

func IngestHandler(db *store.SQLiteStore, fs *store.FileStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		clientID := strings.TrimPrefix(r.URL.Path, "/api/v1/clients/")
		clientID = strings.TrimSuffix(clientID, "/screenshots")
		clientID = strings.Trim(clientID, "/")
		if clientID == "" {
			writeError(w, http.StatusBadRequest, "missing client id")
			return
		}
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeError(w, http.StatusBadRequest, "multipart parse error")
			return
		}
		monitorIndex, _ := strconv.Atoi(r.FormValue("monitor_index"))
		capturedAt, err := time.Parse(time.RFC3339, r.FormValue("captured_at"))
		if err != nil {
			capturedAt = time.Now().UTC()
		}
		file, _, err := r.FormFile("image")
		if err != nil {
			writeError(w, http.StatusBadRequest, "image is required")
			return
		}
		defer file.Close()
		rel, sha, size, err := fs.Save(clientID, capturedAt, monitorIndex, file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if err := db.InsertScreenshot(r.Context(), store.ScreenshotMeta{
			ClientID:     clientID,
			CapturedAt:   capturedAt.UTC(),
			MonitorIndex: monitorIndex,
			RelativePath: rel,
			SizeBytes:    size,
			SHA256:       sha,
			UploadedAt:   time.Now().UTC(),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "path": rel, "sha256": sha, "bytes": size})
	}
}
