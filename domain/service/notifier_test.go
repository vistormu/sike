package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/port"
)

// fakePort keeps what it is sent; while down, it refuses.
type fakePort struct {
	sent []string
	down bool
}

func (f *fakePort) Notify(_ context.Context, topic string, m port.Message) error {
	if f.down {
		return errors.New("ntfy: down")
	}
	f.sent = append(f.sent, topic+": "+m.Title+" · "+m.Body)
	return nil
}

func TestNotifierTest(t *testing.T) {
	you := model.User{ID: 1, Name: "You", Emoji: "🦊"}
	f := &fakePort{}
	n := NewNotifier(f, nil, nil, nil, nil)
	if err := n.Test(context.Background(), you, "sike-you"); err != nil {
		t.Fatal(err)
	}
	if len(f.sent) != 1 || f.sent[0] != "sike-you: 🦊 sike · Notifications reach you, You." {
		t.Errorf("sent %q", f.sent)
	}
	for _, topic := range []string{"", "sike you", "sike/you"} {
		if err := n.Test(context.Background(), you, topic); !errors.Is(err, model.ErrInvalid) {
			t.Errorf("topic %q: err = %v, want ErrInvalid", topic, err)
		}
	}
	if err := NewNotifier(nil, nil, nil, nil, nil).Test(context.Background(), you, "sike-you"); !errors.Is(err, model.ErrInvalid) {
		t.Errorf("no ntfy: err = %v, want ErrInvalid (notifications off)", err)
	}
}

var madrid, _ = time.LoadLocation("Europe/Madrid")

func at(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, madrid)
	if err != nil {
		panic(err)
	}
	return t
}

func dates(spec string, notify bool) *model.Dates {
	// "from to start end", "-" for none, as the upcoming tests write them
	f := strings.Fields(spec)
	d := model.Dates{Start: clock.NoTime, End: clock.NoTime, Notify: notify}
	if f[0] != "-" {
		d.From, _ = clock.ParseDate(f[0])
	}
	d.To, _ = clock.ParseDate(f[1])
	if f[2] != "-" {
		d.Start, _ = clock.ParseTime(f[2])
	}
	if f[3] != "-" {
		d.End, _ = clock.ParseTime(f[3])
	}
	return &d
}

// the week of wed 2026-10-14, notified: what goes out, and when.
func TestNotes(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Name: "Gym"},
		{ID: 2, Name: "Work"},
		{ID: 3, Name: "Dentist", Date: dates("2026-10-15 2026-10-15 11:00 12:00", true)},
		{ID: 4, Name: "Ana's birthday", Date: dates("2026-10-14 2026-10-14 - -", true)},
		{ID: 5, Name: "Trip", Date: dates("2026-10-17 2026-10-20 - -", true)},
		{ID: 6, Name: "Taxes", Date: dates("- 2026-10-16 - -", true)},
		{ID: 7, Name: "Report", Date: dates("- 2026-10-14 - 20:00", true)},
		{ID: 8, Name: "Haircut", Date: dates("2026-10-15 2026-10-15 10:00 11:00", false)}, // notify off
	}
	places := []model.Placement{
		{ID: 1, TaskID: 1, Days: model.DaysOf(time.Wednesday), Start: 18 * 60, End: 19*60 + 30, NotifyStart: true, NotifyEnd: true},
		{ID: 2, TaskID: 2, Days: model.DaysOf(time.Wednesday, time.Thursday), Start: 9 * 60, End: 17 * 60}, // no toggles
	}
	run := func(warn int, from, to string) []string {
		u := model.User{ID: 1, Topic: "sike-you", Warn: warn}
		var out []string
		for _, n := range notes(u, tasks, places, at(from), at(to), madrid) {
			out = append(out, n.At.In(madrid).Format("Mon 15:04")+" "+n.Msg.Title+" · "+n.Msg.Body)
		}
		return out
	}
	tests := []struct {
		name     string
		warn     int
		from, to string
		want     []string
	}{
		{"wednesday, a day's warning", 24 * 60, "2026-10-14 00:00", "2026-10-15 00:00", []string{
			"Wed 09:00 ▶ Ana's birthday · today",
			"Wed 18:00 ▶ Gym · until 19:30",
			"Wed 19:30 ■ Gym · ended",
		}},
		{"thursday, a day's warning", 24 * 60, "2026-10-15 00:00", "2026-10-16 00:00", []string{
			"Thu 09:00 ⏰ Taxes · due tomorrow",
			"Thu 11:00 ▶ Dentist · until 12:00",
		}},
		{"tuesday: the report a day before its hour", 24 * 60, "2026-10-13 00:00", "2026-10-14 00:00", []string{
			"Tue 20:00 ⏰ Report · due tomorrow 20:00",
		}},
		{"two days' warning, wednesday", 48 * 60, "2026-10-14 00:00", "2026-10-14 12:00", []string{
			"Wed 09:00 ▶ Ana's birthday · today",
			"Wed 09:00 ⏰ Taxes · due Fri 16",
		}},
		{"an hour's warning: the hour before, the morning of", 60, "2026-10-14 12:00", "2026-10-16 12:00", []string{
			"Wed 18:00 ▶ Gym · until 19:30",
			"Wed 19:00 ⏰ Report · due today 20:00",
			"Wed 19:30 ■ Gym · ended",
			"Thu 11:00 ▶ Dentist · until 12:00",
			"Fri 09:00 ⏰ Taxes · due today",
		}},
		{"a stretch, on its first morning", 24 * 60, "2026-10-17 00:00", "2026-10-19 00:00", []string{
			"Sat 09:00 ▶ Trip · until Tue 20",
		}},
		{"from is not included, to is", 24 * 60, "2026-10-14 18:00", "2026-10-14 19:30", []string{
			"Wed 19:30 ■ Gym · ended",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := run(tt.warn, tt.from, tt.to); strings.Join(got, "\n") != strings.Join(tt.want, "\n") {
				t.Errorf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(tt.want, "\n  "))
			}
		})
	}
}

