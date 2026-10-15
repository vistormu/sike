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

// group is one area's tasks, under its header.
type group struct {
	Name, Color string
	Tasks       []model.Task
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
