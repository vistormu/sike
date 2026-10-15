// Package handlers holds the http handlers, one file per screen.
package handlers

import (
	"log"
	"net/http"
	"strconv"

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
}

type Option func(*View)

// WithToast shows the floating toast (decisions/0007) with a status: a form
// that failed comes back as it was typed.
func WithToast(status int, t Toast) Option {
	return func(v *View) { v.Status, v.Toast = status, &t }
}

// WithShell wraps the page in the sidebar (desktop) or tab bar (phone), with
// tab active. tab is "" on pages that are no tab, like settings. counts are
// shown next to their tab in the sidebar; a missing or zero count hides.
func WithShell(u model.User, tab string, counts map[string]int) Option {
	return func(v *View) {
		s := &Shell{User: u}
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

// Frame builds the shell for the person in a request: it knows the counts.
type Frame struct {
	Tasks *service.Tasks
}

// countedTabs: the tabs whose sidebar item shows how many tasks they hold.
var countedTabs = map[model.When]string{model.Pending: "pending", model.Someday: "someday"}

func (f Frame) Shell(r *http.Request, tab string) (Option, error) {
	u := UserFrom(r.Context())
	tasks, err := f.Tasks.List(r.Context(), u.ID)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, t := range tasks {
		if name, ok := countedTabs[t.When]; ok {
			counts[name]++
		}
	}
	return WithShell(u, tab, counts), nil
}

// Toast is the one message component: Info, or Error. Action shows Undo.
type Toast struct {
	Message string
	Error   bool
	Action  bool
}

// Shell is the app frame around a tab: who is in, and the four tabs.
type Shell struct {
	User model.User
	Tabs []Tab
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
