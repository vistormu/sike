package server

import (
	"context"
	"html"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

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
	if _, err := svc.Tasks.Add(ctx, you.ID, "Fix bike", "mint", model.Pending, nil); err != nil {
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
			// the form offers the whens the service takes: no date yet (m18)
			if !strings.Contains(page, "Every week") || strings.Contains(page, "On a date") {
				t.Error("the form's whens are not the service's")
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
		{"Fix bike", "sky", "weekly", "Pick at least one day"},
		{"Fix bike", "sky", "date", "When is every week, pending or someday"},
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

func TestTasksOpenTheirForm(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	for _, x := range []struct {
		name, color string
		when        model.When
	}{{"Fix bike", "mint", model.Pending}, {"Learn piano", "pink", model.Someday}} {
		if _, err := svc.Tasks.Add(ctx, you.ID, x.name, x.color, x.when, nil); err != nil {
			t.Fatal(err)
		}
	}
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	// every task on a tab links to its form, selected first by app.js
	if page := body(t, get(h, "/pending", in)); !strings.Contains(page, `href="/pending/1" data-task`) {
		t.Error("a pending row does not link to its form")
	}
	if page := body(t, get(h, "/someday", in)); !strings.Contains(page, `href="/someday/2" data-task`) {
		t.Error("a tile does not link to its form")
	}

	page := body(t, get(h, "/pending/1", in))
	for _, want := range []string{
		`action="/pending/1"`,
		`name="name" value="Fix bike"`,
		`value="mint" class="sr-only" checked`,
		`value="pending" class="sr-only" checked`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the edit form lacks %q", want)
		}
	}
	for _, path := range []string{"/pending/9", "/pending/x"} {
		if res := get(h, path, in); res.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: %d, want 404", path, res.StatusCode)
		}
	}
	// someone else's task is not found
	if _, err := svc.Users.Add(ctx, "Partner", "🐼"); err != nil {
		t.Fatal(err)
	}
	if res := get(h, "/pending/1", &http.Cookie{Name: userCookieName, Value: "2"}); res.StatusCode != http.StatusNotFound {
		t.Errorf("another's task: %d, want 404", res.StatusCode)
	}
}

func TestUpdateTask(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	fix, _ := svc.Tasks.Add(ctx, you.ID, "Fix bike", "mint", model.Pending, nil)
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	// a new when moves it: the redirect follows it to its tab
	res := postAs(h, "/pending/1", url.Values{"name": {"Fix the bike"}, "area": {"pink"}, "when": {"someday"}}, in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/someday" {
		t.Fatalf("update: %d to %q, want 303 to /someday", res.StatusCode, res.Header.Get("Location"))
	}
	got, _ := svc.Tasks.Get(ctx, you.ID, fix.ID)
	if got.Name != "Fix the bike" || got.Color != "pink" || got.When != model.Someday {
		t.Errorf("task = %+v", got)
	}

	// a bad edit: 422, the toast, the form as typed, the task unchanged
	res = postAs(h, "/someday/1", url.Values{"name": {"  "}, "area": {"sky"}, "when": {"pending"}}, in)
	if res.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("bad update: status %d, want 422", res.StatusCode)
	}
	page := body(t, res)
	for _, want := range []string{`role="alert"`, "A task needs a name", `action="/someday/1"`, `value="sky" class="sr-only" checked`, `value="pending" class="sr-only" checked`} {
		if !strings.Contains(page, want) {
			t.Errorf("the failed edit lacks %q", want)
		}
	}
	if again, _ := svc.Tasks.Get(ctx, you.ID, fix.ID); again != got {
		t.Errorf("a failed edit changed the task: %+v", again)
	}

	if res := postAs(h, "/pending/9", url.Values{"name": {"x"}, "area": {"sky"}, "when": {"pending"}}, in); res.StatusCode != http.StatusNotFound {
		t.Errorf("update a missing task: %d, want 404", res.StatusCode)
	}
}

func TestDeleteAndUndo(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	for _, name := range []string{"Fix bike", "Call the bank"} {
		if _, err := svc.Tasks.Add(ctx, you.ID, name, "mint", model.Pending, nil); err != nil {
			t.Fatal(err)
		}
	}
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	// a new task has no delete; an existing one does, posting to its own url
	if page := body(t, get(h, "/pending/new", in)); strings.Contains(page, "Delete task") {
		t.Error("the new task form offers a delete")
	}
	if page := body(t, get(h, "/pending/1", in)); !strings.Contains(page, `formaction="/pending/1/delete"`) {
		t.Error("the edit form has no delete")
	}

	res := postAs(h, "/pending/1/delete", nil, in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/pending?deleted=1" {
		t.Fatalf("delete: %d to %q, want 303 to /pending?deleted=1", res.StatusCode, res.Header.Get("Location"))
	}
	page := body(t, get(h, "/pending?deleted=1", in))
	for _, want := range []string{"Deleted “Fix bike”", `action="/pending/1/restore"`, `role="status" data-undo`} {
		if !strings.Contains(page, html.EscapeString(want)) && !strings.Contains(page, want) {
			t.Errorf("the tab after a delete lacks %q", want)
		}
	}
	if rowRe.FindAllString(page, -1) == nil || strings.Contains(page, `href="/pending/1"`) {
		t.Error("the deleted task is still listed")
	}
	if page := body(t, get(h, "/pending?deleted=2", in)); strings.Contains(page, `role="status"`) {
		t.Error("an undo toast for a task that is not deleted")
	}

	res = postAs(h, "/pending/1/restore", nil, in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/pending" {
		t.Fatalf("undo: %d to %q, want 303 to /pending", res.StatusCode, res.Header.Get("Location"))
	}
	if tasks, _ := svc.Tasks.List(ctx, you.ID); len(tasks) != 2 {
		t.Errorf("%d tasks after undo, want 2", len(tasks))
	}

	// too late: a later delete erased the first
	postAs(h, "/pending/1/delete", nil, in)
	postAs(h, "/pending/2/delete", nil, in)
	res = postAs(h, "/pending/1/restore", nil, in)
	if res.StatusCode != http.StatusConflict || !strings.Contains(body(t, res), "Too late to undo") {
		t.Errorf("a late undo: %d, want 409 with an error toast", res.StatusCode)
	}
	if res := postAs(h, "/pending/9/delete", nil, in); res.StatusCode != http.StatusNotFound {
		t.Errorf("delete a missing task: %d, want 404", res.StatusCode)
	}
}

