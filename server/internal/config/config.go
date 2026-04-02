package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Config represents server runtime configuration.
// All settings should be read from the config file in MVP.
type Config struct {
	Addr string `json:"ADDR"`

	DataDir string `json:"DATA_DIR"`
	DBPath  string `json:"DB_PATH"`

	AuthToken string `json:"AUTH_TOKEN"`

	DefaultCaptureIntervalSeconds int `json:"DEFAULT_CAPTURE_INTERVAL_SECONDS"`
	RetentionDays                 int `json:"RETENTION_DAYS"`

	// If true, console login is bypassed (useful for debugging / initial bring-up).
	ConsoleAuthDisabled bool `json:"CONSOLE_AUTH_DISABLED"`
}

func DefaultConfig() Config {
	return Config{
		Addr:                          ":8080",
		DataDir:                       "./data",
		DBPath:                        "./data/meta.db",
		AuthToken:                     "",
		DefaultCaptureIntervalSeconds: 30,
		RetentionDays:                 14,
		ConsoleAuthDisabled:           true,
	}
}

func Load(path string) (Config, error) {
	if path == "" {
		return Config{}, errors.New("config path is empty")
	}
	path = filepath.Clean(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	// Start with defaults so that unset fields keep default values.
	cfg := DefaultConfig()
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}
	// Fill in missing defaults for fields that might still be zero.
	def := DefaultConfig()
	if cfg.Addr == "" {
		cfg.Addr = def.Addr
	}
	if cfg.DataDir == "" {
		cfg.DataDir = def.DataDir
	}
	if cfg.DBPath == "" {
		cfg.DBPath = def.DBPath
	}
	if cfg.DefaultCaptureIntervalSeconds <= 0 {
		cfg.DefaultCaptureIntervalSeconds = def.DefaultCaptureIntervalSeconds
	}
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = def.RetentionDays
	}
	// ConsoleAuthDisabled already defaults to true from DefaultConfig().
	// If config file explicitly sets it to false, json.Unmarshal will override.
	return cfg, nil
}
