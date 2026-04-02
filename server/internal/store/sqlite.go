package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const DefaultConsolePassword = "123456"

const (
	consolePasswordBcryptKey     = "console_password_bcrypt"
	consolePasswordCustomizedKey = "console_password_customized"
)

type ClientConfig struct {
	ClientID        string
	CaptureInterval int
	UpdatedAt       time.Time
}

type ScreenshotMeta struct {
	ID           int64
	ClientID     string
	CapturedAt   time.Time
	MonitorIndex int
	RelativePath string
	SizeBytes    int64
	SHA256       string
	UploadedAt   time.Time
}

type ClientSummary struct {
	ClientID         string
	LastSeenAt       time.Time
	LastCapturedAt   time.Time
	ScreenshotCount  int64
	CaptureIntervalS int
}

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) Init(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS client_configs (
    client_id TEXT PRIMARY KEY,
    capture_interval_seconds INTEGER NOT NULL DEFAULT 30,
    updated_at DATETIME NOT NULL
);
CREATE TABLE IF NOT EXISTS screenshots (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    client_id TEXT NOT NULL,
    captured_at DATETIME NOT NULL,
    monitor_index INTEGER NOT NULL,
    relative_path TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    sha256 TEXT NOT NULL,
    uploaded_at DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_screenshots_client_time ON screenshots(client_id, captured_at DESC);
CREATE TABLE IF NOT EXISTS server_settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
`
	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return err
	}
	return s.ensureDefaultConsolePassword(ctx)
}

func (s *SQLiteStore) ensureDefaultConsolePassword(ctx context.Context) error {
	// 1) Ensure a bcrypt hash exists.
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM server_settings WHERE key = ?`, consolePasswordBcryptKey).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		if err := s.setConsolePasswordBcrypt(ctx, DefaultConsolePassword); err != nil {
			return err
		}
	}

	// 2) If the user has explicitly customized the password, do not overwrite.
	var customizedCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM server_settings WHERE key = ?`, consolePasswordCustomizedKey).Scan(&customizedCount); err != nil {
		return err
	}
	if customizedCount > 0 {
		return nil
	}

	// 3) If not customized and the stored hash doesn't match the default password, reset to default.
	ok, err := s.VerifyConsolePassword(ctx, DefaultConsolePassword)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	return s.setConsolePasswordBcrypt(ctx, DefaultConsolePassword)
}

func (s *SQLiteStore) GetConsolePasswordBcrypt(ctx context.Context) (string, error) {
	var v string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM server_settings WHERE key = ?`, consolePasswordBcryptKey).Scan(&v)
	return v, err
}

func (s *SQLiteStore) VerifyConsolePassword(ctx context.Context, plaintext string) (bool, error) {
	hashStr, err := s.GetConsolePasswordBcrypt(ctx)
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hashStr), []byte(plaintext)); err != nil {
		return false, nil
	}
	return true, nil
}

func (s *SQLiteStore) SetConsolePassword(ctx context.Context, plaintext string) error {
	if len(plaintext) < 6 {
		return errors.New("password must be at least 6 characters")
	}
	if err := s.setConsolePasswordBcrypt(ctx, plaintext); err != nil {
		return err
	}
	// Mark as customized so Init will keep the user-chosen password.
	_, err := s.db.ExecContext(ctx, `
INSERT INTO server_settings(key, value) VALUES(?, '1')
ON CONFLICT(key) DO UPDATE SET value='1'
`, consolePasswordCustomizedKey)
	return err
}

func (s *SQLiteStore) setConsolePasswordBcrypt(ctx context.Context, plaintext string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO server_settings(key, value) VALUES(?, ?)
ON CONFLICT(key) DO UPDATE SET value = excluded.value
`, consolePasswordBcryptKey, string(hash))
	return err
}

func (s *SQLiteStore) UpsertClientConfig(ctx context.Context, cfg ClientConfig) error {
	if cfg.CaptureInterval <= 0 {
		return errors.New("capture interval must be > 0")
	}
	if cfg.UpdatedAt.IsZero() {
		cfg.UpdatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO client_configs(client_id, capture_interval_seconds, updated_at)
VALUES(?, ?, ?)
ON CONFLICT(client_id) DO UPDATE SET
capture_interval_seconds=excluded.capture_interval_seconds,
updated_at=excluded.updated_at
`, cfg.ClientID, cfg.CaptureInterval, cfg.UpdatedAt)
	return err
}

