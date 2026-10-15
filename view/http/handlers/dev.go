package handlers

import "net/http"

// Dev serves pages for development, not for the product.
type Dev struct {
	R Renderer
}

// Components renders every component variant, for a side-by-side check with
// figma's Components page.
func (h Dev) Components(w http.ResponseWriter, r *http.Request) {
	h.R.Render(w, "dev_components", nil)
}
