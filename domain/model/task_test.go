package model

import (
	"testing"

	"sike/domain/clock"
)

func TestDatesOverAndShows(t *testing.T) {
	today := clock.NewDate(2026, 10, 14)
	day := func(n int) clock.Date { return clock.NewDate(2026, 10, n) }
	no := clock.NoTime
	tests := []struct {
		name  string
		d     Dates
		now   clock.Time
		over  bool
		shows clock.Date
	}{
		{"yesterday's deadline", Dates{To: day(13), Start: no, End: no}, 9 * 60, true, today},
		{"today's deadline, no hour", Dates{To: day(14), Start: no, End: no}, 23 * 60, false, day(14)},
		{"today's deadline at 14:00, at 13:59", Dates{To: day(14), Start: no, End: 14 * 60}, 14*60 - 1, false, day(14)},
		{"today's deadline at 14:00, at 14:00", Dates{To: day(14), Start: no, End: 14 * 60}, 14 * 60, true, day(14)},
		{"the dentist tomorrow", Dates{From: day(15), To: day(15), Start: 11 * 60, End: 12 * 60}, 23 * 60, false, day(15)},
		{"this morning's dentist", Dates{From: day(14), To: day(14), Start: 9 * 60, End: 10 * 60}, 10 * 60, true, day(14)},
		{"all day today", Dates{From: day(14), To: day(14), Start: no, End: no}, 23*60 + 59, false, day(14)},
		{"a trip under way", Dates{From: day(10), To: day(16), Start: no, End: no}, 12 * 60, false, today},
		{"a trip ending today at 09:00", Dates{From: day(10), To: day(14), Start: 18 * 60, End: 9 * 60}, 9 * 60, true, today},
		{"next month", Dates{From: clock.NewDate(2026, 11, 3), To: clock.NewDate(2026, 11, 7), Start: no, End: no}, 0, false, clock.NewDate(2026, 11, 3)},
	}
	for _, tt := range tests {
		if got := tt.d.Over(today, tt.now); got != tt.over {
			t.Errorf("%s: over = %v, want %v", tt.name, got, tt.over)
		}
		if got := tt.d.Shows(today); got != tt.shows {
			t.Errorf("%s: shows on %s, want %s", tt.name, got, tt.shows)
		}
	}
}
