package service

import (
	"context"
	"slices"
	"strings"
	"unicode/utf8"

	"sike/domain/clock"
	"sike/domain/model"
)

type TaskStore interface {
	CreateTask(ctx context.Context, t model.Task, places []model.Placement) (model.Task, error)
	ListTasks(ctx context.Context, userID int64) ([]model.Task, error)
	GetTask(ctx context.Context, userID, id int64) (model.Task, error)
	DeleteTask(ctx context.Context, userID, id int64) error
	GetDeletedTask(ctx context.Context, userID, id int64) (model.Task, error)
	RestoreTask(ctx context.Context, userID, id int64) error
	MoveTask(ctx context.Context, userID, id int64, when model.When) error
	UpdateTask(ctx context.Context, t model.Task, places []model.Placement) (model.Task, error)
	ListPlacements(ctx context.Context, userID int64) ([]model.Placement, error)
	TaskPlacements(ctx context.Context, userID, taskID int64) ([]model.Placement, error)
	GetPlacement(ctx context.Context, userID, id int64) (model.Placement, error)
	AddPlacement(ctx context.Context, userID int64, p model.Placement) (model.Placement, error)
	DeletePlacement(ctx context.Context, userID, id int64) error
}

type Tasks struct {
	store TaskStore
}

func NewTasks(store TaskStore) *Tasks {
	return &Tasks{store: store}
}

const maxTaskName = 80

// whens are the places a task can be put, for now.
var whens = []model.When{model.Weekly, model.Pending, model.Someday}

// Add creates a task on top of its place. a weekly task comes with its
// placements; any other ignores them.
func (s *Tasks) Add(ctx context.Context, userID int64, name, color string, when model.When, places []model.Placement) (model.Task, error) {
	t, err := check(model.Task{UserID: userID, Name: name, Color: color, When: when})
	if err != nil {
		return model.Task{}, err
	}
	if places, err = checkPlaces(when, places); err != nil {
		return model.Task{}, err
	}
	return s.store.CreateTask(ctx, t, places)
}

// Edit changes a task's name, area and when, and replaces its placements. a
// new when moves it on top of its new place (decisions/0005); the same when
// keeps its position. a task that stops being weekly loses its placements.
func (s *Tasks) Edit(ctx context.Context, userID, id int64, name, color string, when model.When, places []model.Placement) (model.Task, error) {
	t, err := check(model.Task{ID: id, UserID: userID, Name: name, Color: color, When: when})
	if err != nil {
		return model.Task{}, err
	}
	if places, err = checkPlaces(when, places); err != nil {
		return model.Task{}, err
	}
	return s.store.UpdateTask(ctx, t, places)
}

// checkPlaces: a weekly task needs at least one valid placement; any other
// has none.
func checkPlaces(when model.When, places []model.Placement) ([]model.Placement, error) {
	if when != model.Weekly {
		return nil, nil
	}
	if len(places) == 0 {
		return nil, model.Invalid("Pick at least one day")
	}
	for _, p := range places {
		if err := checkPlacement(p); err != nil {
			return nil, err
		}
	}
	return places, nil
}

// checkPlacement: some days, and a span that starts before it ends, on one
// day (decisions/0008).
func checkPlacement(p model.Placement) error {
	switch {
	case p.Days == 0 || p.Days > model.DaysOf(model.Week...):
		return model.Invalid("Pick at least one day")
	case p.Start < 0 || p.End > clock.Midnight || p.Start >= clock.Midnight:
		return model.Invalid("Times go from 00:00 to 24:00")
	case p.End <= p.Start:
		return model.Invalid("The end is before the start")
	}
	return nil
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

// Move puts a task in another place in time, on top there. weekly needs
// times: that is Edit.
func (s *Tasks) Move(ctx context.Context, userID, id int64, when model.When) error {
	if when == model.Weekly {
		return model.Invalid("A weekly task needs its times: edit it instead")
	}
	if err := checkWhen(when); err != nil {
		return err
	}
	return s.store.MoveTask(ctx, userID, id, when)
}

// Placements are a task's times in the week, by start.
func (s *Tasks) Placements(ctx context.Context, userID, taskID int64) ([]model.Placement, error) {
	return s.store.TaskPlacements(ctx, userID, taskID)
}

// Week is every placement of a person's tasks: what the week repeats.
func (s *Tasks) Week(ctx context.Context, userID int64) ([]model.Placement, error) {
	return s.store.ListPlacements(ctx, userID)
}

// Place adds a time to a weekly task.
func (s *Tasks) Place(ctx context.Context, userID int64, p model.Placement) (model.Placement, error) {
	if err := checkPlacement(p); err != nil {
		return model.Placement{}, err
	}
	return s.store.AddPlacement(ctx, userID, p)
}

// Unplace removes a time from a weekly task, but not its last one.
func (s *Tasks) Unplace(ctx context.Context, userID, id int64) error {
	p, err := s.store.GetPlacement(ctx, userID, id)
	if err != nil {
		return err
	}
	places, err := s.store.TaskPlacements(ctx, userID, p.TaskID)
	if err != nil {
		return err
	}
	if len(places) == 1 {
		return model.Invalid("A weekly task needs at least one time")
	}
	return s.store.DeletePlacement(ctx, userID, id)
}

func checkWhen(w model.When) error {
	for _, ok := range whens {
		if w == ok {
			return nil
		}
	}
	return model.Invalid("When is every week, pending or someday")
}
