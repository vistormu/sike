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

	fox, err := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊"}, nil)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	panda, err := s.CreateUser(ctx, model.User{Name: "Partner", Emoji: "🐼"}, nil)
	if err != nil {
		t.Fatalf("CreateUser: %v", err)
	}
	if fox.ID == 0 || panda.ID == fox.ID {
		t.Fatalf("ids = %d, %d: want two distinct ids", fox.ID, panda.ID)
	}

	if _, err := s.CreateUser(ctx, model.User{Name: "you", Emoji: "🐸"}, nil); !errors.Is(err, model.ErrTaken) {
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
