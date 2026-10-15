package store

import (
	"context"
	"errors"
	"testing"

	"sike/domain/clock"
	"sike/domain/model"
)

func TestMoods(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, err := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, model.DefaultAreas)
	if err != nil {
		t.Fatal(err)
	}
	other, _ := s.CreateUser(ctx, model.User{Name: "Ana", Emoji: "🐻", Warn: model.DefaultWarn}, model.DefaultAreas)
	today, yesterday := clock.NewDate(2026, 10, 15), clock.NewDate(2026, 10, 14)
	mood := func(user int64, d clock.Date) int {
		t.Helper()
		m, err := s.GetMood(ctx, user, d)
		if err != nil {
			t.Fatal(err)
		}
		return m
	}

	if m := mood(u.ID, today); m != 0 {
		t.Errorf("no mood yet: %d, want 0", m)
	}
	for _, m := range []int{2, 4} { // the second replaces the first
		if err := s.SetMood(ctx, u.ID, today, m); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetMood(ctx, u.ID, yesterday, 1); err != nil {
		t.Fatal(err)
	}
	if mood(u.ID, today) != 4 || mood(u.ID, yesterday) != 1 || mood(other.ID, today) != 0 {
		t.Error("moods are not one per person per day")
	}
	if err := s.SetMood(ctx, u.ID, today, 6); err == nil {
		t.Error("a mood of 6 was stored")
	}
	if err := s.SetMood(ctx, 99, today, 3); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("nobody's mood: err = %v, want ErrNotFound", err)
	}
	if err := s.ClearMood(ctx, u.ID, today); err != nil {
		t.Fatal(err)
	}
	if mood(u.ID, today) != 0 || mood(u.ID, yesterday) != 1 {
		t.Error("clear took the wrong day")
	}
}
