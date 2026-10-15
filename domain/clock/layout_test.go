package clock

import (
	"fmt"
	"strings"
	"testing"
)

func TestLayout(t *testing.T) {
	o := func(start, end string) Occurrence { return Occurrence{Start: tm(start), End: tm(end)} }
	// each slot as "left+width@depth", in thirds and halves rounded to 3 places
	tests := []struct {
		name string
		occs []Occurrence
		want []string
	}{
		{"nothing", nil, nil},
		{"alone", []Occurrence{o("09:00", "17:00")}, []string{"0+1@0"}},
		{"apart", []Occurrence{o("09:00", "10:00"), o("18:00", "19:00")}, []string{"0+1@0", "0+1@0"}},
		{"touching do not clash", []Occurrence{o("09:00", "10:00"), o("10:00", "11:00")}, []string{"0+1@0", "0+1@0"}},
		{"work and the dentist (figma)", []Occurrence{o("11:00", "12:00"), o("09:00", "17:00")}, []string{"0+1@1", "0+1@0"}},
		{"same start, shorter is inside", []Occurrence{o("09:00", "17:00"), o("09:00", "10:00")}, []string{"0+1@0", "0+1@1"}},
		{"nested twice", []Occurrence{o("09:00", "17:00"), o("10:00", "14:00"), o("11:00", "12:00")}, []string{"0+1@0", "0+1@1", "0+1@2"}},
		{"equal clash", []Occurrence{o("18:00", "19:00"), o("18:00", "19:00")}, []string{"0+0.5@0", "0.5+0.5@0"}},
		{"crossing", []Occurrence{o("09:00", "11:00"), o("10:00", "12:00")}, []string{"0+0.5@0", "0.5+0.5@0"}},
		{"a chain reuses the free lane", []Occurrence{o("09:00", "11:00"), o("10:00", "12:00"), o("11:00", "13:00")},
			[]string{"0+0.5@0", "0.5+0.5@0", "0+0.5@0"}},
		{"three at once", []Occurrence{o("09:00", "10:00"), o("09:00", "10:00"), o("09:30", "10:30")},
			[]string{"0+0.333@0", "0.333+0.333@0", "0.667+0.333@0"}},
		{"a clash inside a parent splits the parent", []Occurrence{o("09:00", "17:00"), o("11:00", "12:00"), o("11:30", "12:30")},
			[]string{"0+1@0", "0+0.5@1", "0.5+0.5@1"}},
		{"nested in one of two clashing", []Occurrence{o("09:00", "17:00"), o("16:00", "18:00"), o("16:15", "16:45")},
			// 16:15–16:45 lies in both; the shorter (16–18) is its parent
			[]string{"0+0.5@0", "0.5+0.5@0", "0.5+0.5@1"}},
		{"clusters are separate", []Occurrence{o("09:00", "10:00"), o("09:00", "10:00"), o("15:00", "16:00")},
			[]string{"0+0.5@0", "0.5+0.5@0", "0+1@0"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots, order := Layout(tt.occs)
			var got []string
			for _, s := range slots {
				got = append(got, fmt.Sprintf("%s+%s@%d", short(s.Left), short(s.Width), s.Depth))
			}
			if strings.Join(got, " ") != strings.Join(tt.want, " ") {
				t.Errorf("got  %v\nwant %v", got, tt.want)
			}
			// drawing order: every block after the ones it is nested in
			drawn := map[int]bool{}
			for _, i := range order {
				for j := range tt.occs {
					if holds(tt.occs[j], tt.occs[i]) && !drawn[j] {
						t.Errorf("block %d is drawn before %d, which holds it", i, j)
					}
				}
				drawn[i] = true
			}
		})
	}
}

func short(f float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", f), "0"), ".")
}
