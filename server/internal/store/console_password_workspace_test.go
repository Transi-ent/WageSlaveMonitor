package store

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyDefaultConsolePassword_ForWorkspaceDB(t *testing.T) {
	// This test checks the developer machine DB to diagnose auth issues.
	dbPath := filepath.Join("..", "..", "data", "meta.db") // relative to this package dir: server/internal/store
	if _, err := os.Stat(dbPath); err != nil {
		t.Skipf("workspace meta.db not found at %s: %v", dbPath, err)
	}

	db, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Init(context.Background()); err != nil {
		t.Fatal(err)
	}

	ok, err := db.VerifyConsolePassword(context.Background(), DefaultConsolePassword)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatalf("expected DefaultConsolePassword to verify for workspace meta.db, but it did not (DefaultConsolePassword=%q)", DefaultConsolePassword)
	}
}
