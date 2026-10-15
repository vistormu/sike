package model

import "slices"

// User is one person. each user is a separate world: nothing is shared.
type User struct {
	ID    int64  `db:"id"`
	Name  string `db:"name"`
	Emoji string `db:"emoji"`
}

// Emojis are the faces a person can pick, in picker order.
var Emojis = []string{
	"🦊", "🐼", "🐻", "🐨", "🐯", "🦁", "🐮", "🐷", "🐸", "🐵",
	"🐔", "🐧", "🐦", "🦉", "🐺", "🐗", "🐴", "🦄", "🐝", "🐢",
	"🐙", "🐬", "🐳", "🦋", "🐌", "🐞", "🦔", "🦦", "🦥", "🐱",
}

func IsEmoji(e string) bool { return slices.Contains(Emojis, e) }
