package clock

import (
	"testing"
	"time"
)

func madrid(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/Madrid")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		in   string
		want Time // -1: an error
	}{
		{"00:00", 0},
		{"07:05", 7*60 + 5},
		{"18:30", 18*60 + 30},
		{"23:59", 23*60 + 59},
		{"24:00", Midnight},
		{"24:01", -1},
		{"25:00", -1},
		{"18:60", -1},
		{"9:00", -1},
		{" 9:00", -1},
		{"-1:00", -1},
		{"1830", -1},
		{"ab:cd", -1},
		{"", -1},
	}
	for _, tt := range tests {
		got, err := ParseTime(tt.in)
		if tt.want < 0 {
			if err == nil {
				t.Errorf("ParseTime(%q) = %v, want an error", tt.in, got)
			}
			continue
		}
		if err != nil || got != tt.want {
			t.Errorf("ParseTime(%q) = %v, %v; want %v", tt.in, got, err, tt.want)
		}
		if err == nil && got.String() != tt.in {
			t.Errorf("%q prints as %q", tt.in, got.String())
		}
	}
}

func TestReadTime(t *testing.T) {
	for in, want := range map[string]string{
		"18:30": "18:30", "18.30": "18:30", "18h30": "18:30", "1830": "18:30",
		"830": "08:30", "8:30": "08:30", "18": "18:00", "8": "08:00", "18h": "18:00",
		" 07:05 ": "07:05", "0000": "00:00", "24": "24:00", "2400": "24:00",
	} {
		got, err := ReadTime(in)
		if err != nil || got.String() != want {
			t.Errorf("ReadTime(%q) = %v, %v; want %s", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "7pm", "18:3", "1860", "2401", "25", "12345", "a8", "18:30:00", "-1"} {
		if got, err := ReadTime(bad); err == nil {
			t.Errorf("ReadTime(%q) = %v, want an error", bad, got)
		}
	}
}

func TestDates(t *testing.T) {
	d := func(s string) Date {
		t.Helper()
		x, err := ParseDate(s)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	// monday starts the week
	for day, monday := range map[string]string{
		"2026-10-12": "2026-10-12", // monday
		"2026-10-15": "2026-10-12", // thursday
		"2026-10-18": "2026-10-12", // sunday
		"2026-11-01": "2026-10-26", // sunday, across a month
		"2027-01-01": "2026-12-28", // friday, across a year
	} {
		if got := d(day).Monday(); got != d(monday) {
			t.Errorf("Monday of %s = %s, want %s", day, got, monday)
		}
	}
	for _, tt := range []struct {
		from string
		n    int
		want string
	}{
		{"2026-10-31", 1, "2026-11-01"},
		{"2026-12-31", 1, "2027-01-01"},
		{"2028-02-28", 1, "2028-02-29"},
		{"2026-03-01", -1, "2026-02-28"},
		{"2026-03-28", 2, "2026-03-30"}, // across the dst night: still two days
	} {
		if got := d(tt.from).AddDays(tt.n); got != d(tt.want) {
			t.Errorf("%s %+d days = %s, want %s", tt.from, tt.n, got, tt.want)
		}
	}
	for day, week := range map[string]int{"2026-10-15": 42, "2026-01-01": 1, "2027-01-01": 53, "2026-12-28": 53} {
		if got := d(day).ISOWeek(); got != week {
			t.Errorf("week of %s = %d, want %d", day, got, week)
		}
	}
	if _, err := ParseDate("2026-02-30"); err == nil {
		t.Error("ParseDate accepted Feb 30")
	}
	if !d("2026-10-15").Before(d("2026-10-16")) || d("2026-10-16").Before(d("2026-10-15")) {
		t.Error("Before is wrong")
	}
}

func TestToday(t *testing.T) {
	loc := madrid(t)
	// 23:30 utc is already tomorrow in madrid (cest, +2)
	now := time.Date(2026, 10, 15, 23, 30, 0, 0, time.UTC)
	if got := Today(now, loc); got != NewDate(2026, 10, 16) {
		t.Errorf("Today = %s, want 2026-10-16", got)
	}
	if got := Today(now, time.UTC); got != NewDate(2026, 10, 15) {
		t.Errorf("Today in utc = %s, want 2026-10-15", got)
	}
}

// a routine at 18:00 stays at 18:00 on both sides of a dst change: the days
// between are 23 or 25 hours long, not the wall-clock time.
func TestAtAcrossDST(t *testing.T) {
	loc := madrid(t)
	evening := Time(18 * 60)
	for _, tt := range []struct {
		before, after Date
		hours         float64
	}{
		{NewDate(2026, 3, 28), NewDate(2026, 3, 29), 23},   // spring forward
		{NewDate(2026, 10, 24), NewDate(2026, 10, 25), 25}, // fall back
		{NewDate(2026, 10, 14), NewDate(2026, 10, 15), 24},
	} {
		a, b := tt.before.At(evening, loc), tt.after.At(evening, loc)
		if a.Hour() != 18 || b.Hour() != 18 {
			t.Errorf("18:00 on %s/%s came out %s / %s", tt.before, tt.after, a, b)
		}
		if got := b.Sub(a).Hours(); got != tt.hours {
			t.Errorf("%s → %s: %v h, want %v", tt.before, tt.after, got, tt.hours)
		}
	}
	// the spring-forward night skips 02:00–03:00: 02:30 comes out at 03:30
	if got := NewDate(2026, 3, 29).At(Time(2*60+30), loc); got.Hour() != 3 || got.Minute() != 30 {
		t.Errorf("02:30 on the dst night = %s, want 03:30", got)
	}
	// 24:00 is the next midnight
	if got := NewDate(2026, 10, 15).At(Midnight, loc); DateOf(got) != NewDate(2026, 10, 16) || got.Hour() != 0 {
		t.Errorf("24:00 = %s, want the 16th at 00:00", got)
	}
}
