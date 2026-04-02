package store

import (
	"context"
	"testing"
	"time"
)

func TestSQLiteInsertAndList(t *testing.T) {
	db, err := NewSQLiteStore(t.TempDir() + "/meta.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertClientConfig(context.Background(), ClientConfig{
		ClientID: "c1", CaptureInterval: 10, UpdatedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.InsertScreenshot(context.Background(), ScreenshotMeta{
		ClientID: "c1", CapturedAt: time.Now().UTC(), MonitorIndex: 0,
		RelativePath: "screenshots/c1/a.jpg", SizeBytes: 1, SHA256: "x",
	}); err != nil {
		t.Fatal(err)
	}
	clients, err := db.ListClients(context.Background())
	if err != nil || len(clients) != 1 {
		t.Fatalf("clients err=%v len=%d", err, len(clients))
	}
	shots, err := db.ListScreenshots(context.Background(), "c1", 10, 0)
	if err != nil || len(shots) != 1 {
		t.Fatalf("shots err=%v len=%d", err, len(shots))
	}
}

func TestDefaultConsolePasswordAfterInit(t *testing.T) {
	db, err := NewSQLiteStore(t.TempDir() + "/meta.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	ok, err := db.VerifyConsolePassword(context.Background(), DefaultConsolePassword)
	if err != nil || !ok {
		t.Fatalf("default password verify err=%v ok=%v", err, ok)
	}
	if err := db.SetConsolePassword(context.Background(), "newpass-ok"); err != nil {
		t.Fatal(err)
	}
	ok, err = db.VerifyConsolePassword(context.Background(), DefaultConsolePassword)
	if err != nil || ok {
		t.Fatalf("old password should fail err=%v ok=%v", err, ok)
	}
	ok, err = db.VerifyConsolePassword(context.Background(), "newpass-ok")
	if err != nil || !ok {
		t.Fatalf("new password verify err=%v ok=%v", err, ok)
	}
}

func TestConsolePasswordNotOverwrittenWhenCustomized(t *testing.T) {
	db, err := NewSQLiteStore(t.TempDir() + "/meta.db")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Init(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := db.SetConsolePassword(context.Background(), "newpass-ok"); err != nil {
		t.Fatal(err)
	}
	// Simulate restart.
	if err := db.Init(context.Background()); err != nil {
		t.Fatal(err)
	}

	ok, err := db.VerifyConsolePassword(context.Background(), DefaultConsolePassword)
	if err != nil || ok {
		t.Fatalf("default password should fail after customization err=%v ok=%v", err, ok)
	}

	ok, err = db.VerifyConsolePassword(context.Background(), "newpass-ok")
	if err != nil || !ok {
		t.Fatalf("custom password verify err=%v ok=%v", err, ok)
	}
}
