package clock

import (
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
)

func tm(s string) Time {
	t, err := ParseTime(s)
	if err != nil {
		panic(err)
	}
	return t
}

func occ(o Occurrence) string {
	return fmt.Sprintf("%d %s %s–%s", o.Ref, o.Date.Weekday().String()[:3], o.Start, o.End)
}

func TestExpand(t *testing.T) {
	gym := Weekly{Ref: 1, Days: []time.Weekday{time.Wednesday, time.Saturday}, Start: tm("18:00"), End: tm("19:30")}
	work := Weekly{Ref: 2, Days: []time.Weekday{time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday}, Start: tm("09:00"), End: tm("17:00")}
	late := Weekly{Ref: 3, Days: []time.Weekday{time.Wednesday}, Start: tm("09:00"), End: tm("10:00")}
	ws := []Weekly{gym, work, late}

	tests := []struct {
		name     string
		from, to Date
		want     []string
	}{
		{"the dst week (spring)", NewDate(2026, 3, 23), NewDate(2026, 3, 30), []string{
			"2 Mon 09:00–17:00", "2 Tue 09:00–17:00",
			"3 Wed 09:00–10:00", "2 Wed 09:00–17:00", "1 Wed 18:00–19:30", // by start, then end
			"2 Thu 09:00–17:00", "2 Fri 09:00–17:00", "1 Sat 18:00–19:30",
		}},
		{"a weekend", NewDate(2026, 10, 17), NewDate(2026, 10, 19), []string{"1 Sat 18:00–19:30"}},
		{"to is not included", NewDate(2026, 10, 14), NewDate(2026, 10, 14), nil},
		{"one day", NewDate(2026, 10, 15), NewDate(2026, 10, 16), []string{"2 Thu 09:00–17:00"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []string
			for _, o := range Expand(ws, tt.from, tt.to) {
				got = append(got, occ(o))
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(tt.want, "\n  "))
			}
		})
	}
	if got := Expand(nil, NewDate(2026, 10, 12), NewDate(2026, 10, 19)); len(got) != 0 {
		t.Errorf("nothing placed: %d occurrences", len(got))
	}
}

func TestHourRange(t *testing.T) {
	o := func(start, end string) Occurrence { return Occurrence{Start: tm(start), End: tm(end)} }
	tests := []struct {
		name     string
		occs     []Occurrence
		from, to int
	}{
		{"nothing", nil, 7, 22},
		{"inside", []Occurrence{o("09:00", "17:00"), o("07:00", "22:00")}, 7, 22},
		{"an early taxi", []Occurrence{o("05:30", "06:00")}, 5, 22},
		{"a late end rounds up", []Occurrence{o("21:00", "22:30")}, 7, 23},
		{"to midnight", []Occurrence{o("20:00", "24:00")}, 7, 24},
		{"both ways", []Occurrence{o("06:15", "07:00"), o("22:00", "23:01")}, 6, 24},
	}
	for _, tt := range tests {
		if from, to := HourRange(tt.occs); from != tt.from || to != tt.to {
			t.Errorf("%s: %d–%d, want %d–%d", tt.name, from, to, tt.from, tt.to)
		}
	}
}

func TestExpandDated(t *testing.T) {
	d := func(s string) Date {
		x, err := ParseDate(s)
		if err != nil {
			panic(err)
		}
		return x
	}
	ds := []Dated{
		{Ref: 1, From: d("2026-10-14"), To: d("2026-10-14"), Start: tm("11:00"), End: tm("12:00")}, // dentist
		{Ref: 2, From: d("2026-10-14"), To: d("2026-10-14"), Start: NoTime, End: NoTime},           // a birthday
		{Ref: 3, To: d("2026-10-14"), Start: NoTime, End: tm("14:00")},                             // report due 14:00
		{Ref: 4, To: d("2026-10-16"), Start: NoTime, End: NoTime},                                  // taxes due
		{Ref: 5, From: d("2026-10-10"), To: d("2026-10-13"), Start: NoTime, End: NoTime},           // a trip, from last week
		{Ref: 6, From: d("2026-10-17"), To: d("2026-10-25"), Start: tm("18:00"), End: tm("20:00")}, // into next week: chips
		{Ref: 7, From: d("2026-10-20"), To: d("2026-10-20"), Start: tm("09:00"), End: tm("10:00")}, // next week
	}
	var got []string
	for _, o := range ExpandDated(ds, d("2026-10-12"), d("2026-10-19")) {
		got = append(got, fmt.Sprintf("%d %s %s %d %s-%s", o.Ref, o.Date.Weekday().String()[:3], o.Date, o.Kind, o.Start, o.End))
	}
	want := []string{
		"5 Mon 2026-10-12 1 -", "5 Tue 2026-10-13 1 -",
		"2 Wed 2026-10-14 1 -", "3 Wed 2026-10-14 2 -14:00", "1 Wed 2026-10-14 0 11:00-12:00",
		"4 Fri 2026-10-16 2 -",
		"6 Sat 2026-10-17 1 -", "6 Sun 2026-10-18 1 -",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got\n  %s\nwant\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	// only the timed stretch the hours
	from, to := HourRange(ExpandDated([]Dated{
		{Ref: 1, From: d("2026-10-14"), To: d("2026-10-14"), Start: tm("05:00"), End: tm("06:00")},
		{Ref: 2, To: d("2026-10-14"), Start: NoTime, End: tm("23:30")},
	}, d("2026-10-12"), d("2026-10-19")))
	if from != 5 || to != 22 {
		t.Errorf("hours %d–%d, want 05–22: a deadline's hour does not stretch them", from, to)
	}
}
