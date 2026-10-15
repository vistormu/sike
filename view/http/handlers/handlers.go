// Package handlers holds the http handlers, one file per screen.
package handlers

import (
	"log"
	"net/http"
)

// Renderer writes a page through the layout.
type Renderer interface {
	Render(w http.ResponseWriter, page string, data any)
	// RenderToast is Render with a status and the floating toast
	// (decisions/0007): a form that failed, kept as it was typed.
	RenderToast(w http.ResponseWriter, page string, data any, status int, t Toast)
}

// Toast is the one message component: Info, or Error. Action shows Undo.
type Toast struct {
	Message string
	Error   bool
	Action  bool
}

func serverError(w http.ResponseWriter, err error) {
	log.Printf("error: %v", err)
	http.Error(w, "internal error", http.StatusInternalServerError)
}
