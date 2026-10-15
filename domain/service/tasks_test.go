package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
)

type fakeTasks struct {
	created []model.Task
	moved   map[int64]model.When
	updated []model.Task
	places  [][]model.Placement // what each create or update stored
	dates   []*model.Dates
	of      []model.Placement // a task's placements, for Unplace
	removed []int64
}

func (f *fakeTasks) CreateTask(_ context.Context, t model.Task) (model.Task, error) {
	f.created = append(f.created, t)
	f.places = append(f.places, t.Places)
	f.dates = append(f.dates, t.Date)
	return t, nil
}
func (f *fakeTasks) ListTasks(context.Context, int64) ([]model.Task, error) { return nil, nil }
func (f *fakeTasks) GetTask(context.Context, int64, int64) (model.Task, error) {
	return model.Task{}, nil
}
func (f *fakeTasks) DeleteTask(context.Context, int64, int64) error { return nil }
func (f *fakeTasks) GetDeletedTask(context.Context, int64, int64) (model.Task, error) {
	return model.Task{}, nil
}
func (f *fakeTasks) RestoreTask(context.Context, int64, int64) error { return nil }
func (f *fakeTasks) UpdateTask(_ context.Context, t model.Task) (model.Task, error) {
	f.updated = append(f.updated, t)
	f.places = append(f.places, t.Places)
	f.dates = append(f.dates, t.Date)
	return t, nil
}
func (f *fakeTasks) ListPlacements(context.Context, int64) ([]model.Placement, error) {
	return f.of, nil
}
func (f *fakeTasks) TaskPlacements(context.Context, int64, int64) ([]model.Placement, error) {
	return f.of, nil
}
func (f *fakeTasks) GetPlacement(_ context.Context, _, id int64) (model.Placement, error) {
	for _, p := range f.of {
		if p.ID == id {
			return p, nil
		}
	}
	return model.Placement{}, model.ErrNotFound
}
func (f *fakeTasks) AddPlacement(_ context.Context, _ int64, p model.Placement) (model.Placement, error) {
	return p, nil
}
func (f *fakeTasks) DeletePlacement(_ context.Context, _, id int64) error {
	f.removed = append(f.removed, id)
	return nil
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
		{"Fix bike", "sky", model.Weekly, "at least one day"}, // no placements
		{"Fix bike", "sky", model.Dated, "Pick a date"},       // no dates
		{"Fix bike", "sky", "later", "every week, on a date, pending or someday"},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+string(tt.when), func(t *testing.T) {
			f := &fakeTasks{}
			task, err := NewTasks(f).Add(context.Background(), model.Task{UserID: 1, Name: tt.name, Color: tt.color, When: tt.when})
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
	if len(got) != 4 || got[0] != model.Weekly || got[1] != model.Dated || got[2] != model.Pending || got[3] != model.Someday {
		t.Fatalf("Whens = %v, want [weekly date pending someday]", got)
	}
	got[0] = model.Pending
	if s.Whens()[0] != model.Weekly {
		t.Error("Whens hands out the service's own slice")
	}
}

func TestEditTask(t *testing.T) {
	f := &fakeTasks{}
	s := NewTasks(f)
	got, err := s.Edit(context.Background(), model.Task{ID: 7, UserID: 1, Name: "  Fix bike ", Color: "mint", When: model.Someday})
	if err != nil || got.ID != 7 || got.UserID != 1 || got.Name != "Fix bike" || got.Color != "mint" || got.When != model.Someday {
		t.Errorf("edit: %+v, %v", got, err)
	}
	// the same checks as Add
	for _, bad := range []struct {
		name, color string
		when        model.When
	}{{"", "mint", model.Pending}, {"x", "beige", model.Pending}, {"x", "mint", model.Weekly}} {
		if _, err := s.Edit(context.Background(), model.Task{ID: 7, UserID: 1, Name: bad.name, Color: bad.color, When: bad.when}); !errors.Is(err, model.ErrInvalid) {
			t.Errorf("edit %+v: err = %v, want invalid", bad, err)
		}
	}
	if len(f.updated) != 1 {
		t.Errorf("%d updates stored, want 1", len(f.updated))
	}
}

func TestWeeklyPlacements(t *testing.T) {
	at := func(days model.Days, start, end clock.Time) model.Placement {
		return model.Placement{Days: days, Start: start, End: end}
	}
	wed := model.DaysOf(time.Wednesday)
	ok := at(wed, 18*60, 19*60+30)
	tests := []struct {
		name   string
		places []model.Placement
		err    string
	}{
		{"one time", []model.Placement{ok}, ""},
		{"two times", []model.Placement{ok, at(model.DaysOf(time.Saturday), 10*60, 11*60)}, ""},
		{"until midnight", []model.Placement{at(wed, 23*60, clock.Midnight)}, ""},
		{"none", nil, "at least one day"},
		{"no days", []model.Placement{at(0, 18*60, 19*60)}, "at least one day"},
		{"ends before it starts", []model.Placement{at(wed, 19*60, 18*60)}, "end is before the start"},
		{"no length", []model.Placement{at(wed, 18*60, 18*60)}, "end is before the start"},
		{"past midnight", []model.Placement{at(wed, 23*60, clock.Midnight+60)}, "00:00 to 24:00"},
		{"one bad of two", []model.Placement{ok, at(wed, 9*60, 8*60)}, "end is before the start"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeTasks{}
			_, err := NewTasks(f).Add(context.Background(), model.Task{UserID: 1, Name: "Gym", Color: "apricot", When: model.Weekly, Places: tt.places})
			if tt.err == "" {
				if err != nil || len(f.places) != 1 || len(f.places[0]) != len(tt.places) {
					t.Errorf("err = %v, stored %v", err, f.places)
				}
				return
			}
			if !errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), tt.err) || len(f.created) != 0 {
				t.Errorf("err = %v, want an invalid error with %q and nothing stored", err, tt.err)
			}
		})
	}

	// any other when drops the placements the form still sent
	f := &fakeTasks{}
	if _, err := NewTasks(f).Edit(context.Background(), model.Task{ID: 7, UserID: 1, Name: "Gym", Color: "apricot", When: model.Pending, Places: []model.Placement{ok}}); err != nil || f.places[0] != nil {
		t.Errorf("edit to pending: %v, stored %v placements, want none", err, f.places)
	}
	// weekly is not a move: it needs times
	if err := NewTasks(&fakeTasks{}).Move(context.Background(), 1, 7, model.Weekly); !errors.Is(err, model.ErrInvalid) {
		t.Errorf("move to weekly: err = %v, want invalid", err)
	}
}

