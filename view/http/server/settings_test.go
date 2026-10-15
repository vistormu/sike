package server

import (
	"context"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"sike/domain/model"
)

// postHX is a change in settings: htmx posts the whole form.
func postHX(h http.Handler, form url.Values) *http.Response {
	req := httptest.NewRequest(http.MethodPost, "/settings", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: userCookieName, Value: "1"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

// settingsForm is the form as settings draws it, with changes.
func settingsForm(changes map[string]string) url.Values {
	f := url.Values{"name": {"You"}, "emoji": {"🦊"}, "warn": {"1440"}}
	for _, a := range model.DefaultAreas {
		f.Set("area-"+a.Color, a.Name)
	}
	for k, v := range changes {
		f.Set(k, v)
	}
	return f
}

func TestSettings(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	svc.Users.Add(context.Background(), "Ana", "🐱")
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	page := body(t, get(h, "/settings", in))
	for _, want := range []string{
		`name="name" value="You"`, `name="emoji" value="🦊" class="sr-only" checked`,
		`name="warn" value="1440" class="sr-only" checked`, `name="area-sky" value="Work"`,
		`hx-post="/settings"`, `action="/settings/switch"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("settings lacks %q", want)
		}
	}

	// a change: the person and a renamed area
	res := postHX(h, settingsForm(map[string]string{"name": " Vic ", "emoji": "🐸", "warn": "2880", "area-sky": "Job"}))
	page = body(t, res)
	if res.StatusCode != 200 || !strings.Contains(page, `<span class="truncate text-body">Vic</span>`) || strings.Contains(page, `role="alert"`) {
		t.Errorf("a change: status %d; want 200, the profile renamed, no toast", res.StatusCode)
	}
	if u, _ := svc.Users.Get(context.Background(), you.ID); u.Name != "Vic" || u.Emoji != "🐸" || u.Warn != 2880 {
		t.Errorf("stored %+v, want Vic 🐸 2880", u)
	}
	if areas, _ := svc.Areas.List(context.Background(), you.ID); areas[1].Name != "Job" || areas[0].Name != "Self" {
		t.Errorf("areas %+v, want sky Job, the rest as they were", areas)
	}

	// refused: an error toast, nothing stored
	for _, tt := range []struct{ field, value, toast string }{
		{"name", "ana", "“ana” is already taken"},
		{"name", "", "A name is required"},
		{"area-mint", "Self", "Another area is already called “Self”"},
		{"warn", "30", "Warn 1 hour, 1 day or 2 days before deadlines"},
	} {
		changes := map[string]string{"name": "Vic", "emoji": "🐸", "warn": "2880", "area-sky": "Job", tt.field: tt.value}
		res := postHX(h, settingsForm(changes))
		page := html.UnescapeString(body(t, res))
		if res.StatusCode != 200 || !strings.Contains(page, `role="alert"`) || !strings.Contains(page, tt.toast) {
			t.Errorf("%s=%q: status %d, want 200 and the toast %q", tt.field, tt.value, res.StatusCode, tt.toast)
		}
	}
	if u, _ := svc.Users.Get(context.Background(), you.ID); u.Name != "Vic" {
		t.Errorf("a refused change stored the name %q", u.Name)
	}

	// enter posts it plainly: back to settings
	res = postAs(h, "/settings", settingsForm(map[string]string{"name": "Vic"}), in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/settings" {
		t.Errorf("a plain post: %d to %q, want 303 to /settings", res.StatusCode, res.Header.Get("Location"))
	}
}
