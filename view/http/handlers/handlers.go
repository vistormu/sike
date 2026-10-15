// Package handlers holds the http handlers, one file per screen.
package handlers

import (
	"log"
	"net/http"

	"sike/domain/model"
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
// tab active. tab is "" on pages that are no tab, like settings.
func WithShell(u model.User, tab string) Option {
	return func(v *View) {
		s := &Shell{User: u}
		for _, t := range tabs {
			t.Active = t.Name == tab
			s.Tabs = append(s.Tabs, t)
		}
		v.Shell = s
	}
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
