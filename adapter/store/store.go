// Package store is the sqlite adapter: connection, migrations and queries.
package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"net/url"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // pure go: no cgo, a static binary
)

//go:embed migrations/*.sql
var migrations embed.FS

// pragmas are not optional: wal lets reads run during the one write,
// busy_timeout makes a second writer wait instead of failing, and foreign keys
// are off in sqlite unless asked for.
var pragmas = []string{"foreign_keys(1)", "journal_mode(WAL)", "busy_timeout(5000)"}

// Open opens the database file at path, creating it if needed.
func Open(path string) (*sqlx.DB, error) {
	q := url.Values{"_pragma": pragmas}
	db, err := open("file:" + path + "?" + q.Encode())
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return db, nil
}

func open(dsn string) (*sqlx.DB, error) {
	raw, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	// sqlx knows the dialect as "sqlite3": ? placeholders, named queries
	db := sqlx.NewDb(raw, "sqlite3")
	if err := check(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// check fails on a connection where the pragmas did not take: they default off
// silently.
func check(db *sqlx.DB) error {
	var fk int
	if err := db.Get(&fk, "PRAGMA foreign_keys"); err != nil {
		return err
	}
	if fk != 1 {
		return fmt.Errorf("sqlite: foreign keys are off")
	}
	var journal string
	if err := db.Get(&journal, "PRAGMA journal_mode"); err != nil {
		return err
	}
	if journal != "wal" && journal != "memory" {
		return fmt.Errorf("sqlite: journal mode is %q, not wal", journal)
	}
	return nil
}

// Migrate applies every pending migration and returns the resulting version.
func Migrate(ctx context.Context, db *sqlx.DB) (int64, error) {
	p, err := provider(db)
	if err != nil {
		return 0, err
	}
	if _, err := p.Up(ctx); err != nil {
		return 0, err
	}
	return p.GetDBVersion(ctx)
}

func provider(db *sqlx.DB) (*goose.Provider, error) {
	fsys, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, err
	}
	return goose.NewProvider(goose.DialectSQLite3, db.DB, fsys)
}
