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

func (f *fakeUsers) CreateUser(_ context.Context, u model.User, _ []model.Area) (model.User, error) {
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
func (f *fakeUsers) UpdateUser(_ context.Context, u model.User) error {
	for _, c := range f.created {
		if c.ID != u.ID && strings.EqualFold(c.Name, u.Name) {
			return model.ErrTaken
		}
	}
	f.created[u.ID-1] = u
	return nil
}

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

func TestEditUser(t *testing.T) {
	f := &fakeUsers{}
	s := NewUsers(f)
	you, _ := s.Add(context.Background(), "You", "🦊")
	ana, _ := s.Add(context.Background(), "Ana", "🐱")
	if you.Warn != model.DefaultWarn {
		t.Errorf("a new person warns %d minutes before, want %d", you.Warn, model.DefaultWarn)
	}
	tests := []struct {
		u    model.User
		want string // "" ok, else a piece of the error
	}{
		{model.User{ID: you.ID, Name: " Vic ", Emoji: "🐼", Warn: 60}, ""},
		{model.User{ID: you.ID, Name: "Vic", Emoji: "🐼", Warn: 2880}, ""},
		{model.User{ID: you.ID, Name: "Vic", Emoji: "🐼", Warn: 30}, "1 hour, 1 day or 2 days"},
		{model.User{ID: you.ID, Name: "", Emoji: "🐼", Warn: 60}, "name is required"},
		{model.User{ID: you.ID, Name: "Vic", Emoji: "🐶", Warn: 60}, "Pick one of"},
		{model.User{ID: you.ID, Name: "ana", Emoji: "🐼", Warn: 60}, "“ana” is already taken"},
		{model.User{ID: ana.ID, Name: "ANA", Emoji: "🐱", Warn: 60}, ""}, // her own name, recased
	}
	for _, tt := range tests {
		_, err := s.Edit(context.Background(), tt.u)
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%+v: err = %v", tt.u, err)
		case tt.want != "" && (!errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), tt.want)):
			t.Errorf("%+v: err = %v, want %q", tt.u, err, tt.want)
		}
	}
	if got := f.created[0]; got.Name != "Vic" || got.Emoji != "🐼" || got.Warn != 2880 {
		t.Errorf("stored %+v, want Vic 🐼 2880", got)
	}
}
