package handlers

import "net/http"

// Tabs serves the tabs that have no page of their own yet, and settings: the
// shell and the header.
type Tabs struct {
	R     Renderer
	Frame Frame
}

var titles = map[string]string{
	"week": "Week", "upcoming": "Upcoming", "settings": "Settings",
}

// Tab renders the tab named by the path; settings is no tab.
func (h Tabs) Tab(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		active := name
		if name == "settings" {
			active = ""
		}
		shell, err := h.Frame.Shell(r, active)
		if err != nil {
			serverError(w, err)
			return
		}
		h.R.Render(w, "tab", map[string]string{"Title": titles[name]}, shell)
	}
}
