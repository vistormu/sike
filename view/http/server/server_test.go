package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	s, err := New(Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	h := s.http.Handler

	tests := []struct {
		path   string
		status int
		ctype  string
		body   string // a substring that must be there
	}{
		{"/", 200, "text/html", "<!doctype html>"},
		{"/", 200, "text/html", `href="/static/css/site.css"`},
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
