package server

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"sike/domain/clock"
	"sike/domain/model"
)

var (
	blockRe = regexp.MustCompile(`style="top: calc\(([\d.]+)% \+ 3px\); height: calc\(([\d.]+)% - 6px\); left: calc\(([\d.]+)% \+ (\d+)px \+ 3px\); width: calc\(([\d.]+)% - \d+px - 6px\)">\s*<div[^>]*>\s*<a href="(/week/\d+)" data-task class="[^"]*" style="--fill: var\(--task-(\w+)\)">\s*<span[^>]*>([^<]+)</span>(?:\s*<span[^>]*>([^<]+)</span>)?`)
	hourRe  = regexp.MustCompile(`text-text-muted" style="top: [\d.]+%">(\d\d:00)</span>`)
	nowRe   = regexp.MustCompile(`style="top: ([\d.]+)%" data-now="(\d+)"`)
)

func weekly(t *testing.T, svc services, user int64, name, color string, days model.Days, start, end int) {
	t.Helper()
	p := model.Placement{Days: days, Start: clock.Time(start), End: clock.Time(end)}
	if _, err := svc.Tasks.Add(context.Background(), model.Task{UserID: user, Name: name, Color: color, When: model.Weekly, Places: []model.Placement{p}}); err != nil {
		t.Fatal(err)
	}
}

func TestWeek(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	weekdays := model.DaysOf(time.Monday, time.Tuesday, time.Wednesday, time.Thursday, time.Friday)
	weekly(t, svc, you.ID, "Work", "sky", weekdays, 9*60, 17*60)
	weekly(t, svc, you.ID, "Gym", "apricot", model.DaysOf(time.Monday, time.Wednesday), 18*60, 19*60+30)
	weekly(t, svc, you.ID, "Morning routine", "lavender", model.DaysOf(model.Week...), 7*60+30, 8*60+30)
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	page := body(t, get(h, "/week", in))
	for _, want := range []string{
		"Oct 12 – 18 · week 42",
		`href="/week?date=2026-10-05"`, `href="/week?date=2026-10-19"`, // ‹ ›
		`href="/week/new"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the week lacks %q", want)
		}
	}
	// today (wed 14) is the one butter day head
	if n := strings.Count(page, `border-2 border-border bg-accent flex size-10`); n != 1 || !strings.Contains(page, `bg-accent flex size-10 items-center justify-center text-title text-text">14</span>`) {
		t.Errorf("%d today day heads, want wednesday 14 only", n)
	}

	// 07–22: fifteen hour lines, 07:00 to 21:00
	var hours []string
	for _, m := range hourRe.FindAllStringSubmatch(page, -1) {
		hours = append(hours, m[1])
	}
	if len(hours) != 15 || hours[0] != "07:00" || hours[14] != "21:00" {
		t.Errorf("hours = %v, want 07:00 … 21:00", hours)
	}

	blocks := blockRe.FindAllStringSubmatch(page, -1)
	// 7 mornings + 5 work days + 2 gyms
	if len(blocks) != 14 {
		t.Fatalf("%d blocks, want 14", len(blocks))
	}
	// monday, in time order: morning, work, gym. every block carries its time;
	// css shows it when the block is tall enough
	mon := blocks[:3]
	for i, want := range []struct{ top, height, name, time, color string }{
		{"3.3333", "6.6667", "Morning routine", "07:30–08:30", "lavender"}, // 07:30, 1 h of 15
		{"13.3333", "53.3333", "Work", "09:00–17:00", "sky"},
		{"73.3333", "10.0000", "Gym", "18:00–19:30", "apricot"},
	} {
		b := mon[i]
		if b[1] != want.top || b[2] != want.height || b[8] != want.name || b[9] != want.time || b[7] != want.color {
			t.Errorf("monday block %d = top %s height %s %q %q %s; want %+v", i, b[1], b[2], b[8], b[9], b[7], want)
		}
	}
	if mon[1][6] != "/week/1" {
		t.Errorf("work opens %s, want /week/1", mon[1][6])
	}

	// now: wednesday 18:45 = 78.3333% of 07–22, on one day only
	nows := nowRe.FindAllStringSubmatch(page, -1)
	if len(nows) != 1 || nows[0][1] != "78.3333" || nows[0][2] != "1125" {
		t.Errorf("now lines = %v, want one at 78.3333%% (minute 1125)", nows)
	}

	// another week: no today, no now-line
	page = body(t, get(h, "/week?date=2026-10-21", in))
	if !strings.Contains(page, "Oct 19 – 25 · week 43") || nowRe.MatchString(page) || strings.Contains(page, "bg-accent flex size-10") {
		t.Error("next week shows today or the now-line")
	}
	// a week across months
	if page := body(t, get(h, "/week?date=2026-10-01", in)); !strings.Contains(page, "Sep 28 – Oct 4 · week 40") {
		t.Error("a week across months is not titled Sep 28 – Oct 4")
	}
}

