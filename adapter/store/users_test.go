package store

import (
	"context"
	"errors"
	"testing"

	"sike/domain/model"
)

func TestUsers(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))

	fox, err := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, nil)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	panda, err := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼", Warn: model.DefaultWarn}, nil)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if fox.ID == 0 || panda.ID == fox.ID {
		t.Fatalf("ids = %d, %d: want two distinct ids", fox.ID, panda.ID)
	}

	if _, err := s.CreateUser(ctx, model.User{Name: "you", Emoji: "🐸", Warn: model.DefaultWarn}, nil); !errors.Is(err, model.ErrTaken) {
		t.Errorf("same name, other case: err = %v, want ErrTaken", err)
	}

	got, err := s.GetUser(ctx, panda.ID)
	if err != nil || got != panda {
		t.Errorf("GetUser = %+v, %v; want %+v", got, err, panda)
	}

	list, err := s.ListUsers(ctx)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(list) != 2 || list[0] != fox || list[1] != panda {
		t.Errorf("ListUsers = %+v, want [fox panda] in creation order", list)
	}

	if err := s.DeleteUser(ctx, fox.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if _, err := s.GetUser(ctx, fox.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("GetUser after delete: err = %v, want ErrNotFound", err)
	}
	if err := s.DeleteUser(ctx, fox.ID); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("DeleteUser twice: err = %v, want ErrNotFound", err)
	}
}

func TestListUsersEmpty(t *testing.T) {
	list, err := New(testDB(t)).ListUsers(context.Background())
	if err != nil || list == nil || len(list) != 0 {
		t.Errorf("ListUsers = %#v, %v; want an empty, non-nil slice", list, err)
	}
}

func TestUpdateUser(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	fox, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, nil)
	s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼", Warn: model.DefaultWarn}, nil)

	fox.Name, fox.Emoji, fox.Warn, fox.Topic = "Vic", "🐸", 60, "sike-vic"
	if err := s.UpdateUser(ctx, fox); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetUser(ctx, fox.ID); got != fox {
		t.Errorf("GetUser = %+v, want %+v", got, fox)
	}
	if err := s.UpdateUser(ctx, model.User{ID: fox.ID, Name: "partner", Emoji: "🐸", Warn: 60}); !errors.Is(err, model.ErrTaken) {
		t.Errorf("another's name: err = %v, want ErrTaken", err)
	}
	if err := s.UpdateUser(ctx, model.User{ID: fox.ID, Name: "Vic", Emoji: "🐸", Warn: 5}); err == nil {
		t.Error("a 5 minute warning was stored")
	}
	if err := s.UpdateUser(ctx, model.User{ID: 99, Name: "Nobody", Emoji: "🐸", Warn: 60}); !errors.Is(err, model.ErrNotFound) {
		t.Errorf("nobody: err = %v, want ErrNotFound", err)
	}
}
