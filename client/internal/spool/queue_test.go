package spool

import (
	"testing"
	"time"
)

func TestQueueEnqueueListAck(t *testing.T) {
	q, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := Item{ID: "a", ClientID: "c1", CapturedAt: time.Now().Add(-time.Minute), MonitorIndex: 0}
	b := Item{ID: "b", ClientID: "c1", CapturedAt: time.Now(), MonitorIndex: 1}
	if err := q.Enqueue(a, []byte("x")); err != nil {
		t.Fatal(err)
	}
	if err := q.Enqueue(b, []byte("y")); err != nil {
		t.Fatal(err)
	}
	items, err := q.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].ID != "a" {
		t.Fatalf("expected ordering by captured time")
	}
	q.Ack("a")
	items, err = q.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "b" {
		t.Fatalf("expected only b left, got %+v", items)
	}
}
