package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestRootGoesToWeek(t *testing.T) {
	h, _ := testServer(t)
	res := get(h, "/")
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/week" {
		t.Errorf("/: %d to %q, want 303 to /week", res.StatusCode, res.Header.Get("Location"))
	}
}

var navRe = regexp.MustCompile(`(?s)<nav .*?</nav>`)

func TestTabs(t *testing.T) {
	h, users := testServer(t)
	if _, err := users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	tests := []struct {
		path, title, active string // active: the nav item marked, "" for none
	}{
		{"/week", "Week", "/week"},
		{"/upcoming", "Upcoming", "/upcoming"},
		{"/pending", "Pending", "/pending"},
		{"/someday", "Someday", "/someday"},
		{"/settings", "Settings", "/settings"}, // the profile, in the sidebar only
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			res := get(h, tt.path, in)
			if res.StatusCode != 200 {
				t.Fatalf("status %d", res.StatusCode)
			}
			page := body(t, res)
			// week's is the desktop's: the phone's names the day
			if !regexp.MustCompile(`<h1 class="[^"]*text-display[^"]*">` + tt.title + `</h1>`).MatchString(page) {
				t.Errorf("no %q title", tt.title)
			}
			// the shell: four tabs in the sidebar and four in the tab bar, the
			// person, the way to settings. counted in the two navs only: a
			// page links too (week's Today)
			navs := strings.Join(navRe.FindAllString(page, -1), "")
			for _, href := range []string{"/week", "/upcoming", "/pending", "/someday"} {
				if n := strings.Count(navs, `href="`+href+`"`); n != 2 {
					t.Errorf("%s linked %d times, want 2 (sidebar + tab bar)", href, n)
				}
			}
			// the phone's emoji links to settings too, but not on settings
			links := map[bool]int{true: 1, false: 2}[tt.path == "/settings"]
			if !strings.Contains(page, "🦊") || !strings.Contains(page, "You") || strings.Count(page, `href="/settings"`) != links {
				t.Errorf("the profile (emoji, name, %d links to settings) is missing", links)
			}
			// active = the bordered butter fill, once in each nav; settings'
			// profile only in the sidebar
			want := map[bool]int{true: 1, false: 2}[tt.path == "/settings"]
			if n := strings.Count(navs, "border-border bg-accent"); n != want {
				t.Errorf("%d active items, want %d", n, want)
			}
		})
	}
}

func TestTabsNeedAPerson(t *testing.T) {
	h, _ := testServer(t)
	res := get(h, "/pending")
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/who" {
		t.Errorf("no cookie: %d to %q, want 303 to /who", res.StatusCode, res.Header.Get("Location"))
	}
	// htmx: a boosted request can't follow a 303 into a whole page
	req := httptest.NewRequest(http.MethodGet, "/pending", nil)
	req.Header.Set("HX-Request", "true")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("HX-Redirect") != "/who" {
		t.Errorf("htmx without a cookie: HX-Redirect = %q, want /who", rec.Header().Get("HX-Redirect"))
	}
}
