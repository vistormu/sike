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

// postHX is what settings' htmx posts: the whole form.
func postHX(h http.Handler, path string, form url.Values) *http.Response {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("HX-Request", "true")
	req.AddCookie(&http.Cookie{Name: userCookieName, Value: "1"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

// settingsForm is the form as settings draws it, with changes.
func settingsForm(changes map[string]string) url.Values {
	f := url.Values{"name": {"You"}, "emoji": {"🦊"}, "warn": {"1440"}, "topic": {""}}
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
	res := postHX(h, "/settings", settingsForm(map[string]string{"name": " Vic ", "emoji": "🐸", "warn": "2880", "area-sky": "Job", "topic": "sike-vic"}))
	page = body(t, res)
	if res.StatusCode != 200 || !strings.Contains(page, `<span class="truncate text-body">Vic</span>`) || strings.Contains(page, `role="alert"`) {
		t.Errorf("a change: status %d; want 200, the profile renamed, no toast", res.StatusCode)
	}
	if u, _ := svc.Users.Get(context.Background(), you.ID); u.Name != "Vic" || u.Emoji != "🐸" || u.Warn != 2880 || u.Topic != "sike-vic" {
		t.Errorf("stored %+v, want Vic 🐸 2880 sike-vic", u)
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
		{"topic", "sike vic", "A topic is up to 64 letters, digits, - and _"},
	} {
		changes := map[string]string{"name": "Vic", "emoji": "🐸", "warn": "2880", "area-sky": "Job", tt.field: tt.value}
		res := postHX(h, "/settings", settingsForm(changes))
		page := html.UnescapeString(body(t, res))
		if res.StatusCode != 200 || !strings.Contains(page, `role="alert"`) || !strings.Contains(page, tt.toast) {
			t.Errorf("%s=%q: status %d, want 200 and the toast %q", tt.field, tt.value, res.StatusCode, tt.toast)
		}
	}
	if u, _ := svc.Users.Get(context.Background(), you.ID); u.Name != "Vic" {
		t.Errorf("a refused change stored the name %q", u.Name)
	}

	// areas renamed top to bottom: the first briefly takes the second's name,
	// refused; renaming the second then saves both
	vic := map[string]string{"name": "Vic", "emoji": "🐸", "warn": "2880", "area-sky": "Job", "area-lavender": "Job"}
	if page := body(t, postHX(h, "/settings", settingsForm(vic))); !strings.Contains(page, `role="alert"`) {
		t.Error("two areas called Job: no error toast")
	}
	vic["area-sky"] = "Work"
	if page := body(t, postHX(h, "/settings", settingsForm(vic))); strings.Contains(page, `role="alert"`) {
		t.Error("the second area renamed too: still an error toast")
	}
	if areas, _ := svc.Areas.List(context.Background(), you.ID); areas[0].Name != "Job" || areas[1].Name != "Work" {
		t.Errorf("areas %q, %q; want Job, Work", areas[0].Name, areas[1].Name)
	}

	// enter posts it plainly: back to settings
	res = postAs(h, "/settings", settingsForm(map[string]string{"name": "Vic"}), in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/settings" {
		t.Errorf("a plain post: %d to %q, want 303 to /settings", res.StatusCode, res.Header.Get("Location"))
	}
}

// Send test: to the topic as typed, saved or not; an info toast, or why not.
func TestSettingsSendTest(t *testing.T) {
	h, svc := testApp(t)
	svc.Users.Add(context.Background(), "You", "🦊")

	page := html.UnescapeString(body(t, postHX(h, "/settings/test", settingsForm(map[string]string{"topic": " sike-you "}))))
	if !strings.Contains(page, `role="status"`) || !strings.Contains(page, "Sent a test to sike-you") {
		t.Error("a sent test does not say so")
	}
	if len(*svc.Sent) != 1 || (*svc.Sent)[0] != "sike-you: 🦊 sike · Notifications reach you, You." {
		t.Errorf("sent %q", *svc.Sent)
	}
	for _, tt := range []struct{ topic, toast string }{
		{"", "Type an ntfy topic first"},
		{"refused", "Not sent: ntfy: 403 forbidden"},
	} {
		page := html.UnescapeString(body(t, postHX(h, "/settings/test", settingsForm(map[string]string{"topic": tt.topic}))))
		if !strings.Contains(page, `role="alert"`) || !strings.Contains(page, tt.toast) {
			t.Errorf("topic %q: want the error toast %q", tt.topic, tt.toast)
		}
	}
	if page := body(t, get(h, "/settings", &http.Cookie{Name: userCookieName, Value: "1"})); !strings.Contains(page, `hx-post="/settings/test"`) || !strings.Contains(page, `name="topic" value=""`) {
		t.Error("settings lacks the topic and Send test")
	}
}
