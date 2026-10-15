package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

// Week is the week tab: a person's routines and timed dated tasks on a
// seven-day timeline, what has no hours (all day, due, a stretch) as chips in
// the lane under the day heads; it opens on this week (?date= shows the week
// of that day).
type Week struct {
	R     Renderer
	Frame Frame
	Tasks *service.Tasks
	Loc   *time.Location   // SIKE_TZ: every time is wall-clock here
	Now   func() time.Time // the clock, so tests can stop it
}

type weekPage struct {
	Sub              string // "Oct 12 – 18 · week 42"
	Prev, Next, This string // the links ‹ › and Today
	Hours            []hourLine
	From, Span       int // the first hour, and how many the timeline shows
	Days             []weekDay
}

type hourLine struct {
	Label string
	Top   string // % of the timeline
}

type weekDay struct {
	Day, Date string // "Wed", "14"
	Today     bool
	Chips     []weekChip // all day, due, a day of a stretch: the lane
	Blocks    []weekBlock
	Now       string // the now-line's top (%); "" when it is not on this day
	Minute    int    // now, in minutes after midnight, for app.js to move it on
}

type weekBlock struct {
	Name, Time, Color, Href string
	Top, Height             string // % of the timeline
	Left, Width             string // % of the day's column (clock.Layout)
	Inset                   int    // px from the left: nesting
}

type weekChip struct {
	Text, Color, Href string
	Due               bool
}

// nestInset is how far a nested block sits in from what holds it (figma: 40px).
const nestInset = 40

func (h Week) Show(w http.ResponseWriter, r *http.Request) { h.Page(w, r) }

func (h Week) Page(w http.ResponseWriter, r *http.Request, opts ...Option) {
	u := UserFrom(r.Context())
	now := h.Now().In(h.Loc)
	today := clock.DateOf(now)
	day := today
	if d, err := clock.ParseDate(r.URL.Query().Get("date")); err == nil {
		day = d
	}
	monday := day.Monday()
	end := monday.AddDays(7)

	tasks, err := h.Tasks.List(r.Context(), u.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	places, err := h.Tasks.Week(r.Context(), u.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	byID := map[int64]model.Task{}
	for _, t := range tasks {
		byID[t.ID] = t
	}
	taskOf := map[int64]model.Task{} // placement → its task
	var weekly []clock.Weekly
	for _, p := range places {
		weekly = append(weekly, p.Weekly())
		taskOf[p.ID] = byID[p.TaskID]
	}
	var dated []clock.Dated
	for _, t := range tasks {
		if t.Date != nil {
			dated = append(dated, t.Date.Clock(t.ID))
		}
	}
	// a block on the timeline: a placement's occurrence, or a timed dated one
	type item struct {
		occ  clock.Occurrence
		task model.Task
	}
	var timed []item
	var occs []clock.Occurrence
	for _, o := range clock.Expand(weekly, monday, end) {
		timed = append(timed, item{o, taskOf[o.Ref]})
		occs = append(occs, o)
	}
	var chips []item
	for _, o := range clock.ExpandDated(dated, monday, end) {
		if o.Kind == clock.Timed {
			timed = append(timed, item{o, byID[o.Ref]})
			occs = append(occs, o)
		} else {
			chips = append(chips, item{o, byID[o.Ref]})
		}
	}
	from, to := clock.HourRange(occs)
	span := to - from
	pct := func(t clock.Time) float64 { return float64(int(t)-from*60) / float64(span*60) * 100 }

	page := weekPage{
		Sub:  fmt.Sprintf("%s · week %d", weekRange(monday, end.AddDays(-1)), monday.ISOWeek()),
		Prev: "/week?date=" + monday.AddDays(-7).String(),
		Next: "/week?date=" + end.String(),
		This: "/week",
		From: from,
		Span: span,
	}
	for hr := from; hr < to; hr++ {
		page.Hours = append(page.Hours, hourLine{Label: fmt.Sprintf("%02d:00", hr), Top: percent(pct(clock.Time(hr * 60)))})
	}
	for d := monday; d.Before(end); d = d.AddDays(1) {
		wd := weekDay{Day: d.Weekday().String()[:3], Date: fmt.Sprint(d.Day), Today: d == today}
		for _, c := range chips {
			if c.occ.Date != d {
				continue
			}
			chip := weekChip{Text: c.task.Name, Color: c.task.Color, Href: fmt.Sprintf("/week/%d", c.task.ID)}
			if c.occ.Kind == clock.Deadline {
				chip.Due, chip.Text = true, strings.TrimSpace(c.task.Name+" · due "+c.occ.End.String())
			}
			wd.Chips = append(wd.Chips, chip)
		}
		var day []item
		var spans []clock.Occurrence
		for _, it := range timed {
			if it.occ.Date == d {
				day = append(day, it)
				spans = append(spans, it.occ)
			}
		}
		slots, order := clock.Layout(spans)
		for _, i := range order { // what is nested is drawn over what holds it
			o, sl, t := day[i].occ, slots[i], day[i].task
			wd.Blocks = append(wd.Blocks, weekBlock{
				Name: t.Name, Color: t.Color, Href: fmt.Sprintf("/week/%d", t.ID),
				Time: o.Start.String() + "–" + o.End.String(),
				Top:  percent(pct(o.Start)), Height: percent(pct(o.End) - pct(o.Start)),
				Left: percent(sl.Left * 100), Width: percent(sl.Width * 100), Inset: nestInset * sl.Depth,
			})
		}
		if wd.Today {
			minute := clock.Time(now.Hour()*60 + now.Minute())
			if minute >= clock.Time(from*60) && minute <= clock.Time(to*60) {
				wd.Now, wd.Minute = percent(pct(minute)), int(minute)
			}
		}
		page.Days = append(page.Days, wd)
	}

	shell, err := h.Frame.Shell(r, "week")
	if err != nil {
		serverError(w, err)
		return
	}
	h.R.Render(w, "week", page, append([]Option{shell}, opts...)...)
}

func percent(f float64) string { return fmt.Sprintf("%.4f", f) }

// weekRange is "Oct 12 – 18", "Sep 28 – Oct 4" across months.
func weekRange(first, last clock.Date) string {
	if first.Month == last.Month {
		return fmt.Sprintf("%s %d – %d", first.Month.String()[:3], first.Day, last.Day)
	}
	return fmt.Sprintf("%s %d – %s %d", first.Month.String()[:3], first.Day, last.Month.String()[:3], last.Day)
}
