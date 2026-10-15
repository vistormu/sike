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
	CreateTask(ctx context.Context, t model.Task) (model.Task, error)
	ListTasks(ctx context.Context, userID int64) ([]model.Task, error)
	GetTask(ctx context.Context, userID, id int64) (model.Task, error)
	DeleteTask(ctx context.Context, userID, id int64) error
	GetDeletedTask(ctx context.Context, userID, id int64) (model.Task, error)
	RestoreTask(ctx context.Context, userID, id int64) error
	MoveTask(ctx context.Context, userID, id int64, when model.When) error
	UpdateTask(ctx context.Context, t model.Task) (model.Task, error)
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
var whens = []model.When{model.Weekly, model.Dated, model.Pending, model.Someday}

// Add creates a task on top of its place: a weekly task with its Places, a
// dated one with its Date. what another when does not use is dropped.
func (s *Tasks) Add(ctx context.Context, t model.Task) (model.Task, error) {
	t.ID = 0
	t, err := check(t)
	if err != nil {
		return model.Task{}, err
	}
	return s.store.CreateTask(ctx, t)
}

// Edit changes a task's name, area, when and times, all of them: its places
// and dates are replaced. a new when moves it on top of its new place
// (decisions/0005); the same when keeps its position.
func (s *Tasks) Edit(ctx context.Context, t model.Task) (model.Task, error) {
	t, err := check(t)
	if err != nil {
		return model.Task{}, err
	}
	return s.store.UpdateTask(ctx, t)
}

// check validates a task as typed: its name trimmed, its places or dates as
// its when needs them, the others dropped.
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
	if err := checkWhen(t.When); err != nil {
		return t, err
	}
	places, date := t.Places, t.Date
	t.Places, t.Date = nil, nil
	switch t.When {
	case model.Weekly:
		if len(places) == 0 {
			return t, model.Invalid("Pick at least one day")
		}
		for _, p := range places {
			if err := checkPlacement(p); err != nil {
				return t, err
			}
		}
		t.Places = places
	case model.Dated:
		d, err := checkDates(date)
		if err != nil {
			return t, err
		}
		t.Date = &d
	}
	return t, nil
}

// checkDates: an end date (the start's, if only that is given); no start is a
// deadline, at an hour or not; with a start, both times or none (all day), and
// one day's end after its start (decisions/0008).
func checkDates(d *model.Dates) (model.Dates, error) {
	if d == nil || d.From.IsZero() && d.To.IsZero() {
		return model.Dates{}, model.Invalid("Pick a date")
	}
	out := *d
	if out.To.IsZero() {
		out.To = out.From
	}
	inDay := func(t clock.Time, end bool) bool {
		return t == clock.NoTime || t >= 0 && (t < clock.Midnight || end && t == clock.Midnight)
	}
	switch {
	case !inDay(out.Start, false) || !inDay(out.End, true):
		return out, model.Invalid("Times go from 00:00 to 24:00")
	case out.From.IsZero(): // a deadline
		out.Start = clock.NoTime
	case out.To.Before(out.From):
		return out, model.Invalid("The end is before the start")
	case (out.Start == clock.NoTime) != (out.End == clock.NoTime):
		return out, model.Invalid("A time needs a start and an end")
	case out.From == out.To && out.Start != clock.NoTime && out.End <= out.Start:
		return out, model.Invalid("The end is before the start")
	}
	return out, nil
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

// Move puts a task in another place in time, on top there. weekly and dated
// need times: that is Edit.
func (s *Tasks) Move(ctx context.Context, userID, id int64, when model.When) error {
	if when == model.Weekly || when == model.Dated {
		return model.Invalid("A %s task needs its times: edit it instead", map[model.When]string{model.Weekly: "weekly", model.Dated: "dated"}[when])
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
	return model.Invalid("When is every week, on a date, pending or someday")
}
