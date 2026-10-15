package store

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/jmoiron/sqlx"

	"sike/domain/model"
)

// top is the position above every task of a person in one place in time.
const top = `(SELECT COALESCE(MIN(position), 0) - 1 FROM tasks WHERE user_id = :user_id AND "when" = :when AND NOT deleted)`

// CreateTask adds a task on top of its place, with its placements (weekly) or
// dates (dated), in one transaction.
func (s *Store) CreateTask(ctx context.Context, t model.Task) (model.Task, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.Task{}, err
	}
	defer tx.Rollback()
	res, err := tx.NamedExecContext(ctx,
		`INSERT INTO tasks (user_id, name, color, "when", position) VALUES (:user_id, :name, :color, :when, `+top+`)`, t)
	if err != nil {
		return model.Task{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Task{}, err
	}
	if err := insertPlacements(ctx, tx, id, t.Places); err != nil {
		return model.Task{}, err
	}
	if err := insertDates(ctx, tx, id, t.Date); err != nil {
		return model.Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Task{}, err
	}
	return s.GetTask(ctx, t.UserID, id)
}

func (s *Store) ListTasks(ctx context.Context, userID int64) ([]model.Task, error) {
	tasks := []model.Task{}
	err := s.db.SelectContext(ctx, &tasks,
		`SELECT id, user_id, name, color, "when", position FROM tasks WHERE user_id = ? AND NOT deleted`, userID)
	if err != nil {
		return nil, err
	}
	dates, err := s.datesOf(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].Date = dates[tasks[i].ID]
	}
	// place in time (tab order), then area (palette order), then position
	slices.SortFunc(tasks, func(a, b model.Task) int {
		return cmp.Or(
			cmp.Compare(slices.Index(model.Whens, a.When), slices.Index(model.Whens, b.When)),
			cmp.Compare(slices.Index(model.Colors, a.Color), slices.Index(model.Colors, b.Color)),
			cmp.Compare(a.Position, b.Position),
		)
	})
	return tasks, nil
}

func (s *Store) GetTask(ctx context.Context, userID, id int64) (model.Task, error) {
	return s.getTask(ctx, userID, id, false)
}

// GetDeletedTask is a task waiting for its undo.
func (s *Store) GetDeletedTask(ctx context.Context, userID, id int64) (model.Task, error) {
	return s.getTask(ctx, userID, id, true)
}

func (s *Store) getTask(ctx context.Context, userID, id int64, deleted bool) (model.Task, error) {
	var t model.Task
	err := s.db.GetContext(ctx, &t,
		`SELECT id, user_id, name, color, "when", position FROM tasks WHERE user_id = ? AND id = ? AND deleted = ?`, userID, id, deleted)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, fmt.Errorf("task %d: %w", id, model.ErrNotFound)
	}
	if err != nil {
		return model.Task{}, err
	}
	if err := s.db.SelectContext(ctx, &t.Places,
		`SELECT `+placementCols+` FROM placements p WHERE p.task_id = ? ORDER BY p.start, p.id`, id); err != nil {
		return model.Task{}, err
	}
	var rows []dateRow
	if err := s.db.SelectContext(ctx, &rows, `SELECT `+dateCols+` FROM task_dates d WHERE d.task_id = ?`, id); err != nil {
		return model.Task{}, err
	}
	for _, r := range rows {
		d, err := r.dates()
		if err != nil {
			return model.Task{}, err
		}
		t.Date = &d
	}
	return t, nil
}

// DeleteTask hides a task until its undo, and erases for good the person's
// delete before it: only the last one can be undone.
func (s *Store) DeleteTask(ctx context.Context, userID, id int64) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM tasks WHERE user_id = ? AND deleted`, userID); err != nil {
		return err
	}
	res, err := tx.ExecContext(ctx, `UPDATE tasks SET deleted = 1 WHERE user_id = ? AND id = ? AND NOT deleted`, userID, id)
	if err != nil {
		return err
	}
	if err := oneRow(res, "task", id); err != nil {
		return err
	}
	return tx.Commit()
}

// RestoreTask undoes the delete of a task, back in its place. ErrNotFound once
// a later delete erased it.
func (s *Store) RestoreTask(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `UPDATE tasks SET deleted = 0 WHERE user_id = ? AND id = ? AND deleted`, userID, id)
	if err != nil {
		return err
	}
	return oneRow(res, "task", id)
}

// MoveTask puts a task on top of another place in time (pending or someday:
// the service checks), dropping its placements and dates.
func (s *Store) MoveTask(ctx context.Context, userID, id int64, when model.When) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.NamedExecContext(ctx,
		`UPDATE tasks SET "when" = :when, position = `+top+` WHERE user_id = :user_id AND id = :id AND NOT deleted`,
		map[string]any{"user_id": userID, "id": id, "when": when})
	if err != nil {
		return err
	}
	if err := oneRow(res, "task", id); err != nil {
		return err
	}
	if err := dropTimes(ctx, tx, id); err != nil {
		return err
	}
	return tx.Commit()
}

// UpdateTask sets name, color and when, and replaces the task's placements and
// dates. a task that changes its when goes on top of its new place; sqlite
// reads the old row in SET, so the CASE sees the when before the update.
func (s *Store) UpdateTask(ctx context.Context, t model.Task) (model.Task, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.Task{}, err
	}
	defer tx.Rollback()
	res, err := tx.NamedExecContext(ctx,
		`UPDATE tasks SET name = :name, color = :color,
			position = CASE WHEN "when" = :when THEN position ELSE `+top+` END, "when" = :when
		WHERE user_id = :user_id AND id = :id AND NOT deleted`, t)
	if err != nil {
		return model.Task{}, err
	}
	if err := oneRow(res, "task", t.ID); err != nil {
		return model.Task{}, err
	}
	if err := dropTimes(ctx, tx, t.ID); err != nil {
		return model.Task{}, err
	}
	if err := insertPlacements(ctx, tx, t.ID, t.Places); err != nil {
		return model.Task{}, err
	}
	if err := insertDates(ctx, tx, t.ID, t.Date); err != nil {
		return model.Task{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.Task{}, err
	}
	return s.GetTask(ctx, t.UserID, t.ID)
}

// dropTimes removes a task's placements and dates.
func dropTimes(ctx context.Context, tx *sqlx.Tx, taskID int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM placements WHERE task_id = ?`, taskID); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `DELETE FROM task_dates WHERE task_id = ?`, taskID)
	return err
}

// oneRow turns "no row touched" into ErrNotFound.
func oneRow(res sql.Result, what string, id int64) error {
	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return fmt.Errorf("%s %d: %w", what, id, model.ErrNotFound)
	}
	return nil
}
