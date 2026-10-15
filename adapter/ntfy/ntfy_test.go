package ntfy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"sike/domain/port"
)

func TestNotify(t *testing.T) {
	var got map[string]string
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		json.NewDecoder(r.Body).Decode(&got)
		if r.URL.Path != "/" || r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("posted %s as %s, want json to /", r.URL.Path, r.Header.Get("Content-Type"))
		}
		if got["topic"] == "nope" {
			http.Error(w, `{"code":40301,"error":"forbidden"}`, http.StatusForbidden)
		}
	}))
	defer srv.Close()

	c := New(srv.URL+"/", "tk_abc")
	if err := c.Notify(context.Background(), "sike-you", port.Message{Title: "▶ Gym", Body: "until 19:30"}); err != nil {
		t.Fatal(err)
	}
	if got["topic"] != "sike-you" || got["title"] != "▶ Gym" || got["message"] != "until 19:30" || auth != "Bearer tk_abc" {
		t.Errorf("posted %v with %q", got, auth)
	}
	err := c.Notify(context.Background(), "nope", port.Message{Body: "x"})
	if err == nil || err.Error() != "ntfy: 403 forbidden" {
		t.Errorf("a refused publish: err = %v, want the 403 and ntfy's reason", err)
	}
}
