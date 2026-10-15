package service

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"sike/domain/model"
)

type AreaStore interface {
	ListAreas(ctx context.Context, userID int64) ([]model.Area, error)
	RenameArea(ctx context.Context, userID int64, color, name string) error
}

type Areas struct {
	store AreaStore
}

func NewAreas(store AreaStore) *Areas {
	return &Areas{store: store}
}

const maxAreaName = 20

// List returns a person's seven areas, in palette order.
func (s *Areas) List(ctx context.Context, userID int64) ([]model.Area, error) {
	return s.store.ListAreas(ctx, userID)
}

// Rename names one of a person's areas. names are unique per person, ignoring
// case.
func (s *Areas) Rename(ctx context.Context, userID int64, color, name string) error {
	name = strings.TrimSpace(name)
	switch {
	case !model.IsColor(color):
		return model.Invalid("Pick one of %s", strings.Join(model.Colors, ", "))
	case name == "":
		return model.Invalid("An area needs a name")
	case utf8.RuneCountInString(name) > maxAreaName:
		return model.Invalid("The name is longer than %d characters", maxAreaName)
	}
	err := s.store.RenameArea(ctx, userID, color, name)
	if errors.Is(err, model.ErrTaken) {
		return model.Invalid("Another area is already called “%s”", name)
	}
	return err
}
