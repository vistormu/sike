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

// Occurrence is one time something happens on a date: a wall-clock span on
// the timeline, or, for what has no hours, a chip (Kind).
type Occurrence struct {
	Ref        int64
	Kind       Kind
	Date       Date
	Start, End Time // NoTime where not given
}

// Kind is how an occurrence shows on its day.
type Kind int

const (
	Timed    Kind = iota // a span on the timeline
	AllDay               // a chip: an all-day day, or a day of a multi-day stretch
	Deadline             // a chip: something due that day, at End if given
)

// Dated is a dated task's time. no From: a deadline on To, at End if given.
// From and To the same day, with times: timed. no times, or several days: all
// day (decisions/0008: nothing timed crosses midnight; a multi-day stretch is
// chips, not a span).
type Dated struct {
	Ref        int64
	From, To   Date
	Start, End Time
}

// ExpandDated lists the occurrences of ds on the days from from up to, not
// including, to, sorted as Expand sorts.
func ExpandDated(ds []Dated, from, to Date) []Occurrence {
	var out []Occurrence
	for _, d := range ds {
		switch {
		case d.From.IsZero():
			if !d.To.Before(from) && d.To.Before(to) {
				out = append(out, Occurrence{Ref: d.Ref, Kind: Deadline, Date: d.To, Start: NoTime, End: d.End})
			}
		case d.From == d.To && d.Start != NoTime && d.End != NoTime:
			if !d.To.Before(from) && d.To.Before(to) {
				out = append(out, Occurrence{Ref: d.Ref, Kind: Timed, Date: d.To, Start: d.Start, End: d.End})
			}
		default:
			day := d.From
			if day.Before(from) {
				day = from
			}
			for ; !d.To.Before(day) && day.Before(to); day = day.AddDays(1) {
				out = append(out, Occurrence{Ref: d.Ref, Kind: AllDay, Date: day, Start: NoTime, End: NoTime})
			}
		}
	}
	sortOccurrences(out)
	return out
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
	sortOccurrences(out)
	return out
}

func sortOccurrences(out []Occurrence) {
	slices.SortStableFunc(out, func(a, b Occurrence) int {
		if a.Date != b.Date {
			if a.Date.Before(b.Date) {
				return -1
			}
			return 1
		}
		return cmp.Or(cmp.Compare(a.Start, b.Start), cmp.Compare(a.End, b.End), cmp.Compare(a.Ref, b.Ref))
	})
}

// Default hours of the week's timeline.
const (
	DayStart = 7
	DayEnd   = 22
)

// HourRange is the timeline's hours, from the first to the last: 07–22,
// stretched to the earliest start and latest end of the timed occs, rounded
// out to the hour (decisions/0006). to is at most 24.
func HourRange(occs []Occurrence) (from, to int) {
	from, to = DayStart, DayEnd
	for _, o := range occs {
		if o.Kind != Timed {
			continue
		}
		from = min(from, o.Start.Hour())
		to = max(to, (int(o.End)+59)/60)
	}
	return from, to
}
