// Package database opens the storage used by the service.
package database

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

// Open returns a connected pool. Source is a filesystem path for SQLite or
// a connection string for PostgreSQL. Errors are safe to include in startup logs.
func Open(ctx context.Context, driver, source string) (*sql.DB, error) {
	if source == "" {
		return nil, errors.New("database source is required")
	}
	var db *sql.DB
	var err error
	switch driver {
	case "sqlite":
		var dsn string
		dsn, err = sqliteDSN(source)
		if err != nil {
			return nil, errors.New("cannot prepare SQLite database file")
		}
		db, err = sql.Open("sqlite", dsn)
	case "postgres":
		db, err = sql.Open("pgx", source)
	default:
		return nil, errors.New("unsupported database driver")
	}
	if err != nil {
		return nil, errors.New("cannot configure database connection")
	}
	if driver == "sqlite" {
		db.SetMaxOpenConns(1)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, errors.New("cannot connect to database")
	}
	return db, nil
}

func sqliteDSN(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return "", err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return "", err
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	params := url.Values{"_pragma": {"busy_timeout(5000)", "journal_mode(WAL)", "foreign_keys(ON)"}, "_txlock": {"immediate"}}
	u := url.URL{Scheme: "file", Path: filepath.ToSlash(path), RawQuery: params.Encode()}
	return u.String(), nil
}
