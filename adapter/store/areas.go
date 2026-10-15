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

func (s *Store) RenameArea(ctx context.Context, userID int64, color, name string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE areas SET name = ? WHERE user_id = ? AND color = ?`, name, userID, color)
	if isUnique(err) {
		return fmt.Errorf("%w: %q", model.ErrTaken, name)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("area %s of user %d: %w", color, userID, model.ErrNotFound)
	}
	return nil
}
