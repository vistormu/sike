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
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)

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
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
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
	you, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	them, _ := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼", Warn: model.DefaultWarn}, model.DefaultAreas)
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
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	if _, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "x", Color: "beige", When: model.Pending}); err == nil {
		t.Error("a task in no area was stored")
	}
	if _, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "x", Color: "sky", When: "later"}); err == nil {
		t.Error("a task with an unknown when was stored")
	}
}

func TestUpdateTask(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	a := newTask(t, s, u.ID, "a", "sky", model.Pending)
	newTask(t, s, u.ID, "b", "sky", model.Pending)
	newTask(t, s, u.ID, "wish", "sky", model.Someday)

	// same when: renamed and recolored, its place kept
	got, err := s.UpdateTask(ctx, model.Task{ID: a.ID, UserID: u.ID, Name: "a2", Color: "sky", When: model.Pending})
	if err != nil || got.Name != "a2" || got.Position != a.Position {
		t.Fatalf("update in place: %+v, %v (position was %d)", got, err, a.Position)
	}
	// a new when: on top there
	if _, err := s.UpdateTask(ctx, model.Task{ID: a.ID, UserID: u.ID, Name: "a2", Color: "pink", When: model.Someday}); err != nil {
		t.Fatal(err)
	}
	all, _ := s.ListTasks(ctx, u.ID)
	if g := taskNames(all); !slices.Equal(g, []string{"b", "wish", "a2"}) {
		t.Errorf("order = %q, want b, then someday: wish (sky) before a2 (pink)", g)
	}
	if all[2].When != model.Someday || all[2].Color != "pink" || all[2].Position >= all[1].Position {
		t.Errorf("moved task = %+v, want someday, pink, on top (position below %d)", all[2], all[1].Position)
	}
	// someone else's task: not found
	them, _ := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼", Warn: model.DefaultWarn}, model.DefaultAreas)
	if _, err := s.UpdateTask(ctx, model.Task{ID: a.ID, UserID: them.ID, Name: "x", Color: "sky", When: model.Pending}); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("update another's task: err = %v, want ErrNotFound", err)
	}
}

func TestDeleteThenRestore(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	a := newTask(t, s, u.ID, "a", "sky", model.Pending)
	b := newTask(t, s, u.ID, "b", "sky", model.Pending)
	newTask(t, s, u.ID, "c", "sky", model.Pending)

	if err := s.DeleteTask(ctx, u.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ListTasks(ctx, u.ID); !slices.Equal(taskNames(got), []string{"c", "a"}) {
		t.Errorf("after delete: %q, want c, a", taskNames(got))
	}
	if _, err := s.GetTask(ctx, u.ID, b.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("get a deleted task: err = %v, want ErrNotFound", err)
	}
	if d, err := s.GetDeletedTask(ctx, u.ID, b.ID); err != nil || d.Name != "b" {
		t.Errorf("the deleted task: %+v, %v", d, err)
	}
	if err := s.DeleteTask(ctx, u.ID, b.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("delete twice: err = %v, want ErrNotFound", err)
	}
	// undo: back in its place
	if err := s.RestoreTask(ctx, u.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.ListTasks(ctx, u.ID); !slices.Equal(taskNames(got), []string{"c", "b", "a"}) {
		t.Errorf("after undo: %q, want c, b, a", taskNames(got))
	}
	if err := s.RestoreTask(ctx, u.ID, b.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("restore a live task: err = %v, want ErrNotFound", err)
	}
	// only the last delete waits: the next one erases it for good
	if err := s.DeleteTask(ctx, u.ID, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteTask(ctx, u.ID, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.RestoreTask(ctx, u.ID, a.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("restore an older delete: err = %v, want ErrNotFound", err)
	}
	var rows int
	if err := s.db.GetContext(ctx, &rows, `SELECT COUNT(*) FROM tasks`); err != nil || rows != 2 {
		t.Errorf("%d rows, want 2: c, and b waiting for its undo", rows)
	}
	// someone else's undo: not found
	them, _ := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼", Warn: model.DefaultWarn}, model.DefaultAreas)
	if err := s.RestoreTask(ctx, them.ID, b.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("restore another's task: err = %v, want ErrNotFound", err)
	}
}
