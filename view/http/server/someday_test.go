package server

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"sike/domain/model"
)

var tileRe = regexp.MustCompile(`style="--fill: var\(--task-(\w+)\)">([^<]+)</a>`)

func TestSomedayTilesByArea(t *testing.T) {
	h, svc := testApp(t)
	ctx := context.Background()
	you, _ := svc.Users.Add(ctx, "You", "🦊")
	for _, x := range [][3]string{
		{"Learn piano", "pink", "someday"},
		{"Read Dune", "lavender", "someday"},
		{"Visit Lisbon", "pink", "someday"},
		{"Fix bike", "mint", "pending"}, // not a wish
	} {
		if _, err := svc.Tasks.Add(ctx, you.ID, x[0], x[1], model.When(x[2]), nil); err != nil {
			t.Fatal(err)
		}
	}
	page := body(t, get(h, "/someday", &http.Cookie{Name: userCookieName, Value: "1"}))

	var groups []string
	for _, m := range groupRe.FindAllStringSubmatch(page, -1) {
		groups = append(groups, m[1]+" "+m[2])
	}
	if want := "Self 1, Fun 2"; strings.Join(groups, ", ") != want {
		t.Errorf("groups = %q, want %s", groups, want)
	}
	var tiles []string
	for _, m := range tileRe.FindAllStringSubmatch(page, -1) {
		tiles = append(tiles, m[2]+"/"+m[1])
	}
	if want := "Read Dune/lavender, Visit Lisbon/pink, Learn piano/pink"; strings.Join(tiles, ", ") != want {
		t.Errorf("tiles = %q, want %s (newest first, each in its area's color)", tiles, want)
	}
	if strings.Contains(page, "Fix bike") || strings.Contains(page, "No wishes yet") {
		t.Error("a pending task or the empty state shows among wishes")
	}
}

func TestSomedayEmpty(t *testing.T) {
	h, svc := testApp(t)
	if _, err := svc.Users.Add(context.Background(), "You", "🦊"); err != nil {
		t.Fatal(err)
	}
	page := body(t, get(h, "/someday", &http.Cookie{Name: userCookieName, Value: "1"}))
	if !strings.Contains(page, "No wishes yet") || tileRe.MatchString(page) {
		t.Error("an empty someday tab should show only “No wishes yet”")
	}
}
