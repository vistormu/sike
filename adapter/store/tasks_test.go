package store

import (
	"context"
	"errors"
	"slices"
	"testing"

	"sike/domain/model"
)

func taskNames(ts []model.Task) []string {
	out := make([]string, len(ts))
	for i, t := range ts {
		out[i] = t.Name
	}
	return out
}

func newTask(t *testing.T, s *Store, user int64, name, color string, when model.When) model.Task {
	t.Helper()
	task, err := s.CreateTask(context.Background(), model.Task{UserID: user, Name: name, Color: color, When: when})
	if err != nil {
		t.Fatalf("CreateTask %s: %v", name, err)
	}
	return task
}

func TestTasksOrder(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, model.DefaultAreas)

	newTask(t, s, u.ID, "old admin", "coral", model.Pending)
	newTask(t, s, u.ID, "wish", "lavender", model.Someday)
	newTask(t, s, u.ID, "self thing", "lavender", model.Pending)
	newTask(t, s, u.ID, "new admin", "coral", model.Pending)

	got, err := s.ListTasks(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	// pending before someday; in pending, lavender before coral; newest first
	want := []string{"self thing", "new admin", "old admin", "wish"}
	if g := taskNames(got); !slices.Equal(g, want) {
		t.Errorf("order = %q, want %q", g, want)
	}
}

func TestMoveTaskGoesOnTop(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, model.DefaultAreas)
	w := newTask(t, s, u.ID, "was a wish", "sky", model.Someday)
	newTask(t, s, u.ID, "already owed", "sky", model.Pending)

	if err := s.MoveTask(ctx, u.ID, w.ID, model.Pending); err != nil {
		t.Fatal(err)
	}
	got, _ := s.ListTasks(ctx, u.ID)
	if g := taskNames(got); !slices.Equal(g, []string{"was a wish", "already owed"}) || got[0].When != model.Pending {
		t.Errorf("after move: %q (%s), want the moved task on top of pending", g, got[0].When)
	}
}

func TestTasksBelongToTheirPerson(t *testing.T) {
	ctx := context.Background()
	db := testDB(t)
	s := New(db)
	you, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, model.DefaultAreas)
	them, _ := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼"}, model.DefaultAreas)
	mine := newTask(t, s, you.ID, "mine", "sky", model.Pending)

	if _, err := s.GetTask(ctx, them.ID, mine.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("get another's task: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteTask(ctx, them.ID, mine.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("delete another's task: err = %v, want ErrNotFound", err)
	}
	if err := s.MoveTask(ctx, them.ID, mine.ID, model.Someday); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("move another's task: err = %v, want ErrNotFound", err)
	}
	if theirs, _ := s.ListTasks(ctx, them.ID); len(theirs) != 0 {
		t.Errorf("they see %d tasks, want 0", len(theirs))
	}

	if err := s.DeleteTask(ctx, you.ID, mine.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTask(ctx, you.ID, mine.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("delete twice: err = %v, want ErrNotFound", err)
	}

	newTask(t, s, you.ID, "goes with me", "sky", model.Pending)
	if err := s.DeleteUser(ctx, you.ID); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.Get(&n, `SELECT count(*) FROM tasks`); err != nil || n != 0 {
		t.Errorf("%d tasks left after deleting their person (err %v)", n, err)
	}
}

func TestTaskNeedsAnAreaAndAWhen(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, model.DefaultAreas)
	if _, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "x", Color: "beige", When: model.Pending}); err == nil {
		t.Error("a task in no area was stored")
	}
	if _, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "x", Color: "sky", When: "later"}); err == nil {
		t.Error("a task with an unknown when was stored")
	}
}
