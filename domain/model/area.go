package model

import "slices"

// Area is one of a person's seven colors, with the name they gave it. the
// colors are the tag system: one task, one area (decisions/0005).
type Area struct {
	UserID int64  `db:"user_id"`
	Color  string `db:"color"`
	Name   string `db:"name"`
}

// Colors are the seven task colors, in palette order (figma's task/*).
var Colors = []string{"lavender", "sky", "apricot", "coral", "mint", "lime", "pink"}

func IsColor(c string) bool { return slices.Contains(Colors, c) }

// DefaultAreas are every new person's areas, renamed in settings.
var DefaultAreas = []Area{
	{Color: "lavender", Name: "Self"},
	{Color: "sky", Name: "Work"},
	{Color: "apricot", Name: "Body"},
	{Color: "coral", Name: "Admin"},
	{Color: "mint", Name: "Home"},
	{Color: "lime", Name: "Social"},
	{Color: "pink", Name: "Fun"},
}
