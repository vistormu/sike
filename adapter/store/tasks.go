package store

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"sike/domain/model"
)

// top is the position above every task of a person in one place in time.
const top = `(SELECT COALESCE(MIN(position), 0) - 1 FROM tasks WHERE user_id = :user_id AND "when" = :when)`

func (s *Store) CreateTask(ctx context.Context, t model.Task) (model.Task, error) {
	res, err := s.db.NamedExecContext(ctx,
		`INSERT INTO tasks (user_id, name, color, "when", position) VALUES (:user_id, :name, :color, :when, `+top+`)`, t)
	if err != nil {
		return model.Task{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Task{}, err
	}
	return s.GetTask(ctx, t.UserID, id)
}

func (s *Store) ListTasks(ctx context.Context, userID int64) ([]model.Task, error) {
	tasks := []model.Task{}
	err := s.db.SelectContext(ctx, &tasks,
		`SELECT id, user_id, name, color, "when", position FROM tasks WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
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
	var t model.Task
	err := s.db.GetContext(ctx, &t,
		`SELECT id, user_id, name, color, "when", position FROM tasks WHERE user_id = ? AND id = ?`, userID, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Task{}, fmt.Errorf("task %d: %w", id, model.ErrNotFound)
	}
	return t, err
}

func (s *Store) DeleteTask(ctx context.Context, userID, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE user_id = ? AND id = ?`, userID, id)
	if err != nil {
		return err
	}
	return oneRow(res, "task", id)
}

func (s *Store) MoveTask(ctx context.Context, userID, id int64, when model.When) error {
	res, err := s.db.NamedExecContext(ctx,
		`UPDATE tasks SET "when" = :when, position = `+top+` WHERE user_id = :user_id AND id = :id`,
		map[string]any{"user_id": userID, "id": id, "when": when})
	if err != nil {
		return err
	}
	return oneRow(res, "task", id)
}

// UpdateTask sets name, color and when. a task that changes its when goes on
// top of its new place; sqlite reads the old row in SET, so the CASE sees the
// when before the update.
func (s *Store) UpdateTask(ctx context.Context, t model.Task) (model.Task, error) {
	res, err := s.db.NamedExecContext(ctx,
		`UPDATE tasks SET name = :name, color = :color,
			position = CASE WHEN "when" = :when THEN position ELSE `+top+` END, "when" = :when
		WHERE user_id = :user_id AND id = :id`, t)
	if err != nil {
		return model.Task{}, err
	}
	if err := oneRow(res, "task", t.ID); err != nil {
		return model.Task{}, err
	}
	return s.GetTask(ctx, t.UserID, t.ID)
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
