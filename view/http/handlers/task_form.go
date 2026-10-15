package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

// TaskForm is the task form, to create a task or edit one. it opens over the
// tab it was asked from (a side panel on desktop, the whole screen on the
// phone); a new task gets When preset from that tab. a bad form comes back as
// typed, with the reason in an error toast.
type TaskForm struct {
	Tasks *service.Tasks
	Areas *service.Areas
	Loc   *time.Location // dates are read and shown relative to today here
	Now   func() time.Time
}

type taskForm struct {
	Action, Close string
	Delete        string // the delete url; "" for a new task
	Name, Color   string
	When          model.When
	Areas         []model.Area
	Whens         []whenOption
	Places        []placeRow // the weekly schedule, one box per time
	Blank         placeRow   // the box "+ Add time" copies (app.js)
	Dates         dateFields // on a date
}

// dateFields is the Date box as typed or shown: dates as "Thu, Oct 15",
// times as "11:00". no start date: a deadline.
type dateFields struct {
	From, Start, To, End string
	AllDay, Notify       bool
}

// blankDates: nothing yet, notify at start on (as figma draws it).
var blankDates = dateFields{Notify: true}

func (h TaskForm) today() clock.Date { return clock.Today(h.Now(), h.Loc) }

// storedDates shows a task's dates; a task with none gets a blank box.
func (h TaskForm) storedDates(d *model.Dates) dateFields {
	if d == nil {
		return blankDates
	}
	today := h.today()
	return dateFields{From: d.From.Short(today), Start: d.Start.String(), To: d.To.Short(today), End: d.End.String(),
		AllDay: d.AllDay(), Notify: d.Notify}
}

func typedDates(r *http.Request) dateFields {
	return dateFields{From: r.FormValue("date_from"), Start: r.FormValue("time_from"), To: r.FormValue("date_to"),
		End: r.FormValue("time_to"), AllDay: r.FormValue("all_day") != "", Notify: r.FormValue("notify") != ""}
}

// dates reads the Date box. empty is none: no start date is a deadline, no
// times all day; "All day" drops what was typed in the times.
func (h TaskForm) dates(f dateFields) (*model.Dates, error) {
	today := h.today()
	d := model.Dates{Start: clock.NoTime, End: clock.NoTime, Notify: f.Notify}
	var err error
	for _, x := range []struct {
		in  string
		out *clock.Date
	}{{f.From, &d.From}, {f.To, &d.To}} {
		if strings.TrimSpace(x.in) == "" {
			continue
		}
		if *x.out, err = clock.ReadDate(x.in, today); err != nil {
			return nil, model.Invalid("Dates are like 15/10 or 15/10/2026")
		}
	}
	if f.AllDay {
		return &d, nil
	}
	for _, x := range []struct {
		in  string
		out *clock.Time
	}{{f.Start, &d.Start}, {f.End, &d.End}} {
		if strings.TrimSpace(x.in) == "" {
			continue
		}
		if *x.out, err = clock.ReadTime(x.in); err != nil {
			return nil, model.Invalid("Times are HH:MM, 24h")
		}
	}
	return &d, nil
}

type whenOption struct {
	Value model.When
	Label string
}

var whenLabels = map[model.When]string{
	model.Weekly: "Every week", model.Dated: "On a date", model.Pending: "Pending", model.Someday: "Someday",
}

// placeRow is one time in the weekly schedule, as the form shows it. I names
// its inputs: days-I, start-I, end-I, notify_start-I, notify_end-I.
type placeRow struct {
	I                      string
	Days                   []dayOption
	Start, End             string
	NotifyStart, NotifyEnd bool
}

type dayOption struct {
	Value time.Weekday
	Label string
	On    bool
}

var dayLetters = map[time.Weekday]string{
	time.Monday: "M", time.Tuesday: "T", time.Wednesday: "W", time.Thursday: "T",
	time.Friday: "F", time.Saturday: "S", time.Sunday: "S",
}

func newRow(i string, days model.Days, start, end string, notifyStart, notifyEnd bool) placeRow {
	row := placeRow{I: i, Start: start, End: end, NotifyStart: notifyStart, NotifyEnd: notifyEnd}
	for _, d := range model.Week {
		row.Days = append(row.Days, dayOption{Value: d, Label: dayLetters[d], On: days.Has(d)})
	}
	return row
}

// blankRow is a new time: no days yet, notify at start on (as figma draws it).
func blankRow(i string) placeRow { return newRow(i, 0, "", "", true, false) }

// storedRows shows a task's placements; a task with none gets a blank box.
func storedRows(places []model.Placement) []placeRow {
	var rows []placeRow
	for i, p := range places {
		rows = append(rows, newRow(strconv.Itoa(i), p.Days, p.Start.String(), p.End.String(), p.NotifyStart, p.NotifyEnd))
	}
	if len(rows) == 0 {
		rows = []placeRow{blankRow("0")}
	}
	return rows
}

// typedRows reads the schedule back from the form: one "place" value per box.
func typedRows(r *http.Request) []placeRow {
	var rows []placeRow
	for _, i := range r.Form["place"] {
		var days model.Days
		for _, v := range r.Form["days-"+i] {
			if d, err := strconv.Atoi(v); err == nil && d >= 0 && d < 7 {
				days |= model.DaysOf(time.Weekday(d))
			}
		}
		rows = append(rows, newRow(i, days, r.FormValue("start-"+i), r.FormValue("end-"+i),
			r.FormValue("notify_start-"+i) != "", r.FormValue("notify_end-"+i) != ""))
	}
	if len(rows) == 0 {
		rows = []placeRow{blankRow("0")}
	}
	return rows
}

