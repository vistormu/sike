package handlers

import "net/http"

// Tabs serves settings until it has its own page (m23): the shell and the
// header.
type Tabs struct {
	R     Renderer
	Frame Frame
}

var titles = map[string]string{"settings": "Settings"}

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
