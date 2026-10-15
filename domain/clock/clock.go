// Package clock is the time core: wall-clock times, local dates, the week, and
// what happens when. it never reads the clock: callers pass now and the
// location. times are wall-clock in one location (technical-design.md, time),
// so a routine at "wed 18:00" stays at 18:00 across dst.
package clock

import (
	"fmt"
	"time"
)

// Time is a wall-clock time, minutes after midnight: 00:00 to 24:00, where
// 24:00 can only end something.
type Time int

const Midnight Time = 24 * 60

// ParseTime reads "HH:MM", 24h.
func ParseTime(s string) (Time, error) {
	digit := func(i int) (int, bool) { return int(s[i] - '0'), s[i] >= '0' && s[i] <= '9' }
	if len(s) != 5 || s[2] != ':' {
		return 0, fmt.Errorf("time %q: want HH:MM", s)
	}
	h1, ok1 := digit(0)
	h2, ok2 := digit(1)
	m1, ok3 := digit(3)
	m2, ok4 := digit(4)
	h, m := h1*10+h2, m1*10+m2
	if !ok1 || !ok2 || !ok3 || !ok4 || m > 59 || h > 24 || h == 24 && m > 0 {
		return 0, fmt.Errorf("time %q: want HH:MM, 00:00 to 24:00", s)
	}
	return Time(h*60 + m), nil
}

func (t Time) Hour() int   { return int(t) / 60 }
func (t Time) Minute() int { return int(t) % 60 }

func (t Time) String() string { return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute()) }

// Date is a day on the calendar, in no location. comparable with ==.
type Date struct {
	Year  int
	Month time.Month
	Day   int
}

// NewDate normalizes like time.Date: Oct 32 is Nov 1.
func NewDate(year int, month time.Month, day int) Date {
	return DateOf(time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
}

// DateOf is the date of t, in t's location.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{y, m, d}
}

// Today is the date now, where loc is.
func Today(now time.Time, loc *time.Location) Date { return DateOf(now.In(loc)) }

// ParseDate reads "2006-01-02".
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil {
		return Date{}, fmt.Errorf("date %q: want YYYY-MM-DD", s)
	}
	return DateOf(t), nil
}

func (d Date) String() string { return d.utc().Format(time.DateOnly) }

// utc is the date's midnight in UTC: arithmetic with no dst in it.
func (d Date) utc() time.Time { return time.Date(d.Year, d.Month, d.Day, 0, 0, 0, 0, time.UTC) }

func (d Date) AddDays(n int) Date { return NewDate(d.Year, d.Month, d.Day+n) }

func (d Date) Weekday() time.Weekday { return d.utc().Weekday() }

// Before reports whether d is earlier than e.
func (d Date) Before(e Date) bool { return d.utc().Before(e.utc()) }

// Monday is the first day of d's week: weeks start on monday.
func (d Date) Monday() Date { return d.AddDays(-(int(d.Weekday()) + 6) % 7) }

// ISOWeek is the week's number in its year (iso 8601, as calendars print it).
func (d Date) ISOWeek() int {
	_, w := d.utc().ISOWeek()
	return w
}

// At is the instant at wall-clock t on d, where loc is. a time the dst change
// skips (02:30 on the spring-forward night) comes out an hour later, as
// time.Date does; 24:00 is the next day's midnight.
func (d Date) At(t Time, loc *time.Location) time.Time {
	return time.Date(d.Year, d.Month, d.Day, 0, int(t), 0, 0, loc)
}
