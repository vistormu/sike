// Package ui holds the templates and static files, embedded into the binary.
package ui

import (
	"embed"
	"io/fs"
)

//go:embed templates/*.tmpl templates/pages/*.tmpl templates/components/*.tmpl templates/partials/*.tmpl static
var files embed.FS

// Templates is the templates/ tree: layout.tmpl at the root, pages/,
// components/ and partials/ below.
func Templates() fs.FS { return sub("templates") }

// Static is the static/ tree, served under /static/.
func Static() fs.FS { return sub("static") }

func sub(dir string) fs.FS {
	f, err := fs.Sub(files, dir)
	if err != nil {
		panic(err) // only on a bad literal dir name
	}
	return f
}