func TestUnplaceKeepsOne(t *testing.T) {
	f := &fakeTasks{of: []model.Placement{{ID: 1, TaskID: 7}, {ID: 2, TaskID: 7}}}
	s := NewTasks(f)
	if err := s.Unplace(context.Background(), 1, 2); err != nil || len(f.removed) != 1 {
		t.Fatalf("unplace one of two: %v", err)
	}
	f.of = f.of[:1]
	if err := s.Unplace(context.Background(), 1, 1); !errors.Is(err, model.ErrInvalid) || len(f.removed) != 1 {
		t.Errorf("unplace the last: err = %v, want invalid and nothing removed", err)
	}
	if _, err := s.Place(context.Background(), 1, model.Placement{TaskID: 7, Days: 0, Start: 60, End: 120}); !errors.Is(err, model.ErrInvalid) {
		t.Errorf("place with no days: err = %v, want invalid", err)
	}
}

func TestDatedTasks(t *testing.T) {
	d := func(s string) clock.Date {
		x, err := clock.ParseDate(s)
		if err != nil {
			panic(err)
		}
		return x
	}
	no := clock.NoTime
	tests := []struct {
		name string
		in   *model.Dates
		want string // "from to start-end", or the error
	}{
		{"the dentist", &model.Dates{From: d("2026-10-15"), To: d("2026-10-15"), Start: 11 * 60, End: 12 * 60}, "2026-10-15 2026-10-15 11:00-12:00"},
		{"only a start: one day", &model.Dates{From: d("2026-10-15"), Start: no, End: no}, "2026-10-15 2026-10-15 -"},
		{"a deadline", &model.Dates{To: d("2026-10-20"), Start: no, End: no}, " 2026-10-20 -"},
		{"a deadline at an hour", &model.Dates{To: d("2026-10-20"), Start: 9 * 60, End: 14 * 60}, " 2026-10-20 -14:00"},
		{"a stretch", &model.Dates{From: d("2026-10-16"), To: d("2026-10-18"), Start: no, End: no}, "2026-10-16 2026-10-18 -"},
		{"a stretch with hours", &model.Dates{From: d("2026-10-16"), To: d("2026-10-18"), Start: 18 * 60, End: 9 * 60}, "2026-10-16 2026-10-18 18:00-09:00"},
		{"until midnight", &model.Dates{From: d("2026-10-15"), To: d("2026-10-15"), Start: 23 * 60, End: clock.Midnight}, "2026-10-15 2026-10-15 23:00-24:00"},
		{"none", nil, "Pick a date"},
		{"no dates", &model.Dates{Start: no, End: no}, "Pick a date"},
		{"ends before it starts", &model.Dates{From: d("2026-10-16"), To: d("2026-10-15"), Start: no, End: no}, "The end is before the start"},
		{"the hour before", &model.Dates{From: d("2026-10-15"), To: d("2026-10-15"), Start: 12 * 60, End: 11 * 60}, "The end is before the start"},
		{"one time only", &model.Dates{From: d("2026-10-15"), To: d("2026-10-15"), Start: 11 * 60, End: no}, "A time needs a start and an end"},
		{"past midnight", &model.Dates{From: d("2026-10-15"), To: d("2026-10-15"), Start: 23 * 60, End: clock.Midnight + 30}, "00:00 to 24:00"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeTasks{}
			_, err := NewTasks(f).Add(context.Background(), model.Task{UserID: 1, Name: "Dentist", Color: "coral", When: model.Dated, Date: tt.in})
			if err != nil {
				if !errors.Is(err, model.ErrInvalid) || !strings.Contains(err.Error(), tt.want) {
					t.Errorf("err = %v, want %q", err, tt.want)
				}
				return
			}
			got := f.dates[0]
			if s := got.From.String() + " " + got.To.String() + " " + got.Start.String() + "-" + got.End.String(); s != tt.want {
				t.Errorf("stored %q, want %q", s, tt.want)
			}
		})
	}
	// another when drops the dates; dated is not a move
	f := &fakeTasks{}
	if _, err := NewTasks(f).Add(context.Background(), model.Task{UserID: 1, Name: "x", Color: "coral", When: model.Pending, Date: &model.Dates{To: d("2026-10-20")}}); err != nil || f.dates[0] != nil {
		t.Errorf("a pending task kept its dates: %v", err)
	}
	if err := NewTasks(&fakeTasks{}).Move(context.Background(), 1, 7, model.Dated); !errors.Is(err, model.ErrInvalid) {
		t.Errorf("move to dated: err = %v, want invalid", err)
	}
}
