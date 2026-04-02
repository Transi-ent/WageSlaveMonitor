package main

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"wageslavemonitor/client/internal/capture"
	"wageslavemonitor/client/internal/config"
	"wageslavemonitor/client/internal/spool"
	"wageslavemonitor/client/internal/upload"
)

func main() {
	handled, err := handleServiceCommand()
	if err != nil {
		log.Fatal(err)
	}
	if handled {
		return
	}

	baseURL := getenv("SERVER_BASE_URL", "http://127.0.0.1:8080")
	token := os.Getenv("AUTH_TOKEN")
	dataDir := getenv("AGENT_DATA_DIR", "./agent-data")
	clientID, err := ensureClientID(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	q, err := spool.New(filepath.Join(dataDir, "spool"))
	if err != nil {
		log.Fatal(err)
	}
	poller := config.NewPoller(baseURL, clientID, token, 30)
	uploader := upload.New(baseURL, token)

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			_ = poller.PollOnce(context.Background())
		}
	}()

	for {
		shots, err := capture.CaptureAll(75)
		if err == nil {
			for _, s := range shots {
				_ = q.Enqueue(spool.Item{
					ClientID:     clientID,
					CapturedAt:   s.CapturedAt,
					MonitorIndex: s.MonitorIndex,
				}, s.JPEG)
			}
		}
		flush(q, uploader)
		time.Sleep(poller.CurrentInterval())
	}
}

func flush(q *spool.Queue, u *upload.Uploader) {
	items, err := q.List()
	if err != nil {
		return
	}
	for _, it := range items {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		err := u.Upload(ctx, it)
		cancel()
		if err != nil {
			return
		}
		q.Ack(it.ID)
	}
}

func ensureClientID(root string) (string, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(root, "client-id.txt")
	if b, err := os.ReadFile(path); err == nil && len(b) > 0 {
		return string(b), nil
	}
	id := uuid.NewString()
	if err := os.WriteFile(path, []byte(id), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
