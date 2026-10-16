package store

import (
	"context"
	"errors"
	"testing"

	"sike/domain/model"
)

func TestAreasComeWithTheUser(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, err := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	if err != nil {
		t.Fatal(err)
	}
	areas, err := s.ListAreas(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(areas) != len(model.Colors) {
		t.Fatalf("%d areas, want %d", len(areas), len(model.Colors))
	}
	for i, a := range areas {
		want := model.DefaultAreas[i]
		if a.UserID != u.ID || a.Color != want.Color || a.Name != want.Name {
			t.Errorf("area %d = %+v, want %s %s (palette order)", i, a, want.Color, want.Name)
		}
	}

	if _, err := s.ListAreas(ctx, 99); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("areas of nobody: err = %v, want ErrNotFound", err)
	}
}

func TestCreateUserIsAtomic(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	bad := []model.Area{{Color: "lavender", Name: "Self"}, {Color: "beige", Name: "Nope"}} // fails the check
	if _, err := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, bad); err == nil {
		t.Fatal("a bad area did not fail CreateUser")
	}
	if users, _ := s.ListUsers(ctx); len(users) != 0 {
		t.Errorf("a user without areas was left behind: %+v", users)
	}
}

func TestRenameAreas(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	other, _ := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼", Warn: model.DefaultWarn}, model.DefaultAreas)

	if err := s.RenameAreas(ctx, u.ID, map[string]string{"sky": "Job"}); err != nil {
		t.Fatal(err)
	}
	areas, _ := s.ListAreas(ctx, u.ID)
	if areas[1].Name != "Job" {
		t.Errorf("sky = %q after rename, want Job", areas[1].Name)
	}
	if theirs, _ := s.ListAreas(ctx, other.ID); theirs[1].Name != "Work" {
		t.Errorf("renaming mine renamed theirs: %q", theirs[1].Name)
	}
	if err := s.RenameAreas(ctx, u.ID, map[string]string{"mint": "job", "lime": "Play"}); !errors.Is(err, model.ErrTaken) {
		t.Errorf("a second area named job: err = %v, want ErrTaken", err)
	}
	if areas, _ := s.ListAreas(ctx, u.ID); areas[4].Name != "Home" || areas[5].Name != "Social" {
		t.Errorf("a refused rename stored %q, %q; want none of it", areas[4].Name, areas[5].Name)
	}
	if err := s.RenameAreas(ctx, u.ID, map[string]string{"lavender": "Job", "sky": "Self"}); err != nil {
		t.Errorf("two areas trading names: %v", err)
	}
	if err := s.RenameAreas(ctx, other.ID, map[string]string{"mint": "Job"}); err != nil {
		t.Errorf("another person may use the same name: %v", err)
	}
	if err := s.RenameAreas(ctx, 99, map[string]string{"sky": "x"}); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("rename for nobody: err = %v, want ErrNotFound", err)
	}
}

func TestDeletingAUserDeletesTheirAreas(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	s := New(db)
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	if err := s.DeleteUser(ctx, u.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.Get(&n, `SELECT count(*) FROM areas`); err != nil || n != 0 {
		t.Errorf("%d areas left (err %v), want 0", n, err)
	}
}

// people made before migration 3 get the default areas from it.
func TestAreasMigrationBackfills(t *testing.T) {
	ctx := context.Background()
	db, err := OpenMemory()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	p, err := provider(db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.UpTo(ctx, 2); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO users (name, emoji) VALUES ('You', '🦊'), ('Partner', '🐼')`); err != nil {
		t.Fatal(err)
	}
	if _, err := Migrate(ctx, db); err != nil {
		t.Fatal(err)
	}
	for id := int64(1); id <= 2; id++ {
		areas, err := New(db).ListAreas(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		for i, a := range areas {
			if a.Color != model.DefaultAreas[i].Color || a.Name != model.DefaultAreas[i].Name {
				t.Errorf("user %d area %d = %+v, want the defaults (the sql copy of model.DefaultAreas drifted?)", id, i, a)
			}
		}
	}
}
