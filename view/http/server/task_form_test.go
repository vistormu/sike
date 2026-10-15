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

func postAs(h http.Handler, path string, form url.Values, cookie *http.Cookie) *http.Response {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Result()
}

func TestTaskFormOpensOverItsTab(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	if _, err := svc.Tasks.Add(ctx, you.ID, "Fix bike", "mint", model.Pending); err != nil {
		t.Fatal(err)
	}
	if err := svc.Areas.Rename(ctx, you.ID, "mint", "House"); err != nil {
		t.Fatal(err)
	}
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	for _, tab := range []string{"pending", "someday"} {
		t.Run(tab, func(t *testing.T) {
			res := get(h, "/"+tab+"/new", in)
			if res.StatusCode != 200 {
				t.Fatalf("status %d", res.StatusCode)
			}
			page := body(t, res)
			for _, want := range []string{
				`action="/` + tab + `/new"`,
				`href="/` + tab + `"`,                         // ×, ‹ and the scrim close to the tab
				`value="` + tab + `" class="sr-only" checked`, // When preset from the tab
				`value="mint" class="sr-only">`,               // the person's areas, none picked
				"House",
				"autofocus",
			} {
				if !strings.Contains(page, want) {
					t.Errorf("form lacks %q", want)
				}
			}
			if n := strings.Count(page, `name="area"`); n != 7 {
				t.Errorf("%d area options, want 7", n)
			}
			// m11 offers the whens the service takes: no every week, no date yet
			if strings.Contains(page, "Every week") || strings.Contains(page, "On a date") {
				t.Error("the form offers a when the service refuses")
			}
		})
	}
	if page := body(t, get(h, "/pending/new", in)); !strings.Contains(page, "Fix bike") {
		t.Error("the tab is not under the form")
	}
	if page := body(t, get(h, "/pending", in)); strings.Contains(page, `action="/pending/new"`) {
		t.Error("the tab opens with the form")
	}
}

func TestCreateTask(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	// opened from pending, moved to someday: it lands where its when says
	res := postAs(h, "/pending/new", url.Values{"name": {" Learn piano "}, "area": {"pink"}, "when": {"someday"}}, in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/someday" {
		t.Fatalf("create: %d to %q, want 303 to /someday", res.StatusCode, res.Header.Get("Location"))
	}
	tasks, _ := svc.Tasks.List(ctx, you.ID)
	if len(tasks) != 1 || tasks[0].Name != "Learn piano" || tasks[0].Color != "pink" || tasks[0].When != model.Someday {
		t.Errorf("tasks = %+v", tasks)
	}
}

func TestCreateTaskErrors(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	tests := []struct {
		name, area, when, toast string
	}{
		{"   ", "sky", "pending", "A task needs a name"},
		{"Fix bike", "", "someday", "Pick an area"},
		{"Fix bike", "sky", "weekly", "When is pending or someday"},
		{strings.Repeat("a", 81), "sky", "pending", "longer than 80"},
	}
	for _, tt := range tests {
		t.Run(tt.toast, func(t *testing.T) {
			res := postAs(h, "/someday/new", url.Values{"name": {tt.name}, "area": {tt.area}, "when": {tt.when}}, in)
			if res.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status %d, want 422", res.StatusCode)
			}
			page := body(t, res)
			if !strings.Contains(page, `role="alert"`) || !strings.Contains(page, html.EscapeString(tt.toast)) {
				t.Errorf("no error toast with %q", tt.toast)
			}
			// the form keeps what was typed, still over the tab it came from
			if !strings.Contains(page, `name="name" value="`+html.EscapeString(tt.name)+`"`) {
				t.Errorf("the typed name is lost")
			}
			if tt.area != "" && !strings.Contains(page, `value="`+tt.area+`" class="sr-only" checked`) {
				t.Errorf("the picked area is lost")
			}
			if !strings.Contains(page, `action="/someday/new"`) || !strings.Contains(page, "No wishes yet") {
				t.Errorf("the form is not over someday")
			}
		})
	}
	if tasks, _ := svc.Tasks.List(ctx, you.ID); len(tasks) != 0 {
		t.Errorf("%d tasks after failed creates, want 0", len(tasks))
	}
}

func TestTaskFormNeedsAPerson(t *testing.T) {
	h, _ := testServer(t)
	if res := get(h, "/pending/new"); res.Header.Get("Location") != "/who" {
		t.Errorf("GET without a person: %d to %q", res.StatusCode, res.Header.Get("Location"))
	}
	if res := post(h, "/pending/new", url.Values{"name": {"x"}}); res.Header.Get("Location") != "/who" {
		t.Errorf("POST without a person: %d to %q", res.StatusCode, res.Header.Get("Location"))
	}
}
