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
	RenameAreas(ctx context.Context, userID int64, names map[string]string) error
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

// Rename names one of a person's areas.
func (s *Areas) Rename(ctx context.Context, userID int64, color, name string) error {
	return s.RenameAll(ctx, userID, map[string]string{color: name})
}

// RenameAll names several of a person's areas (color → name) at once. names
// are unique per person, ignoring case, and checked as they end up: two areas
// may trade names, and what is refused changes none (settings posts them all).
func (s *Areas) RenameAll(ctx context.Context, userID int64, names map[string]string) error {
	areas, err := s.store.ListAreas(ctx, userID)
	if err != nil {
		return err
	}
	final := map[string]string{}
	for _, a := range areas {
		final[a.Color] = a.Name
	}
	changed := map[string]string{}
	for color, name := range names {
		name = strings.TrimSpace(name)
		switch {
		case !model.IsColor(color):
			return model.Invalid("Pick one of %s", strings.Join(model.Colors, ", "))
		case name == "":
			return model.Invalid("An area needs a name")
		case utf8.RuneCountInString(name) > maxAreaName:
			return model.Invalid("The name is longer than %d characters", maxAreaName)
		}
		if final[color] != name {
			final[color], changed[color] = name, name
		}
	}
	for _, color := range model.Colors {
		name, ok := changed[color]
		if !ok {
			continue
		}
		for other, n := range final {
			if other != color && strings.EqualFold(n, name) {
				return model.Invalid("Another area is already called “%s”", name)
			}
		}
	}
	if len(changed) == 0 {
		return nil
	}
	err = s.store.RenameAreas(ctx, userID, changed)
	if errors.Is(err, model.ErrTaken) {
		return model.Invalid("Another area is already called that")
	}
	return err
}
