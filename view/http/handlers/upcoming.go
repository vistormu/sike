package handlers

import (
	"cmp"
	"fmt"
	"net/http"
	"slices"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

// Upcoming is the upcoming tab: dated tasks, nearest first, grouped by period.
// what has gone by waits in a collapsed "Past", most recent first.
type Upcoming struct {
	R     Renderer
	Frame Frame
	Tasks *service.Tasks
	Loc   *time.Location
	Now   func() time.Time
}

type upcomingPage struct {
	Groups []upcomingGroup
	Past   []upcomingRow
}

type upcomingGroup struct {
	Name string // "This week", "Next week", "November", "January 2027"
	Rows []upcomingRow
}

type upcomingRow struct {
	Name, Color, Meta, Href string
	Near                    bool // a deadline soon: the meta in danger
}

// nearDays: a deadline this close is near (until settings has the person's
// own warning, m23).
const nearDays = 2

func (h Upcoming) Show(w http.ResponseWriter, r *http.Request) { h.Page(w, r) }

func (h Upcoming) Page(w http.ResponseWriter, r *http.Request, opts ...Option) {
	now := h.Now().In(h.Loc)
	today, minute := clock.DateOf(now), clock.Time(now.Hour()*60+now.Minute())
	tasks, err := h.Tasks.List(r.Context(), UserFrom(r.Context()).ID)
	if err != nil {
		serverError(w, err)
		return
	}
	var coming, past []model.Task
	for _, t := range tasks {
		switch {
		case t.When != model.Dated || t.Date == nil:
		case t.Date.Over(today, minute):
			past = append(past, t)
		default:
			coming = append(coming, t)
		}
	}
	// nearest first: by the day it shows on, then the hour (all day first)
	key := func(t model.Task) (clock.Date, clock.Time) {
		d := t.Date
		at := d.Start
		if d.Deadline() {
			at = d.End
		}
		return d.Shows(today), at
	}
	slices.SortStableFunc(coming, func(a, b model.Task) int {
		da, ta := key(a)
		db, tb := key(b)
		if da != db {
			return compareDates(da, db)
		}
		return cmp.Compare(ta, tb)
	})
	// the past, most recent first: by its last day
	slices.SortStableFunc(past, func(a, b model.Task) int {
		return cmp.Or(compareDates(b.Date.To, a.Date.To), cmp.Compare(b.Date.End, a.Date.End))
	})

	var page upcomingPage
	for _, t := range coming {
		name := period(t.Date.Shows(today), today)
		if n := len(page.Groups); n == 0 || page.Groups[n-1].Name != name {
			page.Groups = append(page.Groups, upcomingGroup{Name: name})
		}
		g := &page.Groups[len(page.Groups)-1]
		g.Rows = append(g.Rows, h.row(t, today, true))
	}
	for _, t := range past {
		page.Past = append(page.Past, h.row(t, today, false))
	}

	shell, err := h.Frame.Shell(r, "upcoming")
	if err != nil {
		serverError(w, err)
		return
	}
	h.R.Render(w, "upcoming", page, append([]Option{shell}, opts...)...)
}

func compareDates(a, b clock.Date) int {
	switch {
	case a == b:
		return 0
	case a.Before(b):
		return -1
	}
	return 1
}

// period names the group a day falls in, seen from today.
func period(d, today clock.Date) string {
	monday := today.Monday()
	switch {
	case d.Before(monday.AddDays(7)):
		return "This week"
	case d.Before(monday.AddDays(14)):
		return "Next week"
	case d.Year == today.Year:
		return d.Month.String()
	}
	return fmt.Sprintf("%s %d", d.Month, d.Year)
}

// row writes a dated task's meta line, as figma does: "Tomorrow · Thu 15 ·
// 11:00–12:00", "Due Fri 16 · in 2 days", "Mon 3 – Fri 7 · 5 days",
// "Sat 21 · all day". coming rows say today and tomorrow, and near deadlines
// how soon.
func (h Upcoming) row(t model.Task, today clock.Date, coming bool) upcomingRow {
	d := *t.Date
	row := upcomingRow{Name: t.Name, Color: t.Color, Href: fmt.Sprintf("/upcoming/%d", t.ID)}
	day := func(x clock.Date) string { return fmt.Sprintf("%s %d", x.Weekday().String()[:3], x.Day) }
	soon := func(x clock.Date) string {
		switch {
		case !coming:
			return ""
		case x == today:
			return "Today · "
		case x == today.AddDays(1):
			return "Tomorrow · "
		}
		return ""
	}
	switch {
	case d.Deadline():
		row.Meta = "Due " + day(d.To)
		if d.End != clock.NoTime {
			row.Meta += " · " + d.End.String()
		}
		if days := daysBetween(today, d.To); coming && days <= nearDays {
			row.Near = true
			row.Meta += " · " + map[int]string{0: "today", 1: "tomorrow"}[days]
			if days > 1 {
				row.Meta += fmt.Sprintf("in %d days", days)
			}
		}
	case d.From != d.To:
		row.Meta = fmt.Sprintf("%s – %s · %d days", day(d.From), day(d.To), daysBetween(d.From, d.To)+1)
	case d.AllDay():
		row.Meta = soon(d.From) + day(d.From) + " · all day"
	default:
		row.Meta = soon(d.From) + day(d.From) + " · " + d.Start.String() + "–" + d.End.String()
	}
	return row
}

func daysBetween(a, b clock.Date) int {
	n := 0
	for ; a.Before(b); a = a.AddDays(1) {
		n++
	}
	return n
}
