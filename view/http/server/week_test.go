package server

import (
	"context"
	"html"
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
	hourRe  = regexp.MustCompile(`text-text-muted [^"]*" style="top: [\d.]+%">(\d\d:00)</span>`)
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

var chipRe = regexp.MustCompile(`<a href="(/week/\d+)" data-task class="inline-flex[^"]*" style="--fill: var\(--task-(\w+)\)"><span class="truncate">([^<]+)</span></a>`)

// dated tasks on the week (figma): the dentist on the timeline, inside work;
// the birthday and the deadlines as chips; a stretch a chip on each of its days.
func TestWeekDated(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	weekly(t, svc, you.ID, "Work", "sky", model.DaysOf(time.Wednesday), 9*60, 17*60)
	dated(t, svc, you.ID, "Dentist", "coral", "2026-10-14 2026-10-14 11:00 12:00")
	dated(t, svc, you.ID, "🎂 Ana's birthday", "pink", "2026-10-14 2026-10-14 - -")
	dated(t, svc, you.ID, "Report", "sky", "- 2026-10-14 - 14:00")
	dated(t, svc, you.ID, "Taxes", "apricot", "- 2026-10-16 - -")
	dated(t, svc, you.ID, "Trip", "lime", "2026-10-17 2026-10-20 - -")
	dated(t, svc, you.ID, "Taxi", "mint", "2026-10-13 2026-10-13 05:30 06:00")
	dated(t, svc, you.ID, "Later", "pink", "2026-10-21 2026-10-21 09:00 10:00") // next week
	page := body(t, get(h, "/week", &http.Cookie{Name: userCookieName, Value: "1"}))

	var chips []string
	for _, m := range chipRe.FindAllStringSubmatch(page, -1) {
		chips = append(chips, html.UnescapeString(m[3])+"/"+m[2])
	}
	// wed: all day first, then due; fri: due (coral, whatever the area); sat, sun: the
	// trip. twice: the phone's lanes, then the desktop's
	want := "🎂 Ana's birthday/pink, Report · due 14:00/coral, Taxes · due/coral, Trip/lime, Trip/lime"
	want += ", " + want
	if strings.Join(chips, ", ") != want {
		t.Errorf("chips = %q\nwant    %s", strings.Join(chips, ", "), want)
	}

	var blocks []string
	for _, b := range blockRe.FindAllStringSubmatch(page, -1) {
		blocks = append(blocks, b[8]+" "+b[4]+"px")
	}
	// the taxi stretches the hours to 05:00; on wed the dentist nests in work
	if want := "Taxi 0px, Work 0px, Dentist 40px"; strings.Join(blocks, ", ") != want {
		t.Errorf("blocks = %q, want %s", strings.Join(blocks, ", "), want)
	}
	if !strings.Contains(page, `data-from="5" data-span="17"`) {
		t.Error("the hours do not run 05–22 (the taxi stretches them; a deadline's hour does not)")
	}
	if strings.Contains(page, "Later") {
		t.Error("next week's task shows")
	}
	// the phone's lanes: only days with chips, today's shown
	var lanes []string
	for _, m := range laneRe.FindAllStringSubmatch(page, -1) {
		lanes = append(lanes, m[1]+m[2])
	}
	if want := "2 data-shown, 4, 5, 6"; strings.Join(lanes, ", ") != want {
		t.Errorf("phone lanes = %q, want %s", strings.Join(lanes, ", "), want)
	}
}

var (
	laneRe = regexp.MustCompile(`data-shown:flex" data-day="(\d)"( data-shown)?>`)
	headRe = regexp.MustCompile(`<a href="(/week\?date=[\d-]+)" data-day="(\d)" data-name="(\w+)"( data-shown)?`)
	colRe  = regexp.MustCompile(`md:block" data-day="(\d)"( data-shown)?>`)
)

// the phone shows one day (figma: Week — Phone): today, or ?date='s. the day
// picker links every day; the day's head, lane and column are marked shown,
// and the title names it.
func TestWeekPhone(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	weekly(t, svc, you.ID, "Work", "sky", model.DaysOf(time.Wednesday, time.Thursday), 9*60, 17*60)
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	for _, tt := range []struct {
		path, name string
		shown      string
	}{
		{"/week", "Wednesday", "2"},
		{"/week?date=2026-10-15", "Thursday", "3"},
	} {
		page := body(t, get(h, tt.path, in))
		heads := headRe.FindAllStringSubmatch(page, -1)
		if len(heads) != 7 || heads[0][1] != "/week?date=2026-10-12" || heads[6][3] != "Sunday" {
			t.Fatalf("%s: %d day picker links, want mon 12 … sunday", tt.path, len(heads))
		}
		var shown []string
		for _, m := range heads {
			if m[4] != "" {
				shown = append(shown, m[2])
			}
		}
		for _, m := range colRe.FindAllStringSubmatch(page, -1) {
			if m[2] != "" {
				shown = append(shown, m[1])
			}
		}
		if strings.Join(shown, " ") != tt.shown+" "+tt.shown {
			t.Errorf("%s: shown %v, want the head and column of day %s", tt.path, shown, tt.shown)
		}
		for _, want := range []string{
			"Week · Oct 12 – 18", `data-day-name>` + tt.name + `</h1>`,
			`data-before="/week?date=2026-10-11" data-after="/week?date=2026-10-19"`,
		} {
			if !strings.Contains(page, want) {
				t.Errorf("%s lacks %q", tt.path, want)
			}
		}
	}
	// thursday picked: today keeps its ink border, no butter of its own
	page := body(t, get(h, "/week?date=2026-10-15", in))
	if !strings.Contains(page, `<span class="border-2 border-border flex size-10 items-center justify-center text-title text-text group-data-shown:border-2 group-data-shown:border-border group-data-shown:bg-accent">14</span>`) {
		t.Error("today is not an ink border in the phone's day picker")
	}
}