func TestWeekHoursStretch(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	weekly(t, svc, you.ID, "Taxi", "coral", model.DaysOf(time.Sunday), 5*60+30, 6*60)
	weekly(t, svc, you.ID, "Late", "pink", model.DaysOf(time.Friday), 22*60, 23*60+15)
	page := body(t, get(h, "/week", &http.Cookie{Name: userCookieName, Value: "1"}))
	var hours []string
	for _, m := range hourRe.FindAllStringSubmatch(page, -1) {
		hours = append(hours, m[1])
	}
	if len(hours) != 19 || hours[0] != "05:00" || hours[18] != "23:00" {
		t.Errorf("hours = %v, want 05:00 … 23:00 (05–24)", hours)
	}
	if !strings.Contains(page, `data-from="5" data-span="19"`) {
		t.Error("the timeline does not say its hours to app.js")
	}
}

func TestWeekForm(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	weekly(t, svc, you.ID, "Gym", "apricot", model.DaysOf(time.Wednesday), 18*60, 19*60)
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	// "+ New task" on week presets every week, over the week
	page := body(t, get(h, "/week/new", in))
	if !strings.Contains(page, `value="weekly" class="sr-only" checked`) || !strings.Contains(page, `action="/week/new"`) || !blockRe.MatchString(page) {
		t.Error("the form from week is not preset to every week, over the week")
	}
	// a block opens its task
	page = body(t, get(h, "/week/1", in))
	if !strings.Contains(page, `name="name" value="Gym"`) || !strings.Contains(page, `name="days-0" value="3" class="sr-only" checked`) {
		t.Error("the block's form lacks the task")
	}
}

// overlap on the page: the dentist sits inside work, 40px in; two at the same
// time share the column.
func TestWeekOverlap(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	wed := model.DaysOf(time.Wednesday)
	weekly(t, svc, you.ID, "Dentist", "coral", wed, 11*60, 12*60)
	weekly(t, svc, you.ID, "Work", "sky", wed, 9*60, 17*60)
	weekly(t, svc, you.ID, "Gym", "apricot", wed, 18*60, 19*60)
	weekly(t, svc, you.ID, "Call", "lime", wed, 18*60, 19*60)
	page := body(t, get(h, "/week", &http.Cookie{Name: userCookieName, Value: "1"}))

	var got []string
	for _, b := range blockRe.FindAllStringSubmatch(page, -1) {
		got = append(got, b[8]+" "+b[3]+"+"+b[5]+" "+b[4]+"px")
	}
	// drawn parents first: work, then the dentist over it
	want := []string{"Work 0.0000+100.0000 0px", "Dentist 0.0000+100.0000 40px", "Gym 0.0000+50.0000 0px", "Call 50.0000+50.0000 0px"}
	if strings.Join(got, ", ") != strings.Join(want, ", ") {
		t.Errorf("blocks = %q\nwant      %q", got, want)
	}
}
