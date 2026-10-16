package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"sike/domain/model"
	"sike/domain/service"
)

// Settings is the person's own page: name, emoji, area names, the ntfy topic
// and its test, the deadline warning, and the way to switch person. there is no save: every change posts
// the form (htmx); the page comes back and only its toast and the profile in
// the sidebar are taken from it.
type Settings struct {
	R        Renderer
	Frame    Frame
	Users    *service.Users
	Areas    *service.Areas
	Notifier *service.Notifier
}

type settingsPage struct {
	Name, Emoji, Topic string
	Emojis             []string
	Warnings           []warning
	Areas              []model.Area
}

type warning struct {
	Value int
	Label string
	On    bool
}

func (h Settings) Show(w http.ResponseWriter, r *http.Request) { h.page(w, r) }

func (h Settings) page(w http.ResponseWriter, r *http.Request, opts ...Option) {
	u := UserFrom(r.Context())
	areas, err := h.Areas.List(r.Context(), u.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	page := settingsPage{Name: u.Name, Emoji: u.Emoji, Topic: u.Topic, Emojis: model.Emojis, Areas: areas}
	for _, v := range model.Warnings {
		page.Warnings = append(page.Warnings, warning{Value: v, Label: model.WarnLabel(v), On: v == u.Warn})
	}
	shell, err := h.Frame.Shell(r, "settings")
	if err != nil {
		serverError(w, err)
		return
	}
	h.R.Render(w, "settings", page, append([]Option{shell}, opts...)...)
}

// Save takes the whole form: the person, then the areas, renamed as one.
// what is refused comes back as an error toast, with a 200 so htmx takes it.
func (h Settings) Save(w http.ResponseWriter, r *http.Request) {
	u := UserFrom(r.Context())
	edited := u
	edited.Name, edited.Emoji = r.FormValue("name"), r.FormValue("emoji")
	edited.Warn, _ = strconv.Atoi(r.FormValue("warn"))
	edited.Topic = r.FormValue("topic")
	err := h.save(r, edited)
	if errors.Is(err, model.ErrInvalid) {
		h.page(w, r, WithToast(http.StatusOK, Toast{Message: err.Error(), Error: true}))
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	if r.Header.Get("HX-Request") == "" { // enter: a plain post
		redirect(w, r, "/settings")
		return
	}
	saved, err := h.Users.Get(r.Context(), u.ID)
	if err != nil {
		serverError(w, err)
		return
	}
	h.page(w, r.WithContext(withUser(r.Context(), saved)))
}

// Test is "Send test": a test message to the topic in the form, saved or not
// yet. sent, an info toast; else why not.
func (h Settings) Test(w http.ResponseWriter, r *http.Request) {
	topic := strings.TrimSpace(r.FormValue("topic"))
	err := h.Notifier.Test(r.Context(), UserFrom(r.Context()), topic)
	switch {
	case errors.Is(err, model.ErrInvalid):
		h.page(w, r, WithToast(http.StatusOK, Toast{Message: err.Error(), Error: true}))
	case err != nil:
		log.Printf("send test: %v", err)
		h.page(w, r, WithToast(http.StatusOK, Toast{Message: "Not sent: " + err.Error(), Error: true}))
	default:
		h.page(w, r, WithToast(http.StatusOK, Toast{Message: "Sent a test to " + topic}))
	}
}

func (h Settings) save(r *http.Request, u model.User) error {
	if _, err := h.Users.Edit(r.Context(), u); err != nil {
		return err
	}
	names := map[string]string{}
	for _, color := range model.Colors {
		if name, ok := r.Form["area-"+color]; ok {
			names[color] = name[0]
		}
	}
	return h.Areas.RenameAll(r.Context(), u.ID, names)
}
