package store

import (
	"context"
	"testing"
	"time"

	"sike/domain/model"
)

func TestNotificationsSent(t *testing.T) {
	ctx := context.Background()
	s := New(testDB(t))
	u, _ := s.CreateUser(ctx, model.User{Name: "You", Emoji: "🦊", Warn: model.DefaultWarn}, nil)
	at := time.Date(2026, 10, 14, 18, 0, 0, 0, time.UTC)
	key := "start:p1:2026-10-14T18:00"
	sent := func() bool {
		t.Helper()
		ok, err := s.WasSent(ctx, u.ID, key)
		if err != nil {
			t.Fatal(err)
		}
		return ok
	}

	if sent() {
		t.Error("sent before it was")
	}
	for range 2 { // twice is once
		if err := s.MarkSent(ctx, u.ID, key, at); err != nil {
			t.Fatal(err)
		}
	}
	if !sent() {
		t.Error("not sent after MarkSent")
	}
	if err := s.ForgetSent(ctx, at); err != nil || !sent() {
		t.Errorf("forgetting what is older than it forgot it: %v", err)
	}
	if err := s.ForgetSent(ctx, at.Add(time.Second)); err != nil || sent() {
		t.Errorf("forgetting what came before a later time kept it: %v", err)
	}
}
