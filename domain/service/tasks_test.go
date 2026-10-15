package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"sike/domain/model"
)

type fakeTasks struct {
	created []model.Task
	moved   map[int64]model.When
	updated []model.Task
}

func (f *fakeTasks) CreateTask(_ context.Context, t model.Task) (model.Task, error) {
	f.created = append(f.created, t)
	return t, nil
}
func (f *fakeTasks) ListTasks(context.Context, int64) ([]model.Task, error) { return nil, nil }
func (f *fakeTasks) GetTask(context.Context, int64, int64) (model.Task, error) {
	return model.Task{}, nil
}
func (f *fakeTasks) DeleteTask(context.Context, int64, int64) error { return nil }
func (f *fakeTasks) UpdateTask(_ context.Context, t model.Task) (model.Task, error) {
	f.updated = append(f.updated, t)
	return t, nil
}
func (f *fakeTasks) MoveTask(_ context.Context, _, id int64, when model.When) error {
	f.moved[id] = when
	return nil
}

func TestAddTask(t *testing.T) {
	tests := []struct {
		name, color string
		when        model.When
		err         string // "" = ok
	}{
		{"Fix bike", "sky", model.Pending, ""},
		{"  Learn piano  ", "lavender", model.Someday, ""},
		{strings.Repeat("a", 80), "sky", model.Pending, ""},
		{strings.Repeat("a", 81), "sky", model.Pending, "longer than 80"},
		{"   ", "sky", model.Pending, "needs a name"},
		{"Fix bike", "beige", model.Pending, "Pick an area"},
		{"Fix bike", "sky", model.Weekly, "pending or someday"}, // m15
		{"Fix bike", "sky", model.Dated, "pending or someday"},  // m18
		{"Fix bike", "sky", "later", "pending or someday"},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+string(tt.when), func(t *testing.T) {
			f := &fakeTasks{}
			task, err := NewTasks(f).Add(context.Background(), 1, tt.name, tt.color, tt.when)
			if tt.err == "" {
				if err != nil || task.Name != strings.TrimSpace(tt.name) || len(f.created) != 1 {
					t.Errorf("got %+v, %v", task, err)
				}
				return
			}
			if !errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), tt.err) || len(f.created) != 0 {
				t.Errorf("err = %v, want an invalid error with %q and nothing stored", err, tt.err)
			}
		})
	}
}

func TestMoveTask(t *testing.T) {
	f := &fakeTasks{moved: map[int64]model.When{}}
	s := NewTasks(f)
	if err := s.Move(context.Background(), 1, 7, model.Someday); err != nil || f.moved[7] != model.Someday {
		t.Errorf("move to someday: %v, moved %v", err, f.moved)
	}
	if err := s.Move(context.Background(), 1, 8, model.Weekly); !errors.Is(err, model.ErrInvalid) || len(f.moved) != 1 {
		t.Errorf("move to weekly (not yet): err = %v", err)
	}
}

func TestTaskWhens(t *testing.T) {
	s := NewTasks(&fakeTasks{})
	got := s.Whens()
	if len(got) != 2 || got[0] != model.Pending || got[1] != model.Someday {
		t.Fatalf("Whens = %v, want [pending someday]", got)
	}
	got[0] = model.Weekly
	if s.Whens()[0] != model.Pending {
		t.Error("Whens hands out the service's own slice")
	}
}

func TestEditTask(t *testing.T) {
	f := &fakeTasks{}
	s := NewTasks(f)
	got, err := s.Edit(context.Background(), 1, 7, "  Fix bike ", "mint", model.Someday)
	if err != nil || got.ID != 7 || got.UserID != 1 || got.Name != "Fix bike" || got.Color != "mint" || got.When != model.Someday {
		t.Errorf("edit: %+v, %v", got, err)
	}
	// the same checks as Add
	for _, bad := range []struct {
		name, color string
		when        model.When
	}{{"", "mint", model.Pending}, {"x", "beige", model.Pending}, {"x", "mint", model.Weekly}} {
		if _, err := s.Edit(context.Background(), 1, 7, bad.name, bad.color, bad.when); !errors.Is(err, model.ErrInvalid) {
			t.Errorf("edit %+v: err = %v, want invalid", bad, err)
		}
	}
	if len(f.updated) != 1 {
		t.Errorf("%d updates stored, want 1", len(f.updated))
	}
}
