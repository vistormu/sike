package model

// When is where a task lives in time, and so which tab shows it
// (decisions/0005). exactly one per task.
type When string

const (
	Weekly  When = "weekly"  // placements on weekdays (m15)
	Dated   When = "date"    // a date, a deadline (m18)
	Pending When = "pending" // owed, no date
	Someday When = "someday" // a wish
)

// Whens in tab order: week, upcoming, pending, someday.
var Whens = []When{Weekly, Dated, Pending, Someday}

// Task is a thing to do, in one area (its color) and one place in time.
type Task struct {
	ID       int64  `db:"id"`
	UserID   int64  `db:"user_id"`
	Name     string `db:"name"`
	Color    string `db:"color"`
	When     When   `db:"when"`
	Position int64  `db:"position"`
}
