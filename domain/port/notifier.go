// Package port declares what the domain needs from the outside world and
// does not store: today, a way to reach a person's phone.
package port

import "context"

// Notifier sends a message to a topic a person subscribes to (ntfy).
type Notifier interface {
	Notify(ctx context.Context, topic string, m Message) error
}

// Message is a notification: a title and a line, informational only
// (product/spec.md: "▶ Gym · until 19:30").
type Message struct {
	Title string
	Body  string
}
