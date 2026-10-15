package templates

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"sike/view/ui"
)

// the go↔template contract. template names are strings, so the compiler sees
// none of it: a typo is a 500 at runtime, and a template nobody calls just sits
// there. this test reads both sides and fails on either.

var (
	defineRe = regexp.MustCompile(`\{\{-?\s*define\s+"([^"]+)"`)
	callRe   = regexp.MustCompile(`\{\{-?\s*(?:template|block)\s+"([^"]+)"`)
	// a shared template named in go code: "layout", "components/x", "partials/y"
	goNameRe = regexp.MustCompile(`"(layout|(?:components|partials)/[A-Za-z0-9_-]+)"`)
	// a page rendered from go code: Render(w, "home", …)
	goPageRe = regexp.MustCompile(`\bRender\(\s*w,\s*"([A-Za-z0-9_-]+)"`)
)

// goSourceDirs are where go code names templates, relative to this package.
var goSourceDirs = []string{"../../http", "../../../cmd"}

type define struct{ name, file string }

func templateFiles(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := fs.WalkDir(ui.Templates(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(ui.Templates(), p)
		if err != nil {
			return err
		}
		out[p] = string(b)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// goSource scans non-test go code: the shared template names it uses and the
// pages it renders.
func goSource(t *testing.T) (names, pages map[string]string) {
	t.Helper()
	names, pages = map[string]string{}, map[string]string{}
	files := 0
	for _, dir := range goSourceDirs {
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			files++
			for _, m := range goNameRe.FindAllStringSubmatch(string(b), -1) {
				names[m[1]] = p
			}
			for _, m := range goPageRe.FindAllStringSubmatch(string(b), -1) {
				pages[m[1]] = p
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if files < 3 {
		t.Fatalf("only %d go files scanned: the walk is not reaching the handlers", files)
	}
	return names, pages
}

// every shared template is reached, every name referenced is defined.
func TestTemplatesHaveNoOrphansOrDanglingNames(t *testing.T) {
	var defines []define
	reached := map[string]bool{}
	for file, src := range templateFiles(t) {
		for _, m := range defineRe.FindAllStringSubmatch(src, -1) {
			defines = append(defines, define{m[1], file})
		}
		for _, m := range callRe.FindAllStringSubmatch(src, -1) {
			reached[m[1]] = true
		}
	}
	names, _ := goSource(t)
	for name := range names {
		reached[name] = true
	}

	defined := map[string]bool{}
	var orphans, dangling []string
	for _, d := range defines {
		defined[d.name] = true
		// a page's "content" is called by the layout of its own set
		if strings.HasPrefix(d.file, "pages/") {
			continue
		}
		if !reached[d.name] {
			orphans = append(orphans, d.name+" ("+d.file+")")
		}
	}
	for name := range reached {
		if !defined[name] {
			dangling = append(dangling, name)
		}
	}
	sort.Strings(orphans)
	sort.Strings(dangling)
	for _, o := range orphans {
		t.Errorf("template %s is defined but nothing calls it: delete it, or call it", o)
	}
	for _, d := range dangling {
		t.Errorf("template %q is referenced but never defined", d)
	}
}

// every page file is rendered by some handler, and every rendered page exists.
func TestPagesMatchHandlers(t *testing.T) {
	files := map[string]bool{}
	for file := range templateFiles(t) {
		if dir, _ := path.Split(file); dir == "pages/" {
			files[strings.TrimSuffix(path.Base(file), ".tmpl")] = true
		}
	}
	_, rendered := goSource(t)

	for page := range files {
		if _, ok := rendered[page]; !ok {
			t.Errorf("pages/%s.tmpl is never rendered: delete it, or render it", page)
		}
	}
	for page, where := range rendered {
		if !files[page] {
			t.Errorf("%s renders page %q, but pages/%s.tmpl does not exist", where, page, page)
		}
	}
}

// a page defines only "content"; anything shared is a component or partial.
// a shared define lives in the directory its name says.
func TestDefinesLiveWhereTheirNameSays(t *testing.T) {
	for file, src := range templateFiles(t) {
		dir, _ := path.Split(file)
		for _, m := range defineRe.FindAllStringSubmatch(src, -1) {
			name := m[1]
			switch {
			case dir == "pages/":
				if name != "content" {
					t.Errorf("%s defines %q: a page defines only \"content\"", file, name)
				}
			case dir == "":
				if name != "layout" {
					t.Errorf("%s defines %q: only the layout lives at the root", file, name)
				}
			case !strings.HasPrefix(name, dir):
				t.Errorf("%s defines %q: a %s file must define %s… names", file, name, dir, dir)
			}
		}
	}
}
