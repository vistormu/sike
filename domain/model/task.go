package model

import "sike/domain/clock"

// When is where a task lives in time, and so which tab shows it
// (decisions/0005). exactly one per task.
type When string

const (
	Weekly  When = "weekly"  // placements on weekdays (m15)
	Dated   When = "date"    // a date, a deadline (m18)
	Pending When = "pending" // owed, no date
	Someday When = "someday" // a wish
)

// Whens in tab order: week, upcoming, pending, someday.
var Whens = []When{Weekly, Dated, Pending, Someday}

// Task is a thing to do, in one area (its color) and one place in time. a
// weekly task has its Places, a dated one its Date.
type Task struct {
	ID       int64  `db:"id"`
	UserID   int64  `db:"user_id"`
	Name     string `db:"name"`
	Color    string `db:"color"`
	When     When   `db:"when"`
	Position int64  `db:"position"`

	Places []Placement `db:"-"`
	Date   *Dates      `db:"-"`
}

// Dates is when a dated task happens. no From: a deadline on To, at End if
// given. one day with Start and End: timed. no times, or several days: all
// day. a stretch of days is never timed on the timeline (decisions/0008).
type Dates struct {
	From, To   clock.Date
	Start, End clock.Time // clock.NoTime when not given
	Notify     bool       // at the start
}

func (d Dates) Deadline() bool { return d.From.IsZero() }

// AllDay: a start and no hours.
func (d Dates) AllDay() bool { return !d.From.IsZero() && d.Start == clock.NoTime }

// Clock is the dated task as the time core expands it.
func (d Dates) Clock(ref int64) clock.Dated {
	return clock.Dated{Ref: ref, From: d.From, To: d.To, Start: d.Start, End: d.End}
}

// Over: by today at minute now, the dated task has gone by: its last day is
// behind, or it is today and its end hour has passed. a day with no end hour
// lasts until midnight.
func (d Dates) Over(today clock.Date, now clock.Time) bool {
	switch {
	case d.To.Before(today):
		return true
	case d.To != today:
		return false
	}
	return d.End != clock.NoTime && now >= d.End
}

// Shows is the day a dated task belongs to from today on: its first day (a
// deadline's day), or today while a stretch that began before goes on.
func (d Dates) Shows(today clock.Date) clock.Date {
	first := d.From
	if first.IsZero() {
		first = d.To
	}
	if first.Before(today) {
		return today
	}
	return first
}
