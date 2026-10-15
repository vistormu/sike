package server

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"sike/domain/model"
)

var (
	groupRe = regexp.MustCompile(`<span class="text-label text-text uppercase">([^<]+)</span>\s*<span class="font-mono text-mono text-text-muted">(\d+)</span>`)
	rowRe   = regexp.MustCompile(`<span class="truncate text-body text-text">([^<]+)</span>`)
)

func TestPendingGroupsByArea(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	them, _ := svc.Users.Add(ctx, "Partner", "🐼")
	add := func(user int64, name, color string, when model.When) {
		if _, err := svc.Tasks.Add(ctx, user, name, color, when, nil); err != nil {
			t.Fatal(err)
		}
	}
	add(you.ID, "Return package", "coral", model.Pending)
	add(you.ID, "Call the bank", "coral", model.Pending)
	add(you.ID, "Fix bike", "mint", model.Pending)
	add(you.ID, "Book haircut", "lavender", model.Pending)
	add(you.ID, "Learn piano", "lavender", model.Someday) // not pending
	add(them.ID, "Their thing", "sky", model.Pending)     // not yours
	if err := svc.Areas.Rename(ctx, you.ID, "mint", "House"); err != nil {
		t.Fatal(err)
	}

	page := body(t, get(h, "/pending", &http.Cookie{Name: userCookieName, Value: "1"}))

	var groups []string
	for _, m := range groupRe.FindAllStringSubmatch(page, -1) {
		groups = append(groups, m[1]+" "+m[2])
	}
	if want := []string{"Self 1", "Admin 2", "House 1"}; strings.Join(groups, ", ") != strings.Join(want, ", ") {
		t.Errorf("groups = %q, want %q (palette order, empty hidden, the person's names)", groups, want)
	}
	var rows []string
	for _, m := range rowRe.FindAllStringSubmatch(page, -1) {
		rows = append(rows, m[1])
	}
	if want := []string{"Book haircut", "Call the bank", "Return package", "Fix bike"}; strings.Join(rows, ", ") != strings.Join(want, ", ") {
		t.Errorf("rows = %q, want %q (newest first in a group)", rows, want)
	}
	if strings.Contains(page, "Nothing pending") {
		t.Error("the empty state shows next to tasks")
	}
	// sidebar counts: pending 4, someday 1
	for _, want := range []string{"Pending 4", "Someday 1"} {
		label, n, _ := strings.Cut(want, " ")
		re := regexp.MustCompile(label + `</span>\s*<span class="font-mono text-mono text-text-muted">` + n + `</span>`)
		if !re.MatchString(page) {
			t.Errorf("the sidebar lacks the count %q", want)
		}
	}
}

func TestPendingEmpty(t *testing.T) {
	h, svc := testApp(t)
	if _, err := svc.Users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	page := body(t, get(h, "/pending", &http.Cookie{Name: userCookieName, Value: "1"}))
	if !strings.Contains(page, "Nothing pending") || groupRe.MatchString(page) {
		t.Error("an empty pending tab should show only “Nothing pending”")
	}
}
