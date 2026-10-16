package server

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"sike/adapter/store"
	"sike/domain/port"
	"sike/domain/service"
)

// services are the ones testServer's handler runs on, to set up data.
type services struct {
	Users *service.Users
	Areas *service.Areas
	Tasks *service.Tasks
	Moods *service.Moods
	Sent  *[]string // what the notifier sent: "topic: title · body"
}

// fakeNtfy stands in for ntfy: it keeps what it is sent, and refuses the
// topic "refused" as a server without access would.
type fakeNtfy struct{ sent *[]string }

func (f fakeNtfy) Notify(_ context.Context, topic string, m port.Message) error {
	if topic == "refused" {
		return errors.New("ntfy: 403 forbidden")
	}
	*f.sent = append(*f.sent, topic+": "+m.Title+" · "+m.Body)
	return nil
}

// the tests' clock: madrid, stopped on wednesday 2026-10-14 at 18:45, the day
// and minute figma's week draws.
var (
	testLoc, _ = time.LoadLocation("Europe/Madrid")
	testNow    = func() time.Time { return time.Date(2026, 10, 14, 18, 45, 0, 0, testLoc) }
)

// testServer is the real server on a fresh in-memory database.
func testServer(t *testing.T) (http.Handler, *service.Users) {
	t.Helper()
	h, svc := testApp(t)
	return h, svc.Users
}

// testApp is testServer with every service.
func testApp(t *testing.T) (http.Handler, services) {
	t.Helper()
	db, err := store.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	if _, err := store.Migrate(context.Background(), db); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	st := store.New(db)
	svc := services{Users: service.NewUsers(st), Areas: service.NewAreas(st), Tasks: service.NewTasks(st), Moods: service.NewMoods(st), Sent: &[]string{}}
	s, err := New(Config{Ping: db.PingContext, Users: svc.Users, Areas: svc.Areas, Tasks: svc.Tasks, Moods: svc.Moods, Notifier: service.NewNotifier(fakeNtfy{svc.Sent}, st, svc.Users, svc.Tasks, testLoc), Loc: testLoc, Now: testNow})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return s.http.Handler, svc
}

func TestRoutes(t *testing.T) {
	h, _ := testServer(t)

	tests := []struct {
		path   string
		status int
		ctype  string
		body   string // a substring that must be there
	}{
		{"/who", 200, "text/html", "<!doctype html>"},
		{"/who", 200, "text/html", `href="/static/css/site.css"`},
		{"/dev/components", 200, "text/html", "sike — components"},
		{"/static/js/htmx.min.js", 200, "text/javascript", "htmx"},
		{"/static/css/site.css", 200, "text/css", "--color-bg"},
		{"/healthz", 200, "text/plain", "ok"},
		{"/static/manifest.webmanifest", 200, "application/manifest+json", `"display": "standalone"`},
		{"/static/icons/apple-touch-icon.png", 200, "image/png", ""},
		{"/static/icons/icon-512.png", 200, "image/png", ""},
		{"/static/icons/logo.svg", 200, "image/svg+xml", "<svg"},
		{"/who", 200, "text/html", `<link rel="manifest" href="/static/manifest.webmanifest">`},
		{"/who", 200, "text/html", `<link rel="apple-touch-icon" href="/static/icons/apple-touch-icon.png">`},
		{"/static/fonts/space-grotesk-latin-700-normal.woff2", 200, "font/woff2", ""},
		{"/static/", 404, "", ""},
		{"/static/js/", 404, "", ""},
		{"/nope", 404, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			res := rec.Result()
			if res.StatusCode != tt.status {
				t.Fatalf("status = %d, want %d", res.StatusCode, tt.status)
			}
			if ct := res.Header.Get("Content-Type"); !strings.HasPrefix(ct, tt.ctype) {
				t.Errorf("content-type = %q, want %q…", ct, tt.ctype)
			}
			body, _ := io.ReadAll(res.Body)
			if !strings.Contains(string(body), tt.body) {
				t.Errorf("body lacks %q", tt.body)
			}
		})
	}
}

// every component renders with the props the catalog passes, and with none:
// a missing optional prop must not break a page.
func TestComponentsRenderWithoutProps(t *testing.T) {
	rd, err := newRenderer()
	if err != nil {
		t.Fatalf("newRenderer: %v", err)
	}
	set := rd.pages["dev_components"]
	for _, tmpl := range set.Templates() {
		name := tmpl.Name()
		if !strings.HasPrefix(name, "components/") {
			continue
		}
		t.Run(name, func(t *testing.T) {
			if err := set.ExecuteTemplate(io.Discard, name, map[string]any{}); err != nil {
				t.Errorf("render with no props: %v", err)
			}
		})
	}
}
