package service

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"sike/domain/model"
)

type TaskStore interface {
	CreateTask(ctx context.Context, t model.Task) (model.Task, error)
	ListTasks(ctx context.Context, userID int64) ([]model.Task, error)
	GetTask(ctx context.Context, userID, id int64) (model.Task, error)
	DeleteTask(ctx context.Context, userID, id int64) error
	GetDeletedTask(ctx context.Context, userID, id int64) (model.Task, error)
	RestoreTask(ctx context.Context, userID, id int64) error
	MoveTask(ctx context.Context, userID, id int64, when model.When) error
	UpdateTask(ctx context.Context, t model.Task) (model.Task, error)
}

type Tasks struct {
	store TaskStore
}

func NewTasks(store TaskStore) *Tasks {
	return &Tasks{store: store}
}

const maxTaskName = 80

// whens are the places a task can be put, for now.
var whens = []model.When{model.Pending, model.Someday}

// Add creates a task on top of its place.
func (s *Tasks) Add(ctx context.Context, userID int64, name, color string, when model.When) (model.Task, error) {
	t, err := check(model.Task{UserID: userID, Name: name, Color: color, When: when})
	if err != nil {
		return model.Task{}, err
	}
	return s.store.CreateTask(ctx, t)
}

// Edit changes a task's name, area and when. a new when moves it on top of
// its new place (decisions/0005); the same when keeps its position.
func (s *Tasks) Edit(ctx context.Context, userID, id int64, name, color string, when model.When) (model.Task, error) {
	t, err := check(model.Task{ID: id, UserID: userID, Name: name, Color: color, When: when})
	if err != nil {
		return model.Task{}, err
	}
	return s.store.UpdateTask(ctx, t)
}

// check validates a task as typed, its name trimmed.
func check(t model.Task) (model.Task, error) {
	t.Name = strings.TrimSpace(t.Name)
	switch {
	case t.Name == "":
		return t, model.Invalid("A task needs a name")
	case utf8.RuneCountInString(t.Name) > maxTaskName:
		return t, model.Invalid("The name is longer than %d characters", maxTaskName)
	case !model.IsColor(t.Color):
		return t, model.Invalid("Pick an area")
	}
	return t, checkWhen(t.When)
}

// Whens are the places a task can be put, in tab order: the task form offers
// these.
func (s *Tasks) Whens() []model.When { return slices.Clone(whens) }

// List returns a person's tasks: by when, then area (palette order), then
// position.
func (s *Tasks) List(ctx context.Context, userID int64) ([]model.Task, error) {
	return s.store.ListTasks(ctx, userID)
}

func (s *Tasks) Get(ctx context.Context, userID, id int64) (model.Task, error) {
	return s.store.GetTask(ctx, userID, id)
}

// Remove deletes a task: done (decisions/0001). it can be undone until the
// person's next delete.
func (s *Tasks) Remove(ctx context.Context, userID, id int64) error {
	return s.store.DeleteTask(ctx, userID, id)
}

// Deleted is a removed task that can still be undone.
func (s *Tasks) Deleted(ctx context.Context, userID, id int64) (model.Task, error) {
	return s.store.GetDeletedTask(ctx, userID, id)
}

// Restore undoes Remove: the task is back where it was.
func (s *Tasks) Restore(ctx context.Context, userID, id int64) error {
	return s.store.RestoreTask(ctx, userID, id)
}

// Move puts a task in another place in time, on top there.
func (s *Tasks) Move(ctx context.Context, userID, id int64, when model.When) error {
	if err := checkWhen(when); err != nil {
		return err
	}
	return s.store.MoveTask(ctx, userID, id, when)
}

func checkWhen(w model.When) error {
	for _, ok := range whens {
		if w == ok {
			return nil
		}
	}
	return model.Invalid("When is pending or someday")
}
