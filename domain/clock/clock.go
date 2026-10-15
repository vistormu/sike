// Package clock is the time core: wall-clock times, local dates, the week, and
// what happens when. it never reads the clock: callers pass now and the
// location. times are wall-clock in one location (technical-design.md, time),
// so a routine at "wed 18:00" stays at 18:00 across dst.
package clock

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Time is a wall-clock time, minutes after midnight: 00:00 to 24:00, where
// 24:00 can only end something.
type Time int

const Midnight Time = 24 * 60

// NoTime is a time not given: an all-day day, a deadline with no hour.
const NoTime Time = -1

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

// ReadTime reads a time as people type it, 24h: "18:30", "18.30", "1830",
// "830", "18", "8". anything else is an error.
func ReadTime(s string) (Time, error) {
	s = strings.TrimSpace(s)
	h, m, sep := strings.Cut(strings.NewReplacer(".", ":", "h", ":", "H", ":").Replace(s), ":")
	if !sep {
		switch len(s) {
		case 1, 2:
			h, m = s, "00"
		case 3, 4:
			h, m = s[:len(s)-2], s[len(s)-2:]
		default:
			return 0, fmt.Errorf("time %q: want HH:MM", s)
		}
	}
	if len(h) == 1 {
		h = "0" + h
	}
	if m == "" {
		m = "00"
	}
	return ParseTime(h + ":" + m)
}

func (t Time) Hour() int   { return int(t) / 60 }
func (t Time) Minute() int { return int(t) % 60 }

// String is "HH:MM"; NoTime is "".
func (t Time) String() string {
	if t == NoTime {
		return ""
	}
	return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
}

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

// String is "2006-01-02"; the zero Date is "".
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.utc().Format(time.DateOnly)
}

// IsZero: no date (a deadline has no start).
func (d Date) IsZero() bool { return d == Date{} }

// days is d's number of days since 1970-01-01.
func (d Date) days() int { return int(d.utc().Unix() / 86400) }

// near is how far from today a date may be and still be written without its
// year: about six months either way.
const near = 182

// Short is d as the form shows it: "Thu, Oct 15", with the year when d is
// more than about six months from today ("Thu, Oct 15, 2027"). ReadDate reads
// it back to the same day.
func (d Date) Short(today Date) string {
	if d.IsZero() {
		return ""
	}
	s := d.utc().Format("Mon, Jan 2")
	if abs(d.days()-today.days()) > near {
		s += d.utc().Format(", 2006")
	}
	return s
}

func abs(n int) int { return max(n, -n) }

var months = []string{"jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

// ReadDate reads a date as people type it, day first: "15/10", "15/10/2026",
// "15.10.26", "2026-10-15", "Oct 15", "15 oct 2026", "Thu, Oct 15". with no
// year, it is the one nearest today (Short leaves out only such years).
func ReadDate(s string, today Date) (Date, error) {
	bad := fmt.Errorf("date %q: want 15/10 or 15/10/2026", s)
	if d, err := ParseDate(strings.TrimSpace(s)); err == nil {
		return d, nil
	}
	// words and numbers, whatever separates them; a weekday name is skipped
	f := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == '/' || r == '.' || r == '-' || r == ',' || r == ' '
	})
	var nums []int
	month := 0
	for _, w := range f {
		if i := slices.IndexFunc(months, func(m string) bool { return strings.HasPrefix(w, m) }); len(w) >= 3 && i >= 0 {
			month = i + 1
			continue
		}
		if len(w) >= 3 && slices.ContainsFunc([]string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}, func(d string) bool { return strings.HasPrefix(w, d) }) {
			continue
		}
		n, err := strconv.Atoi(w)
		if err != nil {
			return Date{}, bad
		}
		nums = append(nums, n)
	}
	var day, year int
	switch {
	case month > 0 && len(nums) == 1: // oct 15
		day = nums[0]
	case month > 0 && len(nums) == 2: // oct 15 2026
		day, year = nums[0], nums[1]
	case month == 0 && len(nums) == 2: // 15/10
		day, month = nums[0], nums[1]
	case month == 0 && len(nums) == 3: // 15/10/2026
		day, month, year = nums[0], nums[1], nums[2]
	default:
		return Date{}, bad
	}
	if year > 0 && year < 100 {
		year += 2000
	}
	if day < 1 || month < 1 || month > 12 {
		return Date{}, bad
	}
	if year == 0 { // the nearest such date: this year's, or the next or last
		best := Date{}
		for y := today.Year - 1; y <= today.Year+1; y++ {
			c := Date{y, time.Month(month), day}
			if best.IsZero() || abs(c.days()-today.days()) < abs(best.days()-today.days()) {
				best = c
			}
		}
		year = best.Year
	}
	d := Date{year, time.Month(month), day}
	if NewDate(year, time.Month(month), day) != d { // 31/2
		return Date{}, bad
	}
	return d, nil
}

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