func (s *SQLiteStore) GetClientConfig(ctx context.Context, clientID string, defaultInterval int) (ClientConfig, error) {
	var cfg ClientConfig
	row := s.db.QueryRowContext(ctx, `SELECT client_id, capture_interval_seconds, updated_at FROM client_configs WHERE client_id = ?`, clientID)
	var updatedRaw any
	err := row.Scan(&cfg.ClientID, &cfg.CaptureInterval, &updatedRaw)
	if errors.Is(err, sql.ErrNoRows) {
		cfg = ClientConfig{ClientID: clientID, CaptureInterval: defaultInterval, UpdatedAt: time.Now().UTC()}
		if upErr := s.UpsertClientConfig(ctx, cfg); upErr != nil {
			return ClientConfig{}, upErr
		}
		return cfg, nil
	}
	cfg.UpdatedAt = parseSQLTime(updatedRaw)
	return cfg, err
}

func (s *SQLiteStore) InsertScreenshot(ctx context.Context, meta ScreenshotMeta) error {
	if meta.UploadedAt.IsZero() {
		meta.UploadedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO screenshots(client_id, captured_at, monitor_index, relative_path, size_bytes, sha256, uploaded_at)
VALUES(?, ?, ?, ?, ?, ?, ?)`,
		meta.ClientID, meta.CapturedAt, meta.MonitorIndex, meta.RelativePath, meta.SizeBytes, meta.SHA256, meta.UploadedAt)
	return err
}

func (s *SQLiteStore) ListClients(ctx context.Context) ([]ClientSummary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT c.client_id,
       COALESCE(MAX(sc.uploaded_at), c.updated_at) AS last_seen_at,
       COALESCE(MAX(sc.captured_at), c.updated_at) AS last_captured_at,
       COALESCE(COUNT(sc.id), 0) AS screenshot_count,
       c.capture_interval_seconds
FROM client_configs c
LEFT JOIN screenshots sc ON sc.client_id = c.client_id
GROUP BY c.client_id, c.capture_interval_seconds
ORDER BY last_seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ClientSummary
	for rows.Next() {
		var sClient ClientSummary
		var lastSeenRaw, lastCapturedRaw any
		if err := rows.Scan(&sClient.ClientID, &lastSeenRaw, &lastCapturedRaw, &sClient.ScreenshotCount, &sClient.CaptureIntervalS); err != nil {
			return nil, err
		}
		sClient.LastSeenAt = parseSQLTime(lastSeenRaw)
		sClient.LastCapturedAt = parseSQLTime(lastCapturedRaw)
		out = append(out, sClient)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) ListScreenshots(ctx context.Context, clientID string, limit, offset int) ([]ScreenshotMeta, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, client_id, captured_at, monitor_index, relative_path, size_bytes, sha256, uploaded_at
FROM screenshots
WHERE client_id = ?
ORDER BY captured_at DESC
LIMIT ? OFFSET ?`, clientID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ScreenshotMeta
	for rows.Next() {
		var m ScreenshotMeta
		var capturedRaw, uploadedRaw any
		if err := rows.Scan(&m.ID, &m.ClientID, &capturedRaw, &m.MonitorIndex, &m.RelativePath, &m.SizeBytes, &m.SHA256, &uploadedRaw); err != nil {
			return nil, err
		}
		m.CapturedAt = parseSQLTime(capturedRaw)
		m.UploadedAt = parseSQLTime(uploadedRaw)
		out = append(out, m)
	}
	return out, rows.Err()
}

func parseSQLTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t.UTC()
	case string:
		if ts, err := time.Parse(time.RFC3339Nano, t); err == nil {
			return ts.UTC()
		}
		if ts, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", t); err == nil {
			return ts.UTC()
		}
	case []byte:
		return parseSQLTime(string(t))
	}
	return time.Now().UTC()
}

func (s *SQLiteStore) DeleteOlderThan(ctx context.Context, before time.Time) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT relative_path FROM screenshots WHERE captured_at < ?`, before)
	if err != nil {
		return nil, err
	}
	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			rows.Close()
			return nil, err
		}
		paths = append(paths, p)
	}
	rows.Close()
	_, err = s.db.ExecContext(ctx, `DELETE FROM screenshots WHERE captured_at < ?`, before)
	return paths, err
}
