package service

import (
	"context"

	"sike/domain/model"
	"sike/domain/port"
)

// Notifier sends people their notifications (product/spec.md), to their
// ntfy topic. no port (SIKE_NTFY_URL unset) is notifications off.
type Notifier struct {
	port port.Notifier
}

func NewNotifier(p port.Notifier) *Notifier {
	return &Notifier{port: p}
}

// Test sends a test message to topic, the one a person typed in settings (it
// may not be saved yet): "Send test".
func (n *Notifier) Test(ctx context.Context, u model.User, topic string) error {
	switch {
	case n.port == nil:
		return model.Invalid("Notifications are off on this server")
	case topic == "":
		return model.Invalid("Type an ntfy topic first")
	case !model.IsTopic(topic):
		return model.Invalid("A topic is up to 64 letters, digits, - and _")
	}
	return n.port.Notify(ctx, topic, port.Message{Title: u.Emoji + " sike", Body: "Notifications reach you, " + u.Name + "."})
}
