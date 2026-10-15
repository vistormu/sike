package service

import (
	"context"
	"errors"
	"testing"

	"sike/domain/clock"
	"sike/domain/model"
)

type fakeMoods struct{ set []int }

func (f *fakeMoods) GetMood(context.Context, int64, clock.Date) (int, error) { return 0, nil }
func (f *fakeMoods) SetMood(_ context.Context, _ int64, _ clock.Date, m int) error {
	f.set = append(f.set, m)
	return nil
}
func (f *fakeMoods) ClearMood(context.Context, int64, clock.Date) error { return nil }

func TestSetMood(t *testing.T) {
	f := &fakeMoods{}
	s := NewMoods(f)
	day := clock.NewDate(2026, 10, 15)
	for _, m := range []int{0, 1, 5, 6, -1} {
		err := s.Set(context.Background(), 1, day, m)
		if ok := m >= 1 && m <= 5; ok != (err == nil) || !ok && !errors.Is(err, model.ErrInvalid) {
			t.Errorf("mood %d: err = %v", m, err)
		}
	}
	if len(f.set) != 2 || f.set[0] != 1 || f.set[1] != 5 {
		t.Errorf("stored %v, want [1 5]", f.set)
	}
}
