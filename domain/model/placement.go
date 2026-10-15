package model

import (
	"time"

	"sike/domain/clock"
)

// Placement is one time a weekly task takes in the week: weekdays, a
// wall-clock span (start < end, same day: decisions/0008), and whether to
// notify at its start and end.
type Placement struct {
	ID          int64      `db:"id"`
	TaskID      int64      `db:"task_id"`
	Days        Days       `db:"days"`
	Start       clock.Time `db:"start"`
	End         clock.Time `db:"end"`
	NotifyStart bool       `db:"notify_start"`
	NotifyEnd   bool       `db:"notify_end"`
}

// Weekly is the placement as the time core expands it.
func (p Placement) Weekly() clock.Weekly {
	return clock.Weekly{Ref: p.ID, Days: p.Days.List(), Start: p.Start, End: p.End}
}

// Days is a set of weekdays, one bit per time.Weekday.
type Days uint8

// Week is monday to sunday: the order days are shown in.
var Week = []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday, time.Saturday, time.Sunday}

func DaysOf(ds ...time.Weekday) Days {
	var s Days
	for _, d := range ds {
		s |= 1 << d
	}
	return s
}

func (s Days) Has(d time.Weekday) bool { return s&(1<<d) != 0 }

// List is the days in the set, monday first.
func (s Days) List() []time.Weekday {
	var out []time.Weekday
	for _, d := range Week {
		if s.Has(d) {
			out = append(out, d)
		}
	}
	return out
}
