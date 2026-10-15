package service

import (
	"context"

	"sike/domain/clock"
	"sike/domain/model"
)

type MoodStore interface {
	GetMood(ctx context.Context, userID int64, day clock.Date) (int, error)
	SetMood(ctx context.Context, userID int64, day clock.Date, mood int) error
	ClearMood(ctx context.Context, userID int64, day clock.Date) error
}

// Moods: how a person's day went, one face a day (model.Faces). only today's
// is shown; the rest is never read back (decisions/0001: no history).
type Moods struct {
	store MoodStore
}

func NewMoods(store MoodStore) *Moods {
	return &Moods{store: store}
}

// Get is a day's mood, 0 for none.
func (s *Moods) Get(ctx context.Context, userID int64, day clock.Date) (int, error) {
	return s.store.GetMood(ctx, userID, day)
}

// Set gives a day its mood, 1 to 5.
func (s *Moods) Set(ctx context.Context, userID int64, day clock.Date, mood int) error {
	if !model.IsMood(mood) {
		return model.Invalid("A mood is 1 to %d", len(model.Faces))
	}
	return s.store.SetMood(ctx, userID, day, mood)
}

// Clear takes a day's mood away.
func (s *Moods) Clear(ctx context.Context, userID int64, day clock.Date) error {
	return s.store.ClearMood(ctx, userID, day)
}
