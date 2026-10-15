package store

import (
	"context"
	"path/filepath"
	"testing"

	"sike/domain/model"
)

func TestBackup(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "sike.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	if _, err := New(db).CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas); err != nil {
		t.Fatal(err)
	}

	to := filepath.Join(dir, "copy.db")
	if err := Backup(ctx, db, to); err != nil {
		t.Fatal(err)
	}
	if err := Backup(ctx, db, to); err == nil {
		t.Error("a backup over an existing file")
	}
	copied, err := Open(to)
	if err != nil {
		t.Fatal(err)
	}
	defer copied.Close()
	users, err := New(copied).ListUsers(ctx)
	if err != nil || len(users) != 1 || users[0].Name != "You" {
		t.Errorf("the copy holds %+v, %v; want You", users, err)
	}
}
