package server

import (
	"context"
	"html"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"sike/domain/clock"
	"sike/domain/model"
)

var (
	upGroupRe = regexp.MustCompile(`<h2 class="pb-1 text-label text-text-muted uppercase">([^<]+)</h2>`)
	upRowRe   = regexp.MustCompile(`<a href="/upcoming/\d+" data-task class="(opacity-50 )?group[^"]*">(?s:.*?)<span class="truncate text-body text-text">([^<]+)</span>\s*<span class="(text-danger|text-text-muted) font-mono text-mono">([^<]+)</span>`)
)

func dated(t *testing.T, svc services, user int64, name, color, spec string) {
	t.Helper()
	d := model.Dates{Start: clock.NoTime, End: clock.NoTime}
	// spec: "from to start end", "-" for none
	f := strings.Fields(spec)
	if f[0] != "-" {
		d.From, _ = clock.ParseDate(f[0])
	}
	d.To, _ = clock.ParseDate(f[1])
	if f[2] != "-" {
		d.Start, _ = clock.ParseTime(f[2])
	}
	if f[3] != "-" {
		d.End, _ = clock.ParseTime(f[3])
	}
	if _, err := svc.Tasks.Add(context.Background(), model.Task{UserID: user, Name: name, Color: color, When: model.Dated, Date: &d}); err != nil {
		t.Fatal(err)
	}
}

// the clock is stopped on wed 2026-10-14 at 18:45 (server_test.go).
func TestUpcoming(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	for _, x := range [][3]string{
		{"Renew passport", "lavender", "- 2026-11-03 - -"},
		{"Dentist", "coral", "2026-10-15 2026-10-15 11:00 12:00"},
		{"Taxes", "apricot", "- 2026-10-16 - -"},
		{"Trip to Lisbon", "sky", "2026-11-02 2026-11-06 - -"},
		{"Mom's birthday", "pink", "2026-11-21 2026-11-21 - -"},
		{"Market", "mint", "2026-10-20 2026-10-20 - -"},
		{"Report", "sky", "- 2026-10-14 - 20:00"}, // tonight
		{"New year", "lime", "2027-01-15 2027-01-15 - -"},
		{"Haircut", "lavender", "2026-10-12 2026-10-12 10:00 11:00"},
		{"Pay rent", "coral", "- 2026-10-01 - -"},
		{"Leo's birthday", "pink", "2026-10-04 2026-10-04 - -"},
		{"Standup", "sky", "2026-10-14 2026-10-14 09:00 09:15"}, // this morning
	} {
		dated(t, svc, you.ID, x[0], x[1], x[2])
	}
	if _, err := svc.Tasks.Add(context.Background(), model.Task{UserID: you.ID, Name: "Fix bike", Color: "mint", When: model.Pending}); err != nil {
		t.Fatal(err)
	}
	page := body(t, get(h, "/upcoming", &http.Cookie{Name: userCookieName, Value: "1"}))

	var groups []string
	for _, m := range upGroupRe.FindAllStringSubmatch(page, -1) {
		groups = append(groups, m[1])
	}
	if want := "This week, Next week, November, January 2027"; strings.Join(groups, ", ") != want {
		t.Errorf("groups = %q, want %s", groups, want)
	}
	var rows []string
	for _, m := range upRowRe.FindAllStringSubmatch(page, -1) {
		r := html.UnescapeString(m[2]) + " | " + m[4]
		if m[3] == "text-danger" {
			r += " !"
		}
		if m[1] != "" {
			r += " (past)"
		}
		rows = append(rows, r)
	}
	want := []string{
		"Report | Due Wed 14 · 20:00 · today !",
		"Dentist | Tomorrow · Thu 15 · 11:00–12:00",
		"Taxes | Due Fri 16", // a day's warning (the default): not near yet
		"Market | Tue 20 · all day",
		"Trip to Lisbon | Mon 2 – Fri 6 · 5 days",
		"Renew passport | Due Tue 3",
		"Mom's birthday | Sat 21 · all day",
		"New year | Fri 15 · all day",
		// the past, most recent first
		"Standup | Wed 14 · 09:00–09:15 (past)",
		"Haircut | Mon 12 · 10:00–11:00 (past)",
		"Leo's birthday | Sun 4 · all day (past)",
		"Pay rent | Due Thu 1 (past)",
	}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows:\n  %s\nwant\n  %s", strings.Join(rows, "\n  "), strings.Join(want, "\n  "))
	}
	if !strings.Contains(page, "Past · 4") || strings.Contains(page, "<details open") {
		t.Error("the past is not a closed “Past · 4”")
	}
	// the sidebar counts what is coming: 8
	if !regexp.MustCompile(`Upcoming</span>\s*<span class="font-mono text-mono text-text-muted">8</span>`).MatchString(page) {
		t.Error("the sidebar does not count 8 upcoming")
	}
	if strings.Contains(page, "Nothing coming up") || strings.Contains(page, "Fix bike") {
		t.Error("the empty state or a pending task shows")
	}

	// with two days' warning, the taxes are near
	you.Warn = 2 * 24 * 60
	if _, err := svc.Users.Edit(context.Background(), you); err != nil {
		t.Fatal(err)
	}
	page = body(t, get(h, "/upcoming", &http.Cookie{Name: userCookieName, Value: "1"}))
	if !strings.Contains(page, `<span class="text-danger font-mono text-mono">Due Fri 16 · in 2 days</span>`) {
		t.Error("with two days' warning, the taxes are not near")
	}
}

func TestUpcomingEmpty(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	in := &http.Cookie{Name: userCookieName, Value: "1"}
	page := body(t, get(h, "/upcoming", in))
	if !strings.Contains(page, "Nothing coming up") || strings.Contains(page, "Past ·") {
		t.Error("an empty upcoming should show only “Nothing coming up”")
	}
	// only the past: the line, and the past under it
	dated(t, svc, you.ID, "Haircut", "lavender", "2026-10-12 2026-10-12 10:00 11:00")
	page = body(t, get(h, "/upcoming", in))
	if !strings.Contains(page, "Nothing coming up") || !strings.Contains(page, "Past · 1") {
		t.Error("with only the past: want the empty line and “Past · 1”")
	}
}

func TestUpcomingForm(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	dated(t, svc, you.ID, "Dentist", "coral", "2026-10-15 2026-10-15 11:00 12:00")
	in := &http.Cookie{Name: userCookieName, Value: "1"}
	if page := body(t, get(h, "/upcoming/new", in)); !strings.Contains(page, `value="date" class="sr-only" checked`) || !strings.Contains(page, `action="/upcoming/new"`) {
		t.Error("the form from upcoming is not preset to on a date")
	}
	if page := body(t, get(h, "/upcoming/1", in)); !strings.Contains(page, `name="date_from" value="Thu, Oct 15"`) {
		t.Error("a row's form lacks its date")
	}
}
