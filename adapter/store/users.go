package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sike/domain/model"
)

// CreateUser creates a person and their areas, together or not at all.
func (s *Store) CreateUser(ctx context.Context, u model.User, areas []model.Area) (model.User, error) {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return model.User{}, err
	}
	defer tx.Rollback()

	res, err := tx.NamedExecContext(ctx, `INSERT INTO users (name, emoji, warn, topic) VALUES (:name, :emoji, :warn, :topic)`, u)
	if isUnique(err) {
		return model.User{}, fmt.Errorf("%w: %q", model.ErrTaken, u.Name)
	}
	if err != nil {
		return model.User{}, err
	}
	if u.ID, err = res.LastInsertId(); err != nil {
		return model.User{}, err
	}
	for _, a := range areas {
		a.UserID = u.ID
		if _, err := tx.NamedExecContext(ctx, `INSERT INTO areas (user_id, color, name) VALUES (:user_id, :color, :name)`, a); err != nil {
			return model.User{}, err
		}
	}
	return u, tx.Commit()
}

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	users := []model.User{}
	err := s.db.SelectContext(ctx, &users, `SELECT id, name, emoji, warn, topic FROM users ORDER BY id`)
	return users, err
}

func (s *Store) GetUser(ctx context.Context, id int64) (model.User, error) {
	var u model.User
	err := s.db.GetContext(ctx, &u, `SELECT id, name, emoji, warn, topic FROM users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("user %d: %w", id, model.ErrNotFound)
	}
	return u, err
}

// UpdateUser changes a person's name, emoji, deadline warning and topic.
func (s *Store) UpdateUser(ctx context.Context, u model.User) error {
	res, err := s.db.NamedExecContext(ctx, `UPDATE users SET name = :name, emoji = :emoji, warn = :warn, topic = :topic WHERE id = :id`, u)
	if isUnique(err) {
		return fmt.Errorf("%w: %q", model.ErrTaken, u.Name)
	}
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("user %d: %w", u.ID, model.ErrNotFound)
	}
	return nil
}

func (s *Store) DeleteUser(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("user %d: %w", id, model.ErrNotFound)
	}
	return nil
}
