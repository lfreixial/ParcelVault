package database

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func TestSQLiteMigrations(t *testing.T) {
	db, err := Open(t.Context(), "sqlite", filepath.Join(t.TempDir(), "pv.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	checkMigrations(t, db, "sqlite")
}

func TestPostgresMigrations(t *testing.T) {
	admin := openTestPostgres(t)
	schema := "pv_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := admin.ExecContext(t.Context(), "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.ExecContext(t.Context(), "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	cfg, err := pgx.ParseConfig(os.Getenv("PARCELVAULT_TEST_POSTGRES_DSN"))
	if err != nil {
		t.Fatal("invalid test database configuration")
	}
	cfg.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	checkMigrations(t, db, "postgres")
}

func checkMigrations(t *testing.T, db *sql.DB, driver string) {
	t.Helper()
	ctx := t.Context()
	// Simulate a conflicting pre-existing table. A failed migration must
	// preserve it and remain pending so the operator can resolve the conflict.
	if _, err := db.ExecContext(ctx, "CREATE TABLE users (legacy_id INTEGER)"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, driver); err == nil {
		t.Fatal("expected a schema conflict")
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE users RENAME TO legacy_users"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, driver); err != nil {
		t.Fatal(err)
	}
	insert := "INSERT INTO users (id, username, password_hash, created_at) VALUES ($1, 'sample', 'test-hash', '2026-01-01T00:00:00Z')"
	if _, err := db.ExecContext(ctx, insert, "019b76da-a800-7000-8000-000000000001"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, insert, "019b76da-a800-7000-8000-000000000002"); err == nil {
		t.Fatal("duplicate username accepted")
	}
	if err := Migrate(ctx, db, driver); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil || count != 1 {
		t.Fatalf("repeat migration: rows %d, error %v", count, err)
	}
	p, err := migrationProvider(db, driver)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Down(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "SELECT id FROM users"); err == nil {
		t.Fatal("users table survived rollback")
	}
	if err := Migrate(ctx, db, driver); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "INSERT INTO goose_db_version (version_id, is_applied) VALUES (2, TRUE)"); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, db, driver); err == nil {
		t.Fatal("accepted newer schema")
	}
}
