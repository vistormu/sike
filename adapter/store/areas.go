package store

import (
	"context"
	"fmt"
	"slices"

	"sike/domain/model"
)

func (s *Store) ListAreas(ctx context.Context, userID int64) ([]model.Area, error) {
	areas := []model.Area{}
	if err := s.db.SelectContext(ctx, &areas, `SELECT user_id, color, name FROM areas WHERE user_id = ?`, userID); err != nil {
		return nil, err
	}
	if len(areas) == 0 {
		return nil, fmt.Errorf("user %d: %w", userID, model.ErrNotFound)
	}
	slices.SortFunc(areas, func(a, b model.Area) int {
		return slices.Index(model.Colors, a.Color) - slices.Index(model.Colors, b.Color)
	})
	return areas, nil
}

// RenameAreas renames a person's areas (color → name), all or none. each
// first takes a stand-in name (" sky": names are trimmed, no one types it), so
// two may trade names past the unique index.
func (s *Store) RenameAreas(ctx context.Context, userID int64, names map[string]string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for color := range names {
		res, err := tx.ExecContext(ctx, `UPDATE areas SET name = ? WHERE user_id = ? AND color = ?`, " "+color, userID, color)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("area %s of user %d: %w", color, userID, model.ErrNotFound)
		}
	}
	for color, name := range names {
		_, err := tx.ExecContext(ctx, `UPDATE areas SET name = ? WHERE user_id = ? AND color = ?`, name, userID, color)
		if isUnique(err) {
			return fmt.Errorf("%w: %q", model.ErrTaken, name)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}
