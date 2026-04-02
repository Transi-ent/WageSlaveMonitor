package config

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPollerUpdatesInterval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"capture_interval_seconds":12}`))
	}))
	defer srv.Close()

	p := NewPoller(srv.URL, "c1", "", 30)
	if err := p.PollOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := p.CurrentInterval().Seconds(); got != 12 {
		t.Fatalf("expected 12s, got %v", got)
	}
}
