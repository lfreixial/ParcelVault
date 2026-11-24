package database

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSQLiteOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "store?#.db")
	db, err := Open(t.Context(), "sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0077 != 0 || db.Stats().MaxOpenConnections != 1 {
		t.Fatalf("permissions %v, pool size %d", info.Mode(), db.Stats().MaxOpenConnections)
	}
	// Exercise a fresh connection each time so connection-local settings are covered.
	db.SetMaxIdleConns(0)
	for query, want := range map[string]string{
		"PRAGMA journal_mode": "wal", "PRAGMA foreign_keys": "1", "PRAGMA busy_timeout": "5000",
	} {
		var got string
		if err := db.QueryRowContext(t.Context(), query).Scan(&got); err != nil || got != want {
			t.Errorf("%s: got %q, want %q, error %v", query, got, want, err)
		}
	}
}

func TestOpenErrorsAreRedacted(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	for _, tc := range []struct{ driver, source string }{
		{"private-marker", "value"}, {"sqlite", ""}, {"postgres", "private-marker"},
		{"sqlite", filepath.Join(t.TempDir(), "private-marker.db")},
	} {
		if db, err := Open(ctx, tc.driver, tc.source); err == nil {
			db.Close()
			t.Fatal("expected error")
		} else if strings.Contains(err.Error(), "private-marker") {
			t.Fatal("database error included supplied value")
		}
	}
}

func openTestPostgres(t *testing.T) *sql.DB {
	t.Helper()
	source := os.Getenv("PARCELVAULT_TEST_POSTGRES_DSN")
	if source == "" {
		t.Skip("set PARCELVAULT_TEST_POSTGRES_DSN to a disposable database")
	}
	db, err := Open(t.Context(), "postgres", source)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestPostgresOpen(t *testing.T) {
	db := openTestPostgres(t)
	var value int
	if err := db.QueryRowContext(t.Context(), "SELECT 1").Scan(&value); err != nil || value != 1 {
		t.Fatal("PostgreSQL query failed")
	}
}
