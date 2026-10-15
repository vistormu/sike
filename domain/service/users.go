// Package service holds the use cases. it talks to storage through the
// interfaces it declares, and knows nothing of sql or http.
package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"sike/domain/model"
)

type UserStore interface {
	// CreateUser creates the person with their areas, atomically.
	CreateUser(ctx context.Context, u model.User, areas []model.Area) (model.User, error)
	ListUsers(ctx context.Context) ([]model.User, error)
	GetUser(ctx context.Context, id int64) (model.User, error)
	DeleteUser(ctx context.Context, id int64) error
}

type Users struct {
	store UserStore
}

func NewUsers(store UserStore) *Users {
	return &Users{store: store}
}

const maxName = 40

// Add validates and creates a user, with the default areas. names are unique,
// ignoring case; the emoji is one of model.Emojis.
func (s *Users) Add(ctx context.Context, name, emoji string) (model.User, error) {
	name, emoji = strings.TrimSpace(name), strings.TrimSpace(emoji)
	switch {
	case name == "":
		return model.User{}, model.Invalid("A name is required")
	case utf8.RuneCountInString(name) > maxName:
		return model.User{}, model.Invalid("The name is longer than %d characters", maxName)
	case !model.IsEmoji(emoji):
		return model.User{}, model.Invalid("Pick one of %s", strings.Join(model.Emojis, " "))
	}
	u, err := s.store.CreateUser(ctx, model.User{Name: name, Emoji: emoji}, model.DefaultAreas)
	if errors.Is(err, model.ErrTaken) {
		return model.User{}, model.Invalid("“%s” is already taken", name)
	}
	return u, err
}

// List returns every user, oldest first.
func (s *Users) List(ctx context.Context) ([]model.User, error) {
	return s.store.ListUsers(ctx)
}

func (s *Users) Get(ctx context.Context, id int64) (model.User, error) {
	return s.store.GetUser(ctx, id)
}

// Remove deletes a user and, through the schema, everything theirs.
func (s *Users) Remove(ctx context.Context, id int64) error {
	return s.store.DeleteUser(ctx, id)
}
