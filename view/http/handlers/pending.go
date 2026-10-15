package handlers

import (
	"net/http"

	"sike/domain/model"
	"sike/domain/service"
)

// Pending is the pending tab: owed things, grouped by area.
type Pending struct {
	R     Renderer
	Frame Frame
	Tasks *service.Tasks
	Areas *service.Areas
}

func (h Pending) Show(w http.ResponseWriter, r *http.Request) { h.Page(w, r) }

func (h Pending) Page(w http.ResponseWriter, r *http.Request, opts ...Option) {
	u := UserFrom(r.Context())
	groups, err := groupByArea(r, h.Tasks, h.Areas, u.ID, model.Pending)
	if err != nil {
		serverError(w, err)
		return
	}
	shell, err := h.Frame.Shell(r, "pending")
	if err != nil {
		serverError(w, err)
		return
	}
	h.R.Render(w, "pending", map[string]any{"Groups": groups}, append([]Option{shell}, opts...)...)
}
