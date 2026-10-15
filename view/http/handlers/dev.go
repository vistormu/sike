// Package handlers holds the http handlers, one file per screen.
package handlers

import "net/http"

// Renderer writes a page through the layout.
type Renderer interface {
	Render(w http.ResponseWriter, page string, data any)
}

// Dev serves pages for development, not for the product.
type Dev struct {
	R Renderer
}

// Components renders every component variant, for a side-by-side check with
// figma's Components page.
func (h Dev) Components(w http.ResponseWriter, r *http.Request) {
	h.R.Render(w, "dev_components", nil)
}
