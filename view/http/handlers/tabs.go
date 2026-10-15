package handlers

import "net/http"

// Tabs serves the four tabs and settings. until each gets its own page, they
// share one: the shell and the header.
type Tabs struct {
	R Renderer
}

var titles = map[string]string{
	"week": "Week", "upcoming": "Upcoming", "pending": "Pending", "someday": "Someday", "settings": "Settings",
}

// Tab renders the tab named by the path; settings is no tab.
func (h Tabs) Tab(name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		active := name
		if name == "settings" {
			active = ""
		}
		h.R.Render(w, "tab", map[string]string{"Title": titles[name]}, WithShell(UserFrom(r.Context()), active))
	}
}
