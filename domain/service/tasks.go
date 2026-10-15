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
	MoveTask(ctx context.Context, userID, id int64, when model.When) error
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
	name = strings.TrimSpace(name)
	switch {
	case name == "":
		return model.Task{}, model.Invalid("A task needs a name")
	case utf8.RuneCountInString(name) > maxTaskName:
		return model.Task{}, model.Invalid("The name is longer than %d characters", maxTaskName)
	case !model.IsColor(color):
		return model.Task{}, model.Invalid("Pick an area")
	}
	if err := checkWhen(when); err != nil {
		return model.Task{}, err
	}
	return s.store.CreateTask(ctx, model.Task{UserID: userID, Name: name, Color: color, When: when})
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

// Remove deletes a task: done (decisions/0001).
func (s *Tasks) Remove(ctx context.Context, userID, id int64) error {
	return s.store.DeleteTask(ctx, userID, id)
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
