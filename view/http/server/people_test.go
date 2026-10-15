package server

import (
	"context"
	"html"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"sike/domain/model"
)

func get(h http.Handler, path string, cookies ...*http.Cookie) *http.Response {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func body(t *testing.T, res *http.Response) string {
	t.Helper()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWhoListsPeople(t *testing.T) {
	h, users := testServer(t)
	ctx := context.Background()

	empty := body(t, get(h, "/who"))
	if !strings.Contains(empty, "Who are you?") || !strings.Contains(empty, "Add person") {
		t.Errorf("empty picker lacks its title or the add button")
	}
	if strings.Contains(empty, `action="/who/`) {
		t.Errorf("empty picker shows a tile")
	}

	if _, err := users.Add(ctx, "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	if _, err := users.Add(ctx, "Partner", "🐼"); err != nil {
		t.Fatal(err)
	}
	page := body(t, get(h, "/who"))
	for _, want := range []string{"🦊  You", "🐼  Partner", `action="/who/1"`, `action="/who/2"`, "--task-apricot", "--task-sky"} {
		if !strings.Contains(page, want) {
			t.Errorf("picker lacks %q", want)
		}
	}
}

func TestPickSetsTheCookie(t *testing.T) {
	h, users := testServer(t)
	u, err := users.Add(context.Background(), "You", "🦊")
	if err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/who/1", nil))
	res := rec.Result()
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/" {
		t.Fatalf("pick: %d to %q, want 303 to /", res.StatusCode, res.Header.Get("Location"))
	}
	cookies := res.Cookies()
	if len(cookies) != 1 || cookies[0].Name != userCookieName || cookies[0].Value != "1" || !cookies[0].HttpOnly {
		t.Fatalf("pick set %+v, want one http-only %s=1", cookies, userCookieName)
	}

	home := get(h, "/week", cookies[0])
	if home.StatusCode != 200 {
		t.Fatalf("home with the cookie: %d, want 200", home.StatusCode)
	}
	if b := body(t, home); !strings.Contains(b, u.Name) {
		t.Errorf("home does not show who is in")
	}

	for _, bad := range []string{"/who/9", "/who/x"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, bad, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("POST %s: %d, want 404", bad, rec.Code)
		}
	}
}

func TestRequireSendsToThePicker(t *testing.T) {
	h, users := testServer(t)
	u, err := users.Add(context.Background(), "You", "🦊")
	if err != nil {
		t.Fatal(err)
	}
	if err := users.Remove(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		cookie  *http.Cookie
		forgets bool // the bad cookie is cleared
	}{
		{"no cookie", nil, false},
		{"not a number", &http.Cookie{Name: userCookieName, Value: "x"}, true},
		{"deleted person", &http.Cookie{Name: userCookieName, Value: "1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res *http.Response
			if tt.cookie == nil {
				res = get(h, "/week")
			} else {
				res = get(h, "/week", tt.cookie)
			}
			if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/who" {
				t.Fatalf("%d to %q, want 303 to /who", res.StatusCode, res.Header.Get("Location"))
			}
			cleared := len(res.Cookies()) == 1 && res.Cookies()[0].MaxAge < 0
			if cleared != tt.forgets {
				t.Errorf("cookie cleared = %v, want %v", cleared, tt.forgets)
			}
		})
	}
}

func post(h http.Handler, path string, form url.Values) *http.Response {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func TestNewPersonForm(t *testing.T) {
	h, users := testServer(t)
	if _, err := users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	res := get(h, "/who/new")
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	page := body(t, res)
	if n := strings.Count(page, `type="radio" name="emoji"`); n != len(model.Emojis) {
		t.Errorf("%d emoji radios, want %d", n, len(model.Emojis))
	}
	// 🦊 is taken: the first free one is preselected
	if !strings.Contains(page, `value="🐼" class="sr-only" checked`) || strings.Count(page, " checked") != 1 {
		t.Errorf("want exactly 🐼 checked")
	}
	if strings.Contains(page, `role="alert"`) {
		t.Errorf("a fresh form shows an error toast")
	}
}

func TestCreatePerson(t *testing.T) {
	h, users := testServer(t)

	res := post(h, "/who/new", url.Values{"name": {"  You "}, "emoji": {"🦊"}})
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/" {
		t.Fatalf("create: %d to %q, want 303 to /", res.StatusCode, res.Header.Get("Location"))
	}
	if c := res.Cookies(); len(c) != 1 || c[0].Name != userCookieName || c[0].Value != "1" {
		t.Errorf("create set %+v, want %s=1: in as the new person", c, userCookieName)
	}
	all, _ := users.List(context.Background())
	if len(all) != 1 || all[0].Name != "You" || all[0].Emoji != "🦊" {
		t.Errorf("users = %+v", all)
	}
}

func TestCreatePersonErrors(t *testing.T) {
	h, users := testServer(t)
	if _, err := users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, emoji, toast string
	}{
		{"you", "🐼", "“you” is already taken"},
		{"   ", "🐼", "A name is required"},
		{"Partner", "🐶", "Pick one of"},
	}
	for _, tt := range tests {
		t.Run(tt.toast, func(t *testing.T) {
			res := post(h, "/who/new", url.Values{"name": {tt.name}, "emoji": {tt.emoji}})
			if res.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status %d, want 422", res.StatusCode)
			}
			if len(res.Cookies()) != 0 {
				t.Errorf("a failed create set a cookie")
			}
			page := body(t, res)
			if !strings.Contains(page, `role="alert"`) || !strings.Contains(page, html.EscapeString(tt.toast)) {
				t.Errorf("no error toast with %q", tt.toast)
			}
			if !strings.Contains(page, `value="`+html.EscapeString(tt.name)+`"`) {
				t.Errorf("the typed name %q is lost", tt.name)
			}
		})
	}
	if all, _ := users.List(context.Background()); len(all) != 1 {
		t.Errorf("%d users after failed creates, want 1", len(all))
	}
}

// settings' switch person forgets who you are.
func TestSwitchPerson(t *testing.T) {
	h, users := testServer(t)
	if _, err := users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	res := postAs(h, "/settings/switch", nil, &http.Cookie{Name: userCookieName, Value: "1"})
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/who" {
		t.Fatalf("%d to %q, want 303 to /who", res.StatusCode, res.Header.Get("Location"))
	}
	if c := res.Cookies(); len(c) != 1 || c[0].Name != userCookieName || c[0].MaxAge >= 0 {
		t.Errorf("cookies = %+v, want %s cleared", c, userCookieName)
	}
}
