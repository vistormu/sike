package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"sike/domain/model"
)

func TestPlacements(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, model.DefaultAreas)
	them, _ := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼"}, model.DefaultAreas)

	gym := model.Placement{Days: model.DaysOf(time.Wednesday, time.Saturday), Start: 18 * 60, End: 19*60 + 30, NotifyStart: true}
	early := model.Placement{Days: model.DaysOf(time.Monday), Start: 7 * 60, End: 8 * 60, NotifyEnd: true}
	task, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "Gym", Color: "apricot", When: model.Weekly}, []model.Placement{gym, early})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.TaskPlacements(ctx, u.ID, task.ID)
	if err != nil || len(got) != 2 {
		t.Fatalf("placements: %+v, %v", got, err)
	}
	// by start; every field round-trips
	if g := got[0]; g.Start != early.Start || g.End != early.End || g.Days != early.Days || g.NotifyStart || !g.NotifyEnd || g.TaskID != task.ID {
		t.Errorf("first = %+v, want the 07:00 one", g)
	}
	if g := got[1]; g.Days.List()[0] != time.Wednesday || !g.NotifyStart {
		t.Errorf("second = %+v, want wed+sat 18:00", g)
	}
	if ps, _ := s.TaskPlacements(ctx, them.ID, task.ID); len(ps) != 0 {
		t.Errorf("someone else sees %d placements", len(ps))
	}

	// add one, remove one
	p, err := s.AddPlacement(ctx, u.ID, model.Placement{TaskID: task.ID, Days: model.DaysOf(time.Sunday), Start: 10 * 60, End: 11 * 60})
	if err != nil || p.ID == 0 {
		t.Fatalf("add: %+v, %v", p, err)
	}
	if _, err := s.AddPlacement(ctx, them.ID, model.Placement{TaskID: task.ID, Days: 1, Start: 0, End: 60}); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("add to another's task: err = %v, want ErrNotFound", err)
	}
	if err := s.DeletePlacement(ctx, them.ID, p.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("delete another's placement: err = %v, want ErrNotFound", err)
	}
	if err := s.DeletePlacement(ctx, u.ID, p.ID); err != nil {
		t.Fatal(err)
	}

	// the sql refuses what the service would: no days, the end before the start
	if _, err := s.AddPlacement(ctx, u.ID, model.Placement{TaskID: task.ID, Days: 0, Start: 0, End: 60}); err == nil {
		t.Error("stored a placement with no days")
	}
	if _, err := s.AddPlacement(ctx, u.ID, model.Placement{TaskID: task.ID, Days: 1, Start: 120, End: 60}); err == nil {
		t.Error("stored a placement that ends before it starts")
	}

	// a pending task takes none
	pend, _ := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "Fix bike", Color: "mint", When: model.Pending}, nil)
	if _, err := s.AddPlacement(ctx, u.ID, model.Placement{TaskID: pend.ID, Days: 1, Start: 0, End: 60}); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("place a pending task: err = %v, want ErrNotFound", err)
	}

	// the week: every live placement of the person
	if all, _ := s.ListPlacements(ctx, u.ID); len(all) != 2 {
		t.Errorf("week has %d placements, want 2", len(all))
	}
	// delete hides them, undo brings them back, the next delete erases them
	if err := s.DeleteTask(ctx, u.ID, task.ID); err != nil {
		t.Fatal(err)
	}
	if all, _ := s.ListPlacements(ctx, u.ID); len(all) != 0 {
		t.Errorf("a deleted task's placements are in the week")
	}
	if err := s.RestoreTask(ctx, u.ID, task.ID); err != nil {
		t.Fatal(err)
	}
	if all, _ := s.ListPlacements(ctx, u.ID); len(all) != 2 {
		t.Errorf("after undo: %d placements, want 2", len(all))
	}
	_ = s.DeleteTask(ctx, u.ID, task.ID)
	_ = s.DeleteTask(ctx, u.ID, pend.ID) // erases Gym for good
	var n int
	if err := s.db.GetContext(ctx, &n, `SELECT COUNT(*) FROM placements`); err != nil || n != 0 {
		t.Errorf("%d placements left after the task is erased, want 0", n)
	}
}

func TestUpdateReplacesPlacements(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, model.DefaultAreas)
	p := model.Placement{Days: model.DaysOf(time.Monday), Start: 9 * 60, End: 17 * 60}
	task, _ := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "Work", Color: "sky", When: model.Weekly}, []model.Placement{p, p})

	q := model.Placement{Days: model.DaysOf(time.Friday), Start: 9 * 60, End: 14 * 60}
	if _, err := s.UpdateTask(ctx, model.Task{ID: task.ID, UserID: u.ID, Name: "Work", Color: "sky", When: model.Weekly}, []model.Placement{q}); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.TaskPlacements(ctx, u.ID, task.ID); len(got) != 1 || got[0].End != q.End {
		t.Errorf("after update: %+v, want only friday", got)
	}
	// moving off weekly drops them
	if err := s.MoveTask(ctx, u.ID, task.ID, model.Pending); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.TaskPlacements(ctx, u.ID, task.ID); len(got) != 0 {
		t.Errorf("a pending task kept %d placements", len(got))
	}
}
