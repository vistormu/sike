package store

import (
	"context"
	"testing"

	"sike/domain/clock"
	"sike/domain/model"
)

func TestTaskDates(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	oct := func(day int) clock.Date { return clock.NewDate(2026, 10, day) }

	dentist := model.Dates{From: oct(15), To: oct(15), Start: 11 * 60, End: 12 * 60, Notify: true}
	taxes := model.Dates{To: oct(20), Start: clock.NoTime, End: clock.NoTime}
	a, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "Dentist", Color: "coral", When: model.Dated, Date: &dentist})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "Taxes", Color: "coral", When: model.Dated, Date: &taxes}); err != nil {
		t.Fatal(err)
	}
	// round trip, with the NULLs as NoTime and the zero date
	if a.Date == nil || *a.Date != dentist {
		t.Errorf("dentist = %+v, want %+v", a.Date, dentist)
	}
	all, _ := s.ListTasks(ctx, u.ID)
	if len(all) != 2 || all[0].Date == nil || all[1].Date == nil {
		t.Fatalf("list: %+v", all)
	}
	for _, x := range all {
		if x.Name == "Taxes" && *x.Date != taxes {
			t.Errorf("taxes = %+v, want %+v (a deadline: no start, no times)", *x.Date, taxes)
		}
	}

	// an edit replaces the dates; a move drops them
	moved := model.Dates{From: oct(16), To: oct(18), Start: clock.NoTime, End: clock.NoTime}
	got, err := s.UpdateTask(ctx, model.Task{ID: a.ID, UserID: u.ID, Name: "Trip", Color: "pink", When: model.Dated, Date: &moved})
	if err != nil || got.Date == nil || *got.Date != moved {
		t.Fatalf("update: %+v, %v", got.Date, err)
	}
	if err := s.MoveTask(ctx, u.ID, a.ID, model.Someday); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetTask(ctx, u.ID, a.ID); got.Date != nil {
		t.Errorf("a someday task kept its dates: %+v", got.Date)
	}
	// the sql refuses a start after the end
	bad := model.Dates{From: oct(20), To: oct(19), Start: clock.NoTime, End: clock.NoTime}
	if _, err := s.CreateTask(ctx, model.Task{UserID: u.ID, Name: "x", Color: "coral", When: model.Dated, Date: &bad}); err == nil {
		t.Error("stored dates that end before they start")
	}
}
