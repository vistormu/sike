package handlers

import (
	"errors"
	"net/http"

	"sike/domain/model"
	"sike/domain/service"
)

// TaskForm is the task form. it opens over the tab it was asked from (a side
// panel on desktop, the whole screen on the phone), with When preset from that
// tab. a bad form comes back as typed, with the reason in an error toast.
type TaskForm struct {
	Tasks *service.Tasks
	Areas *service.Areas
}

type taskForm struct {
	Action, Close string
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
		f, err := h.form(r, tab, "", "", preset)
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
			f, ferr := h.form(r, tab, name, color, when)
			if ferr != nil {
				serverError(w, ferr)
				return
			}
			page(w, r, WithForm(f), WithToast(http.StatusUnprocessableEntity, Toast{Message: err.Error(), Error: true}))
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		http.Redirect(w, r, "/"+tabOf[t.When], http.StatusSeeOther)
	}
}

func (h TaskForm) form(r *http.Request, tab, name, color string, when model.When) (taskForm, error) {
	areas, err := h.Areas.List(r.Context(), UserFrom(r.Context()).ID)
	if err != nil {
		return taskForm{}, err
	}
	f := taskForm{Action: "/" + tab + "/new", Close: "/" + tab, Name: name, Color: color, When: when, Areas: areas}
	for _, w := range h.Tasks.Whens() {
		f.Whens = append(f.Whens, whenOption{Value: w, Label: whenLabels[w]})
	}
	return f, nil
}
