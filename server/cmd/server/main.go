package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"wageslavemonitor/server/internal/api"
	"wageslavemonitor/server/internal/config"
	"wageslavemonitor/server/internal/store"
)

func main() {
	cfgPath := getenv("CONFIG_PATH", "./config/config.json")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	addr := cfg.Addr
	dataDir := cfg.DataDir
	dbPath := cfg.DBPath
	authToken := cfg.AuthToken
	defaultInterval := cfg.DefaultCaptureIntervalSeconds
	retentionDays := cfg.RetentionDays
	consoleAuthDisabled := cfg.ConsoleAuthDisabled

	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := db.Init(context.Background()); err != nil {
		log.Fatal(err)
	}

	fs := store.NewFileStore(dataDir)
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/api/v1/clients", api.RequireBearer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		clients, err := db.ListClients(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		apiWriteJSON(w, http.StatusOK, map[string]any{"clients": clients})
	}, authToken))

	mux.HandleFunc("/api/v1/clients/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/clients/")
		if strings.HasSuffix(path, "/config") {
			api.RequireBearer(api.ConfigHandler(db, defaultInterval), authToken)(w, r)
			return
		}
		if strings.HasSuffix(path, "/screenshots") {
			if r.Method == http.MethodPost {
				api.RequireBearer(api.IngestHandler(db, fs), authToken)(w, r)
				return
			}
			if r.Method == http.MethodGet {
				api.RequireBearer(func(w http.ResponseWriter, r *http.Request) {
					clientID := strings.TrimSuffix(path, "/screenshots")
					clientID = strings.Trim(clientID, "/")
					page := 1
					if p := r.URL.Query().Get("page"); p != "" {
						if pv, err := strconv.Atoi(p); err == nil && pv > 0 {
							page = pv
						}
					}
					limit := 50
					offset := (page - 1) * limit
					shots, err := db.ListScreenshots(r.Context(), clientID, limit, offset)
					if err != nil {
						http.Error(w, err.Error(), http.StatusInternalServerError)
						return
					}
					apiWriteJSON(w, http.StatusOK, map[string]any{"screenshots": shots, "page": page})
				}, authToken)(w, r)
				return
			}
		}
		http.NotFound(w, r)
	})

	console := &api.ConsoleHandler{
		DB:          db,
		DataDir:     dataDir,
		TemplateDir: "./web/templates",
	}
	if err := console.Register(mux, authToken, consoleAuthDisabled); err != nil {
		log.Fatal(err)
	}
	mux.Handle("/data/", api.RequireConsoleAuth(
		http.StripPrefix("/data/", http.FileServer(http.Dir(dataDir))).ServeHTTP,
		authToken,
		db,
		consoleAuthDisabled,
	))

	go runRetentionJob(db, fs, retentionDays)

	log.Printf("server listening on %s", addr)
	if err := http.ListenAndServe(addr, loggingMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func runRetentionJob(db *store.SQLiteStore, fs *store.FileStore, retentionDays int) {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		before := time.Now().UTC().Add(-time.Duration(retentionDays) * 24 * time.Hour)
		paths, err := db.DeleteOlderThan(context.Background(), before)
		if err != nil {
			log.Printf("retention delete metadata error: %v", err)
			continue
		}
		for _, p := range paths {
			if err := fs.Delete(p); err != nil && !os.IsNotExist(err) {
				log.Printf("retention delete file error (%s): %v", p, err)
			}
		}
	}
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func apiWriteJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
