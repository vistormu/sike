package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sike/domain/model"
)

// fakeUsers stores what it is given; validation is the service's job. like
// the real store, a name already there (any case) is ErrTaken.
type fakeUsers struct{ created []model.User }

func (f *fakeUsers) CreateUser(_ context.Context, u model.User) (model.User, error) {
	for _, c := range f.created {
		if strings.EqualFold(c.Name, u.Name) {
			return model.User{}, model.ErrTaken
		}
	}
	u.ID = int64(len(f.created) + 1)
	f.created = append(f.created, u)
	return u, nil
}
func (f *fakeUsers) ListUsers(context.Context) ([]model.User, error)    { return f.created, nil }
func (f *fakeUsers) GetUser(context.Context, int64) (model.User, error) { return model.User{}, nil }
func (f *fakeUsers) DeleteUser(context.Context, int64) error            { return nil }

func TestAddUser(t *testing.T) {
	tests := []struct {
		name, emoji string
		ok          bool
		wantName    string
		wantEmoji   string
	}{
		{"You", "🦊", true, "You", "🦊"},
		{"  You  ", " 🦊 ", true, "You", "🦊"},
		{"Ana María", "🐱", true, "Ana María", "🐱"},
		{"🇪🇸 Flag", "🐼", true, "🇪🇸 Flag", "🐼"},
		{"", "🦊", false, "", ""},
		{"   ", "🦊", false, "", ""},
		{strings.Repeat("a", 40), "🦊", true, strings.Repeat("a", 40), "🦊"},
		{strings.Repeat("a", 41), "🦊", false, "", ""},
		{"You", "", false, "", ""},
		{"You", "🦊 🐼", false, "", ""},
		{"You", "🐶", false, "", ""}, // an emoji, not one of the set
		{"You", "x", false, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.emoji, func(t *testing.T) {
			s := NewUsers(&fakeUsers{})
			u, err := s.Add(context.Background(), tt.name, tt.emoji)
			if !tt.ok {
				if !errors.Is(err, model.ErrInvalid) {
					t.Fatalf("err = %v, want ErrInvalid", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("err = %v", err)
			}
			if u.Name != tt.wantName || u.Emoji != tt.wantEmoji {
				t.Errorf("got %q %q, want %q %q", u.Name, u.Emoji, tt.wantName, tt.wantEmoji)
			}
		})
	}
}

func TestAddUserTakenNameReadsWell(t *testing.T) {
	s := NewUsers(&fakeUsers{})
	if _, err := s.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	_, err := s.Add(context.Background(), "you", "🐼")
	if !errors.Is(err, model.ErrInvalid) || err.Error() != "“you” is already taken" {
		t.Errorf("err = %q, want an invalid “you” is already taken", err)
	}
}
