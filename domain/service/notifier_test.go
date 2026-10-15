package service

import (
	"context"
	"errors"
	"testing"

	"sike/domain/model"
	"sike/domain/port"
)

type fakePort struct{ sent []string }

func (f *fakePort) Notify(_ context.Context, topic string, m port.Message) error {
	f.sent = append(f.sent, topic+": "+m.Title+" · "+m.Body)
	return nil
}

func TestNotifierTest(t *testing.T) {
	you := model.User{ID: 1, Name: "You", Emoji: "🦊"}
	f := &fakePort{}
	n := NewNotifier(f)
	if err := n.Test(context.Background(), you, "sike-you"); err != nil {
		t.Fatal(err)
	}
	if len(f.sent) != 1 || f.sent[0] != "sike-you: 🦊 sike · Notifications reach you, You." {
		t.Errorf("sent %q", f.sent)
	}
	for _, topic := range []string{"", "sike you", "sike/you"} {
		if err := n.Test(context.Background(), you, topic); !errors.Is(err, model.ErrInvalid) {
			t.Errorf("topic %q: err = %v, want ErrInvalid", topic, err)
		}
	}
	if err := NewNotifier(nil).Test(context.Background(), you, "sike-you"); !errors.Is(err, model.ErrInvalid) {
		t.Errorf("no ntfy: err = %v, want ErrInvalid (notifications off)", err)
	}
}
