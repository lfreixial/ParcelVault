package database

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"

	"github.com/lfreixial/ParcelVault/migrations"
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

// Migrate applies pending migrations before the server accepts requests.
func Migrate(ctx context.Context, db *sql.DB, driver string) error {
	p, err := migrationProvider(db, driver)
	if err != nil {
		return errors.New("cannot configure database migrations")
	}
	if _, err = p.Up(ctx); err != nil {
		return errors.New("cannot apply database migrations")
	}
	current, target, err := p.GetVersions(ctx)
	if err != nil || current != target {
		return errors.New("unsupported database schema version")
	}
	return nil
}

func migrationProvider(db *sql.DB, driver string) (*goose.Provider, error) {
	opts := []goose.ProviderOption{goose.WithLogger(goose.NopLogger())}
	var dialect goose.Dialect
	switch driver {
	case "sqlite":
		dialect = goose.DialectSQLite3
	case "postgres":
		dialect = goose.DialectPostgres
		locker, err := lock.NewPostgresSessionLocker()
		if err != nil {
			return nil, err
		}
		opts = append(opts, goose.WithSessionLocker(locker))
	default:
		return nil, errors.New("unsupported migration dialect")
	}
	files, err := fs.Sub(migrations.Files, driver)
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(dialect, db, files, opts...)
}
