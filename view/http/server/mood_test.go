package server

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"sike/domain/clock"
)

// today's mood on the week (figma): the five faces, in the desktop header and
// the phone's row; a tap posts it for today (the clock: wed 2026-10-14).
func TestMood(t *testing.T) {
	h, svc := testApp(t)
	you, _ := svc.Users.Add(context.Background(), "You", "🦊")
	in := &http.Cookie{Name: userCookieName, Value: "1"}

	page := body(t, get(h, "/week", in))
	if n := strings.Count(page, `hx-post="/mood"`); n != 2 || strings.Count(page, `name="mood" value="`) != 10 || strings.Contains(page, `name="mood" value="4" class="sr-only" checked`) {
		t.Errorf("%d mood rows; want 2 (desktop, phone), five faces each, none on", n)
	}
	if res := postAs(h, "/mood", url.Values{"mood": {"4"}}, in); res.StatusCode != http.StatusNoContent {
		t.Fatalf("mood 4: status %d, want 204", res.StatusCode)
	}
	if m, _ := svc.Moods.Get(context.Background(), you.ID, clock.NewDate(2026, 10, 14)); m != 4 {
		t.Errorf("today's mood = %d, want 4", m)
	}
	page = body(t, get(h, "/week", in))
	if strings.Count(page, `name="mood" value="4" class="sr-only" checked`) != 2 {
		t.Error("the week does not show 🙂 on")
	}
	if res := postAs(h, "/mood", url.Values{"mood": {"9"}}, in); res.StatusCode != http.StatusBadRequest {
		t.Errorf("mood 9: status %d, want 400", res.StatusCode)
	}
	if res := post(h, "/mood", url.Values{"mood": {"3"}}); res.StatusCode != http.StatusSeeOther {
		t.Errorf("a mood from nobody: status %d, want the redirect to who are you", res.StatusCode)
	}
}
