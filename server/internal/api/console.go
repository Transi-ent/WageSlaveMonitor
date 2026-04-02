package api

import (
	"context"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"wageslavemonitor/server/internal/store"
)

type ConsoleHandler struct {
	DB          *store.SQLiteStore
	DataDir     string
	TemplateDir string
}

func (h *ConsoleHandler) Register(mux *http.ServeMux, token string) error {
	tpl, err := template.ParseFiles(
		filepath.Join(h.TemplateDir, "layout.html"),
		filepath.Join(h.TemplateDir, "clients.html"),
		filepath.Join(h.TemplateDir, "client_detail.html"),
		filepath.Join(h.TemplateDir, "login.html"),
	)
	if err != nil {
		return err
	}

	consolePassword := getenv("CONSOLE_PASSWORD", "")
	mux.HandleFunc("/console/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = tpl.ExecuteTemplate(w, "login", nil)
			return
		}
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			pwd := r.FormValue("password")
			if consolePassword != "" && pwd == consolePassword {
				http.SetCookie(w, &http.Cookie{
					Name:     "console_session",
					Value:    sessionValue(consolePassword),
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, "/console/clients", http.StatusSeeOther)
				return
			}
			_ = tpl.ExecuteTemplate(w, "login", map[string]any{"Error": "invalid password"})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	mux.HandleFunc("/console/clients", RequireConsoleAuth(func(w http.ResponseWriter, r *http.Request) {
		clients, err := h.DB.ListClients(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = tpl.ExecuteTemplate(w, "clients", map[string]any{"Clients": clients})
	}, token, consolePassword))

	mux.HandleFunc("/console/clients/", RequireConsoleAuth(func(w http.ResponseWriter, r *http.Request) {
		clientID := strings.TrimPrefix(r.URL.Path, "/console/clients/")
		clientID = strings.Trim(clientID, "/")
		if strings.HasSuffix(clientID, "/config") && r.Method == http.MethodPost {
			id := strings.TrimSuffix(clientID, "/config")
			id = strings.Trim(id, "/")
			_ = r.ParseForm()
			interval, _ := strconv.Atoi(r.FormValue("capture_interval_seconds"))
			if interval >= 5 {
				_ = h.DB.UpsertClientConfig(context.Background(), store.ClientConfig{
					ClientID:        id,
					CaptureInterval: interval,
					UpdatedAt:       time.Now().UTC(),
				})
			}
			http.Redirect(w, r, "/console/clients/"+id, http.StatusSeeOther)
			return
		}
		if clientID == "" {
			writeError(w, http.StatusBadRequest, "missing client id")
			return
		}
		page := 1
		if p := r.URL.Query().Get("page"); p != "" {
			if v, err := strconv.Atoi(p); err == nil && v > 0 {
				page = v
			}
		}
		limit := 50
		offset := (page - 1) * limit
		shots, err := h.DB.ListScreenshots(r.Context(), clientID, limit, offset)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = tpl.ExecuteTemplate(w, "client_detail", map[string]any{
			"ClientID":    clientID,
			"Screenshots": shots,
			"Page":        page,
		})
	}, token, consolePassword))

	return nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
