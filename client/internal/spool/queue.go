package spool

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Item struct {
	ID           string    `json:"id"`
	ClientID     string    `json:"client_id"`
	CapturedAt   time.Time `json:"captured_at"`
	MonitorIndex int       `json:"monitor_index"`
	ImagePath    string    `json:"image_path"`
}

type Queue struct {
	root string
}

func New(root string) (*Queue, error) {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	return &Queue{root: root}, nil
}

func (q *Queue) Enqueue(it Item, image []byte) error {
	if it.ID == "" {
		it.ID = time.Now().UTC().Format("20060102150405.000000000")
	}
	imagePath := filepath.Join(q.root, it.ID+".jpg")
	metaPath := filepath.Join(q.root, it.ID+".json")
	if err := os.WriteFile(imagePath, image, 0o600); err != nil {
		return err
	}
	it.ImagePath = imagePath
	body, _ := json.Marshal(it)
	return os.WriteFile(metaPath, body, 0o600)
}

func (q *Queue) List() ([]Item, error) {
	entries, err := os.ReadDir(q.root)
	if err != nil {
		return nil, err
	}
	var out []Item
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(q.root, e.Name()))
		if err != nil {
			continue
		}
		var it Item
		if err := json.Unmarshal(b, &it); err != nil {
			continue
		}
		out = append(out, it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt.Before(out[j].CapturedAt) })
	return out, nil
}

func (q *Queue) Ack(id string) {
	_ = os.Remove(filepath.Join(q.root, id+".json"))
	_ = os.Remove(filepath.Join(q.root, id+".jpg"))
}
