package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sike/domain/model"
)

// fakeAreas keeps one person's areas, color → name, as the store's unique
// index would: a name taken mid-rename fails.
type fakeAreas struct{ names map[string]string }

func newFakeAreas() *fakeAreas {
	f := &fakeAreas{names: map[string]string{}}
	for _, a := range model.DefaultAreas {
		f.names[a.Color] = a.Name
	}
	return f
}

func (f *fakeAreas) ListAreas(context.Context, int64) ([]model.Area, error) {
	var areas []model.Area
	for _, c := range model.Colors {
		areas = append(areas, model.Area{Color: c, Name: f.names[c]})
	}
	return areas, nil
}

func (f *fakeAreas) RenameAreas(_ context.Context, _ int64, names map[string]string) error {
	for color, name := range names {
		f.names[color] = name
	}
	return nil
}

func TestRenameArea(t *testing.T) {
	tests := []struct {
		color, name string
		ok          bool
		want        string // the stored name, or a piece of the error
	}{
		{"sky", "Job", true, "Job"},
		{"sky", "  Job  ", true, "Job"},
		{"sky", strings.Repeat("a", 20), true, strings.Repeat("a", 20)},
		{"sky", strings.Repeat("a", 21), false, "longer than 20"},
		{"sky", "  ", false, "needs a name"},
		{"beige", "Job", false, "Pick one of"},
		{"mint", "self", false, "already called “self”"},
	}
	for _, tt := range tests {
		t.Run(tt.color+"/"+tt.name, func(t *testing.T) {
			f := newFakeAreas()
			err := NewAreas(f).Rename(context.Background(), 1, tt.color, tt.name)
			if tt.ok {
				if err != nil || f.names[tt.color] != tt.want {
					t.Errorf("err = %v, stored %q; want %q", err, f.names[tt.color], tt.want)
				}
				return
			}
			if !errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want an invalid error with %q", err, tt.want)
			}
		})
	}
}

// settings posts every area at once, as typed: checked as they end up.
func TestRenameAll(t *testing.T) {
	tests := []struct {
		name  string
		names map[string]string
		ok    bool
		want  map[string]string // stored after, for the colors named
	}{
		{"unchanged", map[string]string{"lavender": "Self", "sky": "Work"}, true, map[string]string{"lavender": "Self"}},
		{"two trade names", map[string]string{"lavender": "Work", "sky": "Self"}, true, map[string]string{"lavender": "Work", "sky": "Self"}},
		{"typed top to bottom: the first takes the second's name", map[string]string{"lavender": "Work", "sky": "Work"}, false, map[string]string{"lavender": "Self", "sky": "Work"}},
		{"then the second renamed too", map[string]string{"lavender": "Work", "sky": "Job"}, true, map[string]string{"lavender": "Work", "sky": "Job"}},
		{"one bad, none stored", map[string]string{"lavender": "Mind", "sky": " "}, false, map[string]string{"lavender": "Self"}},
		{"a name kept by another", map[string]string{"mint": "self"}, false, map[string]string{"mint": "Home"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeAreas()
			err := NewAreas(f).RenameAll(context.Background(), 1, tt.names)
			if tt.ok != (err == nil) {
				t.Errorf("err = %v, want ok %v", err, tt.ok)
			}
			if err != nil && !errors.Is(err, model.ErrInvalid) {
				t.Errorf("err = %v, want an invalid error", err)
			}
			for color, want := range tt.want {
				if f.names[color] != want {
					t.Errorf("%s = %q, want %q", color, f.names[color], want)
				}
			}
		})
	}
}
