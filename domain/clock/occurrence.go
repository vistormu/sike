package clock

import (
	"cmp"
	"slices"
	"time"
)

// Weekly is a routine's place in the week: weekdays, and one wall-clock span
// on each. Ref is what it belongs to (a placement, m15).
type Weekly struct {
	Ref        int64
	Days       []time.Weekday
	Start, End Time
}

// Occurrence is one time something happens: a date and a wall-clock span.
type Occurrence struct {
	Ref        int64
	Date       Date
	Start, End Time
}

// Expand lists the occurrences of ws on the days from from up to, not
// including, to: by date, then start, then end, then ref. weekly places repeat
// every week, nothing skipped (product/spec.md).
func Expand(ws []Weekly, from, to Date) []Occurrence {
	var out []Occurrence
	for d := from; d.Before(to); d = d.AddDays(1) {
		for _, w := range ws {
			if slices.Contains(w.Days, d.Weekday()) {
				out = append(out, Occurrence{Ref: w.Ref, Date: d, Start: w.Start, End: w.End})
			}
		}
	}
	slices.SortStableFunc(out, func(a, b Occurrence) int {
		if a.Date != b.Date {
			if a.Date.Before(b.Date) {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(a.Start, b.Start), cmp.Compare(a.End, b.End), cmp.Compare(a.Ref, b.Ref))
	})
	return out
}

// Default hours of the week's timeline.
const (
	DayStart = 7
	DayEnd   = 22
)

// HourRange is the timeline's hours, from the first to the last: 07–22,
// stretched to the earliest start and latest end in occs, rounded out to the
// hour (decisions/0006). to is at most 24.
func HourRange(occs []Occurrence) (from, to int) {
	from, to = DayStart, DayEnd
	for _, o := range occs {
		from = min(from, o.Start.Hour())
		to = max(to, (int(o.End)+59)/60)
	}
	return from, to
}
