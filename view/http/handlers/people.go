package handlers

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"strconv"

	"sike/domain/model"
	"sike/domain/service"
)

// People is "who are you?": the picker, the cookie that remembers the choice,
// and the middleware that sends you to the picker without one. no passwords:
// tailscale is the gate (decisions/0002).
type People struct {
	R     Renderer
	Users *service.Users
}

const userCookie = "sike_user"

// tileColors give each person's tile a color by their order, so a person
// needs no color of their own.
var tileColors = []string{"apricot", "sky", "mint", "lavender", "coral", "lime", "pink"}

type whoTile struct {
	ID    int64
	Label string
	Color string
}

// Who renders the picker.
func (h People) Who(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	tiles := make([]whoTile, len(users))
	for i, u := range users {
		tiles[i] = whoTile{ID: u.ID, Label: u.Emoji + "  " + u.Name, Color: tileColors[i%len(tileColors)]}
	}
	h.R.Render(w, "who", map[string]any{"Users": tiles})
}

type newPerson struct {
	Name   string
	Emoji  string
	Emojis []string
}

// New renders the add-person form, with the first emoji nobody uses yet.
func (h People) New(w http.ResponseWriter, r *http.Request) {
	users, err := h.Users.List(r.Context())
	if err != nil {
		serverError(w, err)
		return
	}
	emoji := model.Emojis[0]
	for _, e := range model.Emojis {
		if !slices.ContainsFunc(users, func(u model.User) bool { return u.Emoji == e }) {
			emoji = e
			break
		}
	}
	h.R.Render(w, "who_new", newPerson{Emoji: emoji, Emojis: model.Emojis})
}

// Create adds the person and goes home as them. a bad form comes back as it
// was typed, with the reason in an error toast.
func (h People) Create(w http.ResponseWriter, r *http.Request) {
	name, emoji := r.FormValue("name"), r.FormValue("emoji")
	u, err := h.Users.Add(r.Context(), name, emoji)
	if errors.Is(err, model.ErrInvalid) {
		h.R.RenderToast(w, "who_new", newPerson{Name: name, Emoji: emoji, Emojis: model.Emojis},
			http.StatusUnprocessableEntity, Toast{Message: err.Error(), Error: true})
		return
	}
	if err != nil {
		serverError(w, err)
		return
	}
	setUser(w, u.ID)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Pick remembers the chosen person and goes home.
func (h People) Pick(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if _, err := h.Users.Get(r.Context(), id); errors.Is(err, model.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		serverError(w, err)
		return
	}
	setUser(w, id)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func setUser(w http.ResponseWriter, id int64) {
	http.SetCookie(w, &http.Cookie{
		Name:     userCookie,
		Value:    strconv.FormatInt(id, 10),
		Path:     "/",
		MaxAge:   400 * 24 * 60 * 60, // the longest browsers keep a cookie
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Require lets a request through only with a known person in the cookie, and
// puts that person in the context (UserFrom). otherwise: the picker.
func (h People) Require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(userCookie)
		if err != nil {
			http.Redirect(w, r, "/who", http.StatusSeeOther)
			return
		}
		id, err := strconv.ParseInt(c.Value, 10, 64)
		if err != nil {
			forget(w, r)
			return
		}
		u, err := h.Users.Get(r.Context(), id)
		if errors.Is(err, model.ErrNotFound) {
			forget(w, r) // a person since deleted
			return
		}
		if err != nil {
			serverError(w, err)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

// forget drops a cookie that names nobody and goes to the picker.
func forget(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: userCookie, Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/who", http.StatusSeeOther)
}

type userKey struct{}

// UserFrom is the person Require let through.
func UserFrom(ctx context.Context) model.User {
	u, _ := ctx.Value(userKey{}).(model.User)
	return u
}
