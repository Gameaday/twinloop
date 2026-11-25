package platform

import (
	"path/filepath"
	"testing"
)

const testMigrations = "../../migrations"

func TestOpenMigratesAndReopensIdempotently(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(dir, testMigrations)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	var name string
	if err := db.QueryRow(
		`SELECT name FROM sqlite_master WHERE type='table' AND name='corpse_remnants'`,
	).Scan(&name); err != nil {
		t.Fatalf("schema missing: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	db2, err := Open(dir, testMigrations)
	if err != nil {
		t.Fatalf("reopen (migrations must be idempotent): %v", err)
	}
	defer db2.Close()
	var applied int
	if err := db2.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&applied); err != nil || applied != 1 {
		t.Fatalf("applied migrations = %d (err %v), want 1", applied, err)
	}
}

func TestOpenRejectsMissingMigrationsDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	if db, err := Open(t.TempDir(), missing); err == nil {
		db.Close()
		t.Fatal("missing migrations dir must fail the open")
	}
}
