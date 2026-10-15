package main

import (
	_ "time/tzdata" // SIKE_TZ works on an image with no os tz database

	"sike/cmd"
)

func main() {
	cmd.Execute()
}
