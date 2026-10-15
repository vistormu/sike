// Package handlers holds the http handlers, one file per screen.
package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
	"sike/domain/service"
)

// Renderer writes a page through the layout. options add the shell, a toast,
// a status.
type Renderer interface {
	Render(w http.ResponseWriter, page string, data any, opts ...Option)
}

// View is what the layout needs besides the page's own data.
type View struct {
	Status int
	Toast  *Toast
	Shell  *Shell
	Form   *taskForm
}

type Option func(*View)

// WithToast shows the floating toast (decisions/0007) with a status: a form
// that failed comes back as it was typed.
func WithToast(status int, t Toast) Option {
	return func(v *View) { v.Status, v.Toast = status, &t }
}

// WithShell wraps the page in the sidebar (desktop) or tab bar (phone), with
// tab active. tab is "settings" on settings, which is no tab: the profile is
// active instead. counts are
// shown next to their tab in the sidebar; a missing or zero count hides.
func WithShell(u model.User, tab string, counts map[string]int) Option {
	return func(v *View) {
		s := &Shell{User: u, Settings: tab == "settings"}
		for _, t := range tabs {
			t.Active = t.Name == tab
			if n := counts[t.Name]; n > 0 {
				t.Count = strconv.Itoa(n)
			}
			s.Tabs = append(s.Tabs, t)
		}
		v.Shell = s
	}
}

// WithForm opens the task form over the page.
func WithForm(f taskForm) Option {
	return func(v *View) { v.Form = &f }
}

// Page renders a tab with extra options: the task form opens over it.
type Page func(w http.ResponseWriter, r *http.Request, opts ...Option)

// Frame builds the shell for the person in a request: it knows the counts.
type Frame struct {
	Tasks *service.Tasks
	Loc   *time.Location // upcoming counts what has not gone by
	Now   func() time.Time
}

// tabOf is the tab each when puts a task in (decisions/0005).
var tabOf = map[model.When]string{model.Weekly: "week", model.Dated: "upcoming", model.Pending: "pending", model.Someday: "someday"}

// countedTabs: the tabs whose sidebar item shows how many tasks they hold
// (upcoming: those that have not gone by).
var countedTabs = map[model.When]string{model.Dated: "upcoming", model.Pending: "pending", model.Someday: "someday"}

// Shell also brings the undo toast after a delete: the tab opens with
// ?deleted=<id> while that task can still be undone.
func (f Frame) Shell(r *http.Request, tab string) (Option, error) {
	u := UserFrom(r.Context())
	tasks, err := f.Tasks.List(r.Context(), u.ID)
	if err != nil {
		return nil, err
	}
	now := f.Now().In(f.Loc)
	today, minute := clock.DateOf(now), clock.Time(now.Hour()*60+now.Minute())
	counts := map[string]int{}
	for _, t := range tasks {
		if t.Date != nil && t.Date.Over(today, minute) {
			continue
		}
		if name, ok := countedTabs[t.When]; ok {
			counts[name]++
		}
	}
	shell := WithShell(u, tab, counts)
	id, err := strconv.ParseInt(r.URL.Query().Get("deleted"), 10, 64)
	if err != nil {
		return shell, nil
	}
	d, err := f.Tasks.Deleted(r.Context(), u.ID, id)
	if errors.Is(err, model.ErrNotFound) {
		return shell, nil
	}
	if err != nil {
		return nil, err
	}
	undo := Toast{Message: "Deleted “" + d.Name + "”", Undo: "/" + tab + "/" + strconv.FormatInt(id, 10) + "/restore"}
	return func(v *View) { shell(v); v.Toast = &undo }, nil
}

// Toast is the one message component: Info, or Error. Undo, the url that
// restores a deleted task, shows the Undo button.
type Toast struct {
	Message string
	Error   bool
	Undo    string
}

// Shell is the app frame around a tab: who is in, and the four tabs.
type Shell struct {
	User     model.User
	Tabs     []Tab
	Settings bool // on settings: the profile is active, the phone has no emoji
}

type Tab struct {
	Name, Label, Color string
	Count              string // "" hides it
	Active             bool
}

func (t Tab) Href() string { return "/" + t.Name }

// tabs, in nav order, each with its swatch color.
var tabs = []Tab{
	{Name: "week", Label: "Week", Color: "sky"},
	{Name: "upcoming", Label: "Upcoming", Color: "coral"},
	{Name: "pending", Label: "Pending", Color: "mint"},
	{Name: "someday", Label: "Someday", Color: "lavender"},
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}

// redirect is http.Redirect, except for htmx: a boosted request would swap
// the target into the page; HX-Redirect makes htmx load it as a whole page.
func redirect(w http.ResponseWriter, r *http.Request, url string) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		return
	}
	http.Redirect(w, r, url, http.StatusSeeOther)
}

// group is one area's tasks, under its header.
type group struct {
	Name, Color string
	Tasks       []model.Task
}

// groupByArea returns a person's tasks in one place in time, one group per
// area that has any, in palette order. tasks keep the list's order.
func groupByArea(r *http.Request, tasks *service.Tasks, areas *service.Areas, userID int64, when model.When) ([]group, error) {
	all, err := tasks.List(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	as, err := areas.List(r.Context(), userID)
	if err != nil {
		return nil, err
	}
	var groups []group
	for _, a := range as {
		g := group{Name: a.Name, Color: a.Color}
		for _, t := range all {
			if t.When == when && t.Color == a.Color {
				g.Tasks = append(g.Tasks, t)
			}
		}
		if len(g.Tasks) > 0 {
			groups = append(groups, g)
		}
	}
	return groups, nil
}
