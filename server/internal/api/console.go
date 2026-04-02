package api

import (
	"context"
	"html/template"
	"net/http"
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

func (h *ConsoleHandler) Register(mux *http.ServeMux, token string, consoleAuthDisabled bool) error {
	// IMPORTANT:
	// Each page file defines a template named "content". If we parse all pages into one
	// template set, the last parsed "content" wins and every page renders the same content.
	// To avoid this, we parse a base layout, then CLONE + parse the page file per page.
	base, err := template.ParseFiles(filepath.Join(h.TemplateDir, "layout.html"))
	if err != nil {
		return err
	}

	type pageTemplates struct {
		login          *template.Template
		changePassword *template.Template
		clients        *template.Template
		clientDetail   *template.Template
	}

	buildPage := func(pageFile string) (*template.Template, error) {
		t, err := base.Clone()
		if err != nil {
			return nil, err
		}
		_, err = t.ParseFiles(filepath.Join(h.TemplateDir, pageFile))
		if err != nil {
			return nil, err
		}
		return t, nil
	}

	pages := pageTemplates{}
	if pages.login, err = buildPage("login.html"); err != nil {
		return err
	}
	if pages.changePassword, err = buildPage("change_password.html"); err != nil {
		return err
	}
	if pages.clients, err = buildPage("clients.html"); err != nil {
		return err
	}
	// client detail needs layout + client_detail template
	// (still uses "content" name, so it must be isolated too).
	if pages.clientDetail, err = buildPage("client_detail.html"); err != nil {
		return err
	}

	mux.HandleFunc("/console/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = pages.login.ExecuteTemplate(w, "login", map[string]any{"ShowNav": false})
			return
		}
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			// Accept both the current template field name and potential legacy ones.
			pwd := firstNonEmptyFormValue(r, "password", "pwd", "pass")
			ok, err := h.DB.VerifyConsolePassword(r.Context(), pwd)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if ok {
				hashStr, err := h.DB.GetConsolePasswordBcrypt(r.Context())
				if err != nil {
					writeError(w, http.StatusInternalServerError, err.Error())
					return
				}
				SetConsoleSessionCookie(w, hashStr)
				http.Redirect(w, r, "/console/clients", http.StatusSeeOther)
				return
			}
			_ = pages.login.ExecuteTemplate(w, "login", map[string]any{"ShowNav": false, "Error": "invalid password"})
			return
		}
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	})

	mux.HandleFunc("/console/change-password", RequireConsoleAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = pages.changePassword.ExecuteTemplate(w, "change_password", map[string]any{"ShowNav": true})
		case http.MethodPost:
			_ = r.ParseForm()
			// Accept both the current template field names and potential legacy ones.
			current := firstNonEmptyFormValue(r, "current_password", "currentPassword")
			newPwd := firstNonEmptyFormValue(r, "new_password", "newPassword")
			confirm := firstNonEmptyFormValue(r, "confirm_password", "confirmPassword", "confirm")
			if newPwd != confirm {
				_ = pages.changePassword.ExecuteTemplate(w, "change_password", map[string]any{"ShowNav": true, "Error": "new passwords do not match"})
				return
			}
			ok, err := h.DB.VerifyConsolePassword(r.Context(), current)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if !ok {
				_ = pages.changePassword.ExecuteTemplate(w, "change_password", map[string]any{"ShowNav": true, "Error": "current password is incorrect"})
				return
			}
			if err := h.DB.SetConsolePassword(r.Context(), newPwd); err != nil {
				_ = pages.changePassword.ExecuteTemplate(w, "change_password", map[string]any{"ShowNav": true, "Error": err.Error()})
				return
			}
			hashStr, err := h.DB.GetConsolePasswordBcrypt(r.Context())
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			SetConsoleSessionCookie(w, hashStr)
			http.Redirect(w, r, "/console/clients", http.StatusSeeOther)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
	}, token, h.DB, consoleAuthDisabled))

	mux.HandleFunc("/console/clients", RequireConsoleAuth(func(w http.ResponseWriter, r *http.Request) {
		clients, err := h.DB.ListClients(r.Context())
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_ = pages.clients.ExecuteTemplate(w, "clients", map[string]any{"Clients": clients, "ShowNav": true})
	}, token, h.DB, consoleAuthDisabled))

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
		_ = pages.clientDetail.ExecuteTemplate(w, "client_detail", map[string]any{
			"ClientID":    clientID,
			"Screenshots": shots,
			"Page":        page,
			"ShowNav":     true,
		})
	}, token, h.DB, consoleAuthDisabled))

	return nil
}

func firstNonEmptyFormValue(r *http.Request, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(r.FormValue(k)); v != "" {
			return v
		}
	}
	return ""
}