// the clocks go back on sun 2026-10-25: 02:30 happens twice, and is notified
// once; they go forward on sun 2026-03-29: 02:30 never happens, and is
// notified at 03:30.
func TestNotesDST(t *testing.T) {
	u := model.User{ID: 1, Topic: "sike-you", Warn: 24 * 60}
	tasks := []model.Task{{ID: 1, Name: "Night bus"}}
	places := []model.Placement{{ID: 1, TaskID: 1, Days: model.DaysOf(time.Sunday), Start: 2*60 + 30, End: 3 * 60, NotifyStart: true}}
	back := notes(u, tasks, places, at("2026-10-25 00:00"), at("2026-10-25 06:00"), madrid)
	if len(back) != 1 {
		t.Errorf("autumn: %d notes for 02:30, want 1", len(back))
	}
	fwd := notes(u, tasks, places, at("2026-03-29 00:00"), at("2026-03-29 06:00"), madrid)
	if len(fwd) != 1 || fwd[0].At.In(madrid).Format("15:04") != "03:30" {
		t.Errorf("spring: %v, want one at 03:30", fwd)
	}
}

type fakeSent map[string]bool

func (f fakeSent) WasSent(_ context.Context, _ int64, key string) (bool, error) { return f[key], nil }
func (f fakeSent) MarkSent(_ context.Context, _ int64, key string, _ time.Time) error {
	f[key] = true
	return nil
}
func (f fakeSent) ForgetSent(context.Context, time.Time) error { return nil }

type fakeTaskStore struct {
	TaskStore
	tasks  []model.Task
	places []model.Placement
}

func (f fakeTaskStore) ListTasks(context.Context, int64) ([]model.Task, error) { return f.tasks, nil }
func (f fakeTaskStore) ListPlacements(context.Context, int64) ([]model.Placement, error) {
	return f.places, nil
}

// a tick sends what is due once, retries what failed, and a dry run only says.
func TestTick(t *testing.T) {
	users := &fakeUsers{}
	you, _ := NewUsers(users).Add(context.Background(), "You", "🦊")
	you.Topic = "sike-you"
	users.created[0] = you
	NewUsers(users).Add(context.Background(), "Ana", "🐱") // no topic: nothing for her
	tasks := NewTasks(fakeTaskStore{
		tasks:  []model.Task{{ID: 1, UserID: you.ID, Name: "Gym"}},
		places: []model.Placement{{ID: 1, TaskID: 1, Days: model.DaysOf(time.Wednesday), Start: 18 * 60, End: 19 * 60, NotifyStart: true}},
	})
	p, sent := &fakePort{}, fakeSent{}
	n := NewNotifier(p, sent, NewUsers(users), tasks, madrid)
	tick := func(when string, dry bool) int {
		t.Helper()
		notes, err := n.Tick(context.Background(), at(when), dry)
		if err != nil {
			t.Fatal(err)
		}
		return len(notes)
	}

	if tick("2026-10-14 18:00", true) != 1 || len(p.sent) != 0 || len(sent) != 0 {
		t.Error("a dry run sent or recorded")
	}
	p.down = true
	if tick("2026-10-14 18:00", false) != 0 || len(sent) != 0 {
		t.Error("a failed send counted as sent")
	}
	p.down = false
	if tick("2026-10-14 18:04", false) != 1 || tick("2026-10-14 18:05", false) != 0 {
		t.Error("the next tick did not send it once")
	}
	if len(p.sent) != 1 || p.sent[0] != "sike-you: ▶ Gym · until 19:00" {
		t.Errorf("sent %q", p.sent)
	}
	if tick("2026-10-14 18:11", false) != 0 {
		t.Error("looking back further than LookBack")
	}
}
