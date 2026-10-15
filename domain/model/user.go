package model

import "slices"

// User is one person. each user is a separate world: nothing is shared.
type User struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Emoji string `db:"emoji"`
	Warn  int    `db:"warn"`  // the deadline warning, in minutes: one of Warnings
	Topic string `db:"topic"` // their ntfy topic; "" is no notifications
}

// IsTopic: an ntfy topic name, 1–64 of letters, digits, - and _ (as ntfy
// takes them), or "" for none.
func IsTopic(t string) bool {
	if len(t) > 64 {
		return false
	}
	for _, r := range t {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// Warnings are the deadline warnings a person can pick: 1 hour, 1 day (a new
// person's), 2 days.
var Warnings = []int{60, 24 * 60, 2 * 24 * 60}

// DefaultWarn is a new person's deadline warning.
const DefaultWarn = 24 * 60

func IsWarning(w int) bool { return slices.Contains(Warnings, w) }

// WarnLabel says a warning as settings does: "1 hour", "1 day", "2 days".
func WarnLabel(w int) string {
	return map[int]string{60: "1 hour", 24 * 60: "1 day", 2 * 24 * 60: "2 days"}[w]
}

// Emojis are the faces a person can pick, in picker order.
var Emojis = []string{
	"🦊", "🐼", "🐻", "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵",
	"🐔", "🐧", "🐦", "🦉", "🐺", "🐗", "🐴", "🦄", "🐝", "🐢",
	"🐙", "🐬", "🐳", "🦋", "🐌", "🐞", "🦔", "🦦", "🦥", "🐱",
}

func IsEmoji(e string) bool { return slices.Contains(Emojis, e) }
