package server

import (
	"bytes"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"sike/view/ui"
)

// renderer holds one template set per page: the layout and components cloned,
// plus that page's {{define "content"}}. Sets are independent, so every page
// can define "content" without colliding with the others.
type renderer struct {
	pages map[string]*template.Template
}

func newRenderer() (*renderer, error) {
	tmplFS := ui.Templates()
	base, err := template.New("").Funcs(funcMap()).ParseFS(tmplFS, "layout.tmpl", "components/*.tmpl")
	if err != nil {
		return nil, err
	}

	files, err := fs.Glob(tmplFS, "pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	pages := make(map[string]*template.Template, len(files))
	for _, file := range files {
		set, err := base.Clone()
		if err != nil {
			return nil, err
		}
		if _, err := set.ParseFS(tmplFS, file); err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(path.Base(file), ".tmpl")
		if set.Lookup("content") == nil {
			return nil, fmt.Errorf("page %q does not define \"content\"", name)
		}
		pages[name] = set
	}
	return &renderer{pages: pages}, nil
}

// Render writes a page through the layout. It renders into a buffer first, so a
// template error is a clean 500 instead of half a page.
func (rd *renderer) Render(w http.ResponseWriter, page string, data any) {
	set, ok := rd.pages[page]
	if !ok {
		log.Printf("render: unknown page %q", page)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := set.ExecuteTemplate(&buf, "layout", data); err != nil {
		log.Printf("render %s: %v", page, err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"dict": dict, // (dict "Name" .Name "Color" .Color): props for a component
	}
}

func dict(values ...any) (map[string]any, error) {
	if len(values)%2 != 0 {
		return nil, fmt.Errorf("dict expects an even number of args, got %d", len(values))
	}
	m := make(map[string]any, len(values)/2)
	for i := 0; i < len(values); i += 2 {
		key, ok := values[i].(string)
		if !ok {
			return nil, fmt.Errorf("dict keys must be strings, got %T", values[i])
		}
		m[key] = values[i+1]
	}
	return m, nil
}
