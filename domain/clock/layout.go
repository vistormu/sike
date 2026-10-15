package clock

import (
	"cmp"
	"slices"
)

// Slot is where an occurrence sits across its day's column: Left and Width
// are fractions of the column, Depth how many blocks it is nested in (the view
// insets it that much more).
type Slot struct {
	Left, Width float64
	Depth       int
}

// Layout places one day's occurrences: overlap is nesting by time
// (product/spec.md). a block inside a longer one is nested in it: it keeps its
// parent's share of the column, one level deeper. blocks that clash, where
// neither holds the other (the same span, or crossing), share their parent's
// share side by side. touching blocks (one ends when the next starts) do not
// clash. slots come back in the order of occs; order is the drawing order,
// parents before what is nested in them.
func Layout(occs []Occurrence) (slots []Slot, order []int) {
	order = make([]int, len(occs))
	for i := range order {
		order[i] = i
	}
	// by start, longer first: a parent comes before what it holds
	slices.SortStableFunc(order, func(a, b int) int {
		return cmp.Or(cmp.Compare(occs[a].Start, occs[b].Start), cmp.Compare(occs[b].End, occs[a].End))
	})

	parent := make([]int, len(occs)) // -1: the column itself
	depth := make([]int, len(occs))
	for k, i := range order {
		parent[i] = -1
		for _, j := range order[:k] { // the innermost earlier block that holds i
			if holds(occs[j], occs[i]) && (parent[i] < 0 || span(occs[j]) < span(occs[parent[i]])) {
				parent[i] = j
			}
		}
		if parent[i] >= 0 {
			depth[i] = depth[parent[i]] + 1
		}
	}

	// lanes among the children of each parent: clashing siblings go side by
	// side, as many lanes as their busiest cluster needs
	lane := make([]int, len(occs))
	lanes := make([]int, len(occs))
	children := map[int][]int{}
	for _, i := range order {
		children[parent[i]] = append(children[parent[i]], i)
	}
	for _, kids := range children {
		var cluster []int // the siblings clashing, directly or through others
		var end Time      // the cluster's latest end
		flush := func() {
			n := 0
			for _, i := range cluster {
				n = max(n, lane[i]+1)
			}
			for _, i := range cluster {
				lanes[i] = n
			}
			cluster = nil
		}
		for _, i := range kids { // kids are in start order
			if len(cluster) > 0 && occs[i].Start >= end {
				flush()
			}
			if len(cluster) == 0 {
				end = occs[i].End
			}
			// the first lane free of every clashing sibling in the cluster
			used := map[int]bool{}
			for _, j := range cluster {
				if occs[j].End > occs[i].Start {
					used[lane[j]] = true
				}
			}
			for used[lane[i]] {
				lane[i]++
			}
			cluster = append(cluster, i)
			end = max(end, occs[i].End)
		}
		flush()
	}

	slots = make([]Slot, len(occs))
	for _, i := range order { // parents are placed before their children
		left, width := 0.0, 1.0
		if p := parent[i]; p >= 0 {
			left, width = slots[p].Left, slots[p].Width
		}
		w := width / float64(lanes[i])
		slots[i] = Slot{Left: left + float64(lane[i])*w, Width: w, Depth: depth[i]}
	}
	return slots, order
}

// holds: o lies within p, and p is longer.
func holds(p, o Occurrence) bool {
	return p.Start <= o.Start && o.End <= p.End && span(p) > span(o)
}

func span(o Occurrence) Time { return o.End - o.Start }
