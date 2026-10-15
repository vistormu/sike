package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"

	"sike/domain/model"
)

const placementCols = `p.id, p.task_id, p.days, p.start, p."end", p.notify_start, p.notify_end`

func insertPlacements(ctx context.Context, tx *sqlx.Tx, taskID int64, places []model.Placement) error {
	for _, p := range places {
		p.TaskID = taskID
		if _, err := tx.NamedExecContext(ctx,
			`INSERT INTO placements (task_id, days, start, "end", notify_start, notify_end)
			VALUES (:task_id, :days, :start, :end, :notify_start, :notify_end)`, p); err != nil {
			return err
		}
	}
	return nil
}

// ListPlacements is every placement of a person's tasks (not deleted), by
// task, then start: the week reads it.
func (s *Store) ListPlacements(ctx context.Context, userID int64) ([]model.Placement, error) {
	places := []model.Placement{}
	err := s.db.SelectContext(ctx, &places,
		`SELECT `+placementCols+` FROM placements p JOIN tasks t ON t.id = p.task_id
		WHERE t.user_id = ? AND NOT t.deleted ORDER BY p.task_id, p.start, p.id`, userID)
	return places, err
}

// TaskPlacements is one task's placements, by start.
func (s *Store) TaskPlacements(ctx context.Context, userID, taskID int64) ([]model.Placement, error) {
	places := []model.Placement{}
	err := s.db.SelectContext(ctx, &places,
		`SELECT `+placementCols+` FROM placements p JOIN tasks t ON t.id = p.task_id
		WHERE t.user_id = ? AND t.id = ? AND NOT t.deleted ORDER BY p.start, p.id`, userID, taskID)
	return places, err
}

func (s *Store) GetPlacement(ctx context.Context, userID, id int64) (model.Placement, error) {
	var p model.Placement
	err := s.db.GetContext(ctx, &p,
		`SELECT `+placementCols+` FROM placements p JOIN tasks t ON t.id = p.task_id
		WHERE t.user_id = ? AND p.id = ? AND NOT t.deleted`, userID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Placement{}, fmt.Errorf("placement %d: %w", id, model.ErrNotFound)
	}
	return p, err
}

// AddPlacement adds a time to a person's weekly task.
func (s *Store) AddPlacement(ctx context.Context, userID int64, p model.Placement) (model.Placement, error) {
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO placements (task_id, days, start, "end", notify_start, notify_end)
		SELECT id, ?, ?, ?, ?, ? FROM tasks WHERE id = ? AND user_id = ? AND "when" = 'weekly' AND NOT deleted`,
		p.Days, p.Start, p.End, p.NotifyStart, p.NotifyEnd, p.TaskID, userID)
	if err != nil {
		return model.Placement{}, err
	}
	if err := oneRow(res, "weekly task", p.TaskID); err != nil {
		return model.Placement{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Placement{}, err
	}
	return s.GetPlacement(ctx, userID, id)
}

func (s *Store) DeletePlacement(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM placements WHERE id = ? AND task_id IN (SELECT id FROM tasks WHERE user_id = ? AND NOT deleted)`, id, userID)
	if err != nil {
		return err
	}
	return oneRow(res, "placement", id)
}
