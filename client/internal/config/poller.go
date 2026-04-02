package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"
)

type Poller struct {
	baseURL  string
	clientID string
	token    string
	http     *http.Client
	interval atomic.Int64
}

func NewPoller(baseURL, clientID, token string, defaultSec int) *Poller {
	p := &Poller{
		baseURL:  baseURL,
		clientID: clientID,
		token:    token,
		http:     &http.Client{Timeout: 15 * time.Second},
	}
	p.interval.Store(int64(defaultSec))
	return p
}

func (p *Poller) CurrentInterval() time.Duration {
	return time.Duration(p.interval.Load()) * time.Second
}

func (p *Poller) PollOnce(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/clients/%s/config", p.baseURL, p.clientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	if p.token != "" {
		req.Header.Set("Authorization", "Bearer "+p.token)
	}
	resp, err := p.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("config poll status: %s", resp.Status)
	}
	var payload struct {
		CaptureIntervalSeconds int `json:"capture_interval_seconds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	if payload.CaptureIntervalSeconds >= 5 {
		p.interval.Store(int64(payload.CaptureIntervalSeconds))
	}
	return nil
}
