package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"sike/adapter/store"
	"sike/domain/service"
)

// services are the ones testServer's handler runs on, to set up data.
type services struct {
	Users *service.Users
	Areas *service.Areas
	Tasks *service.Tasks
}

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
	svc := services{Users: service.NewUsers(st), Areas: service.NewAreas(st), Tasks: service.NewTasks(st)}
	s, err := New(Config{Users: svc.Users, Areas: svc.Areas, Tasks: svc.Tasks})
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
