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

func (h Pending) Show(w http.ResponseWriter, r *http.Request) {
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
	h.R.Render(w, "pending", map[string]any{"Groups": groups}, shell)
}