// weekly: the schedule boxes come back as placements; a box with no days is
// dropped; the edit form shows them again.
func TestWeeklyForm(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	// a new form has one blank box, and the blank one "+ Add time" copies
	page := body(t, get(h, "/pending/new", in))
	for _, want := range []string{`name="place" value="0"`, `name="place" value="__i__"`, `name="start-0" value="" class="h-12 w-full border-2 border-border bg-surface px-3 text-body text-text" inputmode="numeric"`, `name="days-0" value="1"`, "Weekly schedule"} {
		if !strings.Contains(page, want) {
			t.Errorf("the new form lacks %q", want)
		}
	}

	form := url.Values{
		"name": {"Gym"}, "area": {"apricot"}, "when": {"weekly"},
		"place":  {"0", "3", "5"},
		"days-0": {"3", "6"}, "start-0": {"18:00"}, "end-0": {"19:30"}, "notify_start-0": {"on"},
		"days-3": {"1"}, "start-3": {"7"}, "end-3": {"0800"}, "notify_end-3": {"on"}, // as typed on a phone
		"start-5": {"12:00"}, "end-5": {"13:00"}, // no days: dropped
	}
	res := postAs(h, "/pending/new", form, in)
	if res.StatusCode != http.StatusSeeOther || res.Header.Get("Location") != "/week" {
		t.Fatalf("create weekly: %d to %q, want 303 to /week", res.StatusCode, res.Header.Get("Location"))
	}
	places, _ := svc.Tasks.Placements(ctx, you.ID, 1)
	if len(places) != 2 {
		t.Fatalf("%d placements, want 2: %+v", len(places), places)
	}
	mon, wedSat := places[0], places[1] // by start
	if mon.Days != model.DaysOf(time.Monday) || mon.Start.String() != "07:00" || mon.NotifyStart || !mon.NotifyEnd {
		t.Errorf("monday = %+v", mon)
	}
	if wedSat.Days != model.DaysOf(time.Wednesday, time.Saturday) || wedSat.End.String() != "19:30" || !wedSat.NotifyStart {
		t.Errorf("wed+sat = %+v", wedSat)
	}

	// the edit form shows them, in order, with their days on
	// weekly tasks open from the week in m16; until then from any tab
	page = body(t, get(h, "/pending/1", in))
	for _, want := range []string{
		`value="weekly" class="sr-only" checked`,
		`name="days-0" value="1" class="sr-only" checked`,
		`name="start-0" value="07:00"`,
		`name="days-1" value="3" class="sr-only" checked`,
		`name="days-1" value="6" class="sr-only" checked`,
		`name="end-1" value="19:30"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the edit form lacks %q", want)
		}
	}

	// to pending: the schedule left in the form is ignored, the times go
	edit := url.Values{"name": {"Gym"}, "area": {"apricot"}, "when": {"pending"}, "place": {"0"}, "days-0": {"1"}, "start-0": {"bad"}}
	if res := postAs(h, "/pending/1", edit, in); res.StatusCode != http.StatusSeeOther {
		t.Fatalf("to pending: %d", res.StatusCode)
	}
	if places, _ := svc.Tasks.Placements(ctx, you.ID, 1); len(places) != 0 {
		t.Errorf("a pending task kept %d placements", len(places))
	}
}

func TestWeeklyFormErrors(t *testing.T) {
	h, svc := testApp(t)
	if _, err := svc.Users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	in := &http.Cookie{Name: userCookieName, Value: "1"}
	tests := []struct {
		days       []string
		start, end string
		toast      string
	}{
		{nil, "18:00", "19:00", "Pick at least one day"},
		{[]string{"1"}, "", "19:00", "A time needs a start and an end"},
		{[]string{"1"}, "19:00", "18:00", "The end is before the start"},
		{[]string{"1"}, "7pm", "8pm", "Times are HH:MM, 24h"},
	}
	for _, tt := range tests {
		t.Run(tt.toast, func(t *testing.T) {
			form := url.Values{"name": {"Gym"}, "area": {"apricot"}, "when": {"weekly"}, "place": {"0"},
				"days-0": tt.days, "start-0": {tt.start}, "end-0": {tt.end}}
			res := postAs(h, "/pending/new", form, in)
			if res.StatusCode != http.StatusUnprocessableEntity {
				t.Fatalf("status %d, want 422", res.StatusCode)
			}
			page := body(t, res)
			if !strings.Contains(page, tt.toast) {
				t.Errorf("no toast %q", tt.toast)
			}
			// as typed
			if !strings.Contains(page, `name="start-0" value="`+tt.start+`"`) || !strings.Contains(page, `value="weekly" class="sr-only" checked`) {
				t.Error("the schedule is not as typed")
			}
		})
	}
}
