package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sike/domain/clock"
	"sike/domain/model"
)

// GetMood is a person's mood on a day, 0 if they gave none.
func (s *Store) GetMood(ctx context.Context, userID int64, day clock.Date) (int, error) {
	var mood int
	err := s.db.GetContext(ctx, &mood, `SELECT mood FROM moods WHERE user_id = ? AND date = ?`, userID, day.String())
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return mood, err
}

// SetMood gives a person's day its mood, over the one it had.
func (s *Store) SetMood(ctx context.Context, userID int64, day clock.Date, mood int) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO moods (user_id, date, mood) VALUES (?, ?, ?)
		ON CONFLICT (user_id, date) DO UPDATE SET mood = excluded.mood`, userID, day.String(), mood)
	if isForeignKey(err) {
		return fmt.Errorf("user %d: %w", userID, model.ErrNotFound)
	}
	return err
}

// ClearMood takes a day's mood away; a day with none stays so.
func (s *Store) ClearMood(ctx context.Context, userID int64, day clock.Date) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM moods WHERE user_id = ? AND date = ?`, userID, day.String())
	return err
}
