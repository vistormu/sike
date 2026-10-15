package handlers

import (
	"errors"
	"net/http"
	"strconv"

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
}

type taskForm struct {
	Action, Close string
	Delete        string // the delete url; "" for a new task
	Name, Color   string
	When          model.When
	Areas         []model.Area
	Whens         []whenOption
}

type whenOption struct {
	Value model.When
	Label string
}

var whenLabels = map[model.When]string{
	model.Weekly: "Every week", model.Dated: "On a date", model.Pending: "Pending", model.Someday: "Someday",
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
		f, err := h.form(r, tab, "new", "", "", preset)
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
		u := UserFrom(r.Context())
		name, color, when := r.FormValue("name"), r.FormValue("area"), model.When(r.FormValue("when"))
		t, err := h.Tasks.Add(r.Context(), u.ID, name, color, when)
		if errors.Is(err, model.ErrInvalid) {
			h.retry(w, r, tab, "new", page, err)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/"+tabOf[t.When], http.StatusSeeOther)
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
		t, err := h.Tasks.Get(r.Context(), UserFrom(r.Context()).ID, id)
		if errors.Is(err, model.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		f, err := h.form(r, tab, r.PathValue("id"), t.Name, t.Color, t.When)
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
		u := UserFrom(r.Context())
		t, err := h.Tasks.Edit(r.Context(), u.ID, id, r.FormValue("name"), r.FormValue("area"), model.When(r.FormValue("when")))
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
	f, err := h.form(r, tab, path, r.FormValue("name"), r.FormValue("area"), model.When(r.FormValue("when")))
	if err != nil {
		serverError(w, err)
		return
	}
	page(w, r, WithForm(f), WithToast(http.StatusUnprocessableEntity, Toast{Message: reason.Error(), Error: true}))
}

// form posts to /{tab}/{path}: "new", or the task's id.
func (h TaskForm) form(r *http.Request, tab, path, name, color string, when model.When) (taskForm, error) {
	areas, err := h.Areas.List(r.Context(), UserFrom(r.Context()).ID)
	if err != nil {
		return taskForm{}, err
	}
	f := taskForm{Action: "/" + tab + "/" + path, Close: "/" + tab, Name: name, Color: color, When: when, Areas: areas}
	if path != "new" {
		f.Delete = f.Action + "/delete"
	}
	for _, w := range h.Tasks.Whens() {
		f.Whens = append(f.Whens, whenOption{Value: w, Label: whenLabels[w]})
	}
	return f, nil
}
