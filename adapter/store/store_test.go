package store

import (
	"context"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jmoiron/sqlx"
)

var memN atomic.Int64

// testDB returns a fresh, migrated in-memory database, closed with the test.
// in a _test file on purpose: the binary must not import "testing".
func testDB(t *testing.T) *sqlx.DB {
	t.Helper()
	// one connection: each new connection to :memory: is a different, empty db
	dsn := fmt.Sprintf("file:mem%d?mode=memory&_pragma=foreign_keys(1)", memN.Add(1))
	db, err := open(dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { db.Close() })
	if _, err := Migrate(context.Background(), db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestOpenSetsPragmas(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "sike.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	var journal string
	if err := db.Get(&journal, "PRAGMA journal_mode"); err != nil {
		t.Fatal(err)
	}
	if journal != "wal" {
		t.Errorf("journal_mode = %q, want wal", journal)
	}
	var timeout int
	if err := db.Get(&timeout, "PRAGMA busy_timeout"); err != nil {
		t.Fatal(err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := testDB(t)
	v1, err := Migrate(context.Background(), db)
	if err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	if v1 < 1 {
		t.Errorf("version = %d after migrating, want ≥ 1", v1)
	}
}

func TestTestDBsAreIsolated(t *testing.T) {
	a, b := testDB(t), testDB(t)
	if _, err := a.Exec("CREATE TABLE only_in_a (x)"); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := b.Get(&n, "SELECT count(*) FROM sqlite_master WHERE name = 'only_in_a'"); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Error("a table made in one test db shows up in another")
	}
}
