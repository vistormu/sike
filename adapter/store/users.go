package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"sike/domain/model"
)

func (s *Store) CreateUser(ctx context.Context, u model.User) (model.User, error) {
	res, err := s.db.NamedExecContext(ctx, `INSERT INTO users (name, emoji) VALUES (:name, :emoji)`, u)
	if isUnique(err) {
		return model.User{}, fmt.Errorf("%w: %q", model.ErrTaken, u.Name)
	}
	if err != nil {
		return model.User{}, err
	}
	u.ID, err = res.LastInsertId()
	return u, err
}

func (s *Store) ListUsers(ctx context.Context) ([]model.User, error) {
	users := []model.User{}
	err := s.db.SelectContext(ctx, &users, `SELECT id, name, emoji FROM users ORDER BY id`)
	return users, err
}

func (s *Store) GetUser(ctx context.Context, id int64) (model.User, error) {
	var u model.User
	err := s.db.GetContext(ctx, &u, `SELECT id, name, emoji FROM users WHERE id = ?`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, fmt.Errorf("user %d: %w", id, model.ErrNotFound)
	}
	return u, err
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
