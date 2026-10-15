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
	UpdateUser(ctx context.Context, u model.User) error
	DeleteUser(ctx context.Context, id int64) error
}

type Users struct {
	store UserStore
}

func NewUsers(store UserStore) *Users {
	return &Users{store: store}
}

const maxName = 40

// Add validates and creates a user, with the default areas and deadline
// warning. names are unique, ignoring case; the emoji is one of model.Emojis.
func (s *Users) Add(ctx context.Context, name, emoji string) (model.User, error) {
	u, err := checkUser(model.User{Name: name, Emoji: emoji, Warn: model.DefaultWarn})
	if err != nil {
		return model.User{}, err
	}
	created, err := s.store.CreateUser(ctx, u, model.DefaultAreas)
	if errors.Is(err, model.ErrTaken) {
		return model.User{}, model.Invalid("“%s” is already taken", u.Name)
	}
	return created, err
}

// Edit changes a person's name, emoji and deadline warning (settings), checked
// as Add checks them.
func (s *Users) Edit(ctx context.Context, u model.User) (model.User, error) {
	u, err := checkUser(u)
	if err != nil {
		return model.User{}, err
	}
	err = s.store.UpdateUser(ctx, u)
	if errors.Is(err, model.ErrTaken) {
		return model.User{}, model.Invalid("“%s” is already taken", u.Name)
	}
	return u, err
}

func checkUser(u model.User) (model.User, error) {
	u.Name, u.Emoji = strings.TrimSpace(u.Name), strings.TrimSpace(u.Emoji)
	switch {
	case u.Name == "":
		return u, model.Invalid("A name is required")
	case utf8.RuneCountInString(u.Name) > maxName:
		return u, model.Invalid("The name is longer than %d characters", maxName)
	case !model.IsEmoji(u.Emoji):
		return u, model.Invalid("Pick one of %s", strings.Join(model.Emojis, " "))
	case !model.IsWarning(u.Warn):
		return u, model.Invalid("Warn 1 hour, 1 day or 2 days before deadlines")
	}
	return u, nil
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
