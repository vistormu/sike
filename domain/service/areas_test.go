package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sike/domain/model"
)

type fakeAreas struct{ renamed map[string]string }

func (f *fakeAreas) ListAreas(context.Context, int64) ([]model.Area, error) { return nil, nil }
func (f *fakeAreas) RenameArea(_ context.Context, _ int64, color, name string) error {
	for c, n := range f.renamed {
		if c != color && strings.EqualFold(n, name) {
			return model.ErrTaken
		}
	}
	f.renamed[color] = name
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
			f := &fakeAreas{renamed: map[string]string{"lavender": "Self"}}
			err := NewAreas(f).Rename(context.Background(), 1, tt.color, tt.name)
			if tt.ok {
				if err != nil || f.renamed[tt.color] != tt.want {
					t.Errorf("err = %v, stored %q; want %q", err, f.renamed[tt.color], tt.want)
				}
				return
			}
			if !errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want an invalid error with %q", err, tt.want)
			}
		})
	}
}
