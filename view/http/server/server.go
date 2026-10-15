package server

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"sike/domain/service"
	"sike/view/http/handlers"
	"sike/view/ui"
)

type Config struct {
	Addr  string
	Users *service.Users
}

type Server struct {
	http *http.Server
	rd   *renderer
}

// New parses every template up front: a broken template fails the boot, not a
// request.
func New(cfg Config) (*Server, error) {
	rd, err := newRenderer()
	if err != nil {
		return nil, err
	}
	s := &Server{rd: rd}

	mux := http.NewServeMux()
	mux.Handle("GET /static/", noListing(http.StripPrefix("/static/", http.FileServerFS(ui.Static()))))
	people := handlers.People{R: rd, Users: cfg.Users}
	mux.Handle("GET /{$}", http.RedirectHandler("/week", http.StatusSeeOther))
	t := handlers.Tabs{R: rd}
	for _, name := range []string{"week", "upcoming", "pending", "someday", "settings"} {
		mux.Handle("GET /"+name, people.Require(t.Tab(name)))
	}
	mux.HandleFunc("GET /who", people.Who)
	mux.HandleFunc("GET /who/new", people.New)
	mux.HandleFunc("POST /who/new", people.Create)
	mux.HandleFunc("POST /who/{id}", people.Pick)
	mux.HandleFunc("GET /dev/components", handlers.Dev{R: rd}.Components)
	mux.HandleFunc("GET /dev/forget", people.Forget)

	s.http = &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return s, nil
}

// Run serves until the context ends or the process gets SIGINT/SIGTERM, then
// gives in-flight requests a few seconds to finish.
func (s *Server) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errc := make(chan error, 1)
	go func() {
		log.Printf("listening on %s", s.http.Addr)
		if err := s.http.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
		close(errc)
	}()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.http.Shutdown(shutdownCtx)
}

// noListing 404s directory paths, so /static/ never lists its files.
func noListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}
