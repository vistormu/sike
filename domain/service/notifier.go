package service

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/port"
)

type SentStore interface {
	WasSent(ctx context.Context, userID int64, key string) (bool, error)
	MarkSent(ctx context.Context, userID int64, key string, at time.Time) error
	ForgetSent(ctx context.Context, before time.Time) error
}

// Notifier sends people their notifications (product/spec.md), to their ntfy
// topic: a weekly place's start and end (as its toggles say), a dated task's
// start, a deadline's warning (with Notify on). informational only. no port
// (SIKE_NTFY_URL unset) is notifications off.
type Notifier struct {
	port  port.Notifier
	sent  SentStore
	users *Users
	tasks *Tasks
	loc   *time.Location
}

func NewNotifier(p port.Notifier, sent SentStore, users *Users, tasks *Tasks, loc *time.Location) *Notifier {
	return &Notifier{port: p, sent: sent, users: users, tasks: tasks, loc: loc}
}

// LookBack is how far back every tick looks: a late tick or a short restart
// loses nothing, a failed send is tried again, and what was missed for longer
// is not sent late ("▶ Gym" an hour after is noise). what was sent is written
// down, so looking back never sends twice.
const LookBack = 10 * time.Minute

// Morning is when what has no hour notifies: an all-day day, a stretch's first
// day, a deadline with no hour.
const Morning = clock.Time(9 * 60)

// Note is one notification due: to whom, when, and Key, the moment it is for
// ("start:p3:2026-10-14T18:00"), which is what makes it sent once.
type Note struct {
	UserID int64
	Topic  string
	At     time.Time
	Key    string
	Msg    port.Message
}

// Run ticks at the start of every minute until ctx ends.
func (n *Notifier) Run(ctx context.Context, now func() time.Time) {
	for {
		if _, err := n.Tick(ctx, now(), false); err != nil {
			log.Printf("notifier: %v", err)
		}
		t := now()
		select {
		case <-ctx.Done():
			return
		case <-time.After(t.Truncate(time.Minute).Add(time.Minute).Sub(t)):
		}
	}
}

// Tick sends what is due in the LookBack before now and was not sent yet, and
// returns it. dry: only return it. a send that fails is logged and left for
// the next tick.
func (n *Notifier) Tick(ctx context.Context, now time.Time, dry bool) ([]Note, error) {
	if n.port == nil && !dry {
		return nil, nil
	}
	due, err := n.Due(ctx, now.Add(-LookBack), now)
	if err != nil {
		return nil, err
	}
	var out []Note
	for _, note := range due {
		sent, err := n.sent.WasSent(ctx, note.UserID, note.Key)
		if err != nil {
			return out, err
		}
		if sent {
			continue
		}
		if !dry {
			if err := n.port.Notify(ctx, note.Topic, note.Msg); err != nil {
				log.Printf("notifier: %s to %s: %v", note.Key, note.Topic, err)
				continue
			}
			if err := n.sent.MarkSent(ctx, note.UserID, note.Key, now); err != nil {
				return out, err
			}
		}
		out = append(out, note)
	}
	if dry {
		return out, nil
	}
	return out, n.sent.ForgetSent(ctx, now.Add(-24*time.Hour))
}

// Due is every notification whose moment falls in (from, to], for everyone
// with a topic: person by person, each by moment.
func (n *Notifier) Due(ctx context.Context, from, to time.Time) ([]Note, error) {
	users, err := n.users.List(ctx)
	if err != nil {
		return nil, err
	}
	var out []Note
	for _, u := range users {
		if u.Topic == "" {
			continue
		}
		tasks, err := n.tasks.List(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		places, err := n.tasks.Week(ctx, u.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, notes(u, tasks, places, from, to, n.loc)...)
	}
	return out, nil
}

// notes is one person's notifications with moments in (from, to], by moment.
func notes(u model.User, tasks []model.Task, places []model.Placement, from, to time.Time, loc *time.Location) []Note {
	var out []Note
	add := func(key string, d clock.Date, t clock.Time, title, body string) {
		at := d.At(t, loc)
		if at.After(from) && !at.After(to) {
			key = fmt.Sprintf("%s:%sT%s", key, d, t) // the wall-clock moment: a dst hour twice is once
			out = append(out, Note{UserID: u.ID, Topic: u.Topic, At: at, Key: key, Msg: port.Message{Title: title, Body: body}})
		}
	}
	first, last := clock.DateOf(from.In(loc)).AddDays(-1), clock.DateOf(to.In(loc)).AddDays(1)

	name := map[int64]string{}
	for _, t := range tasks {
		name[t.ID] = t.Name
	}
	for _, p := range places {
		if !p.NotifyStart && !p.NotifyEnd {
			continue
		}
		for _, o := range clock.Expand([]clock.Weekly{p.Weekly()}, first, last) {
			key := fmt.Sprintf("p%d", p.ID)
			if p.NotifyStart {
				add("start:"+key, o.Date, o.Start, "▶ "+name[p.TaskID], "until "+o.End.String())
			}
			if p.NotifyEnd {
				add("end:"+key, o.Date, o.End, "■ "+name[p.TaskID], "ended")
			}
		}
	}

	for _, t := range tasks {
		if t.Date == nil || !t.Date.Notify {
			continue
		}
		d, key := *t.Date, fmt.Sprintf("t%d", t.ID)
		switch {
		case d.Deadline():
			day, at := d.To, d.End
			if at == clock.NoTime { // no hour: the morning of the warning's first day
				day, at = d.To.AddDays(-u.Warn/(24*60)), Morning
			} else {
				due := d.To.At(d.End, loc).Add(-time.Duration(u.Warn) * time.Minute)
				day, at = clock.DateOf(due), clock.Time(due.Hour()*60+due.Minute())
			}
			add("warn:"+key, day, at, "⏰ "+t.Name, "due "+dueWhen(d, day))
		case d.AllDay() || d.From != d.To:
			body := "today"
			if d.From != d.To {
				body = fmt.Sprintf("until %s %d", d.To.Weekday().String()[:3], d.To.Day)
			}
			add("start:"+key, d.From, Morning, "▶ "+t.Name, body)
		default:
			add("start:"+key, d.From, d.Start, "▶ "+t.Name, "until "+d.End.String())
		}
	}
	slices.SortStableFunc(out, func(a, b Note) int { return a.At.Compare(b.At) })
	return out
}

// dueWhen says a deadline from day: "today 20:00", "tomorrow", "Fri 16".
func dueWhen(d model.Dates, day clock.Date) string {
	s := fmt.Sprintf("%s %d", d.To.Weekday().String()[:3], d.To.Day)
	switch day.DaysTo(d.To) {
	case 0:
		s = "today"
	case 1:
		s = "tomorrow"
	}
	if d.End != clock.NoTime {
		s += " " + d.End.String()
	}
	return s
}

// Test sends a test message to topic, the one a person typed in settings (it
// may not be saved yet): "Send test".
func (n *Notifier) Test(ctx context.Context, u model.User, topic string) error {
	switch {
	case n.port == nil:
		return model.Invalid("Notifications are off on this server")
	case topic == "":
		return model.Invalid("Type an ntfy topic first")
	case !model.IsTopic(topic):
		return model.Invalid("A topic is up to 64 letters, digits, - and _")
	}
	return n.port.Notify(ctx, topic, port.Message{Title: u.Emoji + " sike", Body: "Notifications reach you, " + u.Name + "."})
}
