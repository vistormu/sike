package handlers

import (
	"fmt"
	"net/http"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

// Week is the week tab: a person's routines on a seven-day timeline, opening
// on this week (?date= shows the week of that day).
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
	Blocks    []weekBlock
	Now       string // the now-line's top (%); "" when it is not on this day
	Minute    int    // now, in minutes after midnight, for app.js to move it on
}

type weekBlock struct {
	Name, Time, Color, Href string
	Top, Height             string // % of the timeline
}

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
	occs := clock.Expand(weekly, monday, end)
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
		for _, o := range occs {
			if o.Date != d {
				continue
			}
			t := taskOf[o.Ref]
			wd.Blocks = append(wd.Blocks, weekBlock{
				Name: t.Name, Color: t.Color, Href: fmt.Sprintf("/week/%d", t.ID),
				Time: o.Start.String() + "–" + o.End.String(),
				Top:  percent(pct(o.Start)), Height: percent(pct(o.End) - pct(o.Start)),
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