// placements turns the boxes into placements. a box with no days is no time:
// unticking every day removes it.
func placements(rows []placeRow) ([]model.Placement, error) {
	var places []model.Placement
	for _, row := range rows {
		var p model.Placement
		for _, d := range row.Days {
			if d.On {
				p.Days |= model.DaysOf(d.Value)
			}
		}
		if p.Days == 0 {
			continue
		}
		if row.Start == "" || row.End == "" {
			return nil, model.Invalid("A time needs a start and an end")
		}
		var err1, err2 error
		p.Start, err1 = clock.ReadTime(row.Start)
		p.End, err2 = clock.ReadTime(row.End)
		if err1 != nil || err2 != nil {
			return nil, model.Invalid("Times are HH:MM, 24h")
		}
		p.NotifyStart, p.NotifyEnd = row.NotifyStart, row.NotifyEnd
		places = append(places, p)
	}
	return places, nil
}

// New opens an empty form over tab.
func (h TaskForm) New(tab string, page Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var preset model.When
		for when, t := range tabOf {
			if t == tab {
				preset = when
			}
		}
		f, err := h.form(r, tab, "new", model.Task{When: preset}, []placeRow{blankRow("0")}, blankDates)
		if err != nil {
			serverError(w, err)
			return
		}
		page(w, r, WithForm(f))
	}
}

// Create adds the task and opens the tab it lands in.
func (h TaskForm) Create(tab string, page Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		t, err := h.typed(r)
		if err == nil {
			t.UserID = UserFrom(r.Context()).ID
			t, err = h.Tasks.Add(r.Context(), t)
			if err == nil {
				http.Redirect(w, r, "/"+tabOf[t.When], http.StatusSeeOther)
				return
			}
		}
		if errors.Is(err, model.ErrInvalid) {
			h.retry(w, r, tab, "new", page, err)
			return
		}
		serverError(w, err)
	}
}

// Edit opens a task's form over tab.
func (h TaskForm) Edit(tab string, page Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		u := UserFrom(r.Context())
		t, err := h.Tasks.Get(r.Context(), u.ID, id)
		if errors.Is(err, model.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		f, err := h.form(r, tab, r.PathValue("id"), t, storedRows(t.Places), h.storedDates(t.Date))
		if err != nil {
			serverError(w, err)
			return
		}
		page(w, r, WithForm(f))
	}
}

// Update saves a task and opens the tab it lands in: a new when moves it
// (decisions/0005).
func (h TaskForm) Update(tab string, page Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		t, err := h.typed(r)
		if err == nil {
			t.ID, t.UserID = id, UserFrom(r.Context()).ID
			t, err = h.Tasks.Edit(r.Context(), t)
		}
		switch {
		case errors.Is(err, model.ErrInvalid):
			h.retry(w, r, tab, r.PathValue("id"), page, err)
		case errors.Is(err, model.ErrNotFound):
			http.NotFound(w, r)
		case err != nil:
			serverError(w, err)
		default:
			http.Redirect(w, r, "/"+tabOf[t.When], http.StatusSeeOther)
		}
	}
}

// typed reads the task as the form sent it: for a weekly task its schedule,
// for a dated one its dates. another when ignores those boxes, whatever was
// left in them.
func (h TaskForm) typed(r *http.Request) (model.Task, error) {
	t := model.Task{Name: r.FormValue("name"), Color: r.FormValue("area"), When: model.When(r.FormValue("when"))}
	var err error
	switch t.When {
	case model.Weekly:
		t.Places, err = placements(typedRows(r))
	case model.Dated:
		t.Date, err = h.dates(typedDates(r))
	}
	return t, err
}

// Delete deletes the task (done) and opens the tab with the undo toast.
func (h TaskForm) Delete(tab string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		err = h.Tasks.Remove(r.Context(), UserFrom(r.Context()).ID, id)
		if errors.Is(err, model.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/"+tab+"?deleted="+r.PathValue("id"), http.StatusSeeOther)
	}
}

// Restore undoes the delete. too late (a later delete erased it): the tab says
// so in an error toast.
func (h TaskForm) Restore(tab string, page Page) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		err = h.Tasks.Restore(r.Context(), UserFrom(r.Context()).ID, id)
		if errors.Is(err, model.ErrNotFound) {
			page(w, r, WithToast(http.StatusConflict, Toast{Message: "Too late to undo", Error: true}))
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/"+tab, http.StatusSeeOther)
	}
}

// retry shows the form again as it was typed, with why it failed.
func (h TaskForm) retry(w http.ResponseWriter, r *http.Request, tab, path string, page Page, reason error) {
	typed := model.Task{Name: r.FormValue("name"), Color: r.FormValue("area"), When: model.When(r.FormValue("when"))}
	f, err := h.form(r, tab, path, typed, typedRows(r), typedDates(r))
	if err != nil {
		serverError(w, err)
		return
	}
	page(w, r, WithForm(f), WithToast(http.StatusUnprocessableEntity, Toast{Message: reason.Error(), Error: true}))
}

// form posts to /{tab}/{path}: "new", or the task's id. t gives the name,
// area and when; the boxes come as shown or typed.
func (h TaskForm) form(r *http.Request, tab, path string, t model.Task, rows []placeRow, dates dateFields) (taskForm, error) {
	areas, err := h.Areas.List(r.Context(), UserFrom(r.Context()).ID)
	if err != nil {
		return taskForm{}, err
	}
	f := taskForm{Action: "/" + tab + "/" + path, Close: "/" + tab, Name: t.Name, Color: t.Color, When: t.When,
		Areas: areas, Places: rows, Blank: blankRow("__i__"), Dates: dates}
	if path != "new" {
		f.Delete = f.Action + "/delete"
	}
	for _, w := range h.Tasks.Whens() {
		f.Whens = append(f.Whens, whenOption{Value: w, Label: whenLabels[w]})
	}
	return f, nil
}
