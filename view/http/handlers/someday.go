package handlers

import (
	"net/http"

	"sike/domain/model"
	"sike/domain/service"
)

// Someday is the someday tab: wishes, as tiles grouped by area.
type Someday struct {
	R     Renderer
	Frame Frame
	Tasks *service.Tasks
	Areas *service.Areas
}

func (h Someday) Show(w http.ResponseWriter, r *http.Request) { h.Page(w, r) }

func (h Someday) Page(w http.ResponseWriter, r *http.Request, opts ...Option) {
	u := UserFrom(r.Context())
	groups, err := groupByArea(r, h.Tasks, h.Areas, u.ID, model.Someday)
	if err != nil {
		serverError(w, err)
		return
	}
	shell, err := h.Frame.Shell(r, "someday")
	if err != nil {
		serverError(w, err)
		return
	}
	h.R.Render(w, "someday", map[string]any{"Groups": groups}, append([]Option{shell}, opts...)...)
}
