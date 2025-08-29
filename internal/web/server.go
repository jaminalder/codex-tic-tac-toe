package web

import (
    "net/http"

    "github.com/go-chi/chi/v5"
)

// Server wires a chi router with minimal routes.
type Server struct {
    router chi.Router
    t      *templates
}

// NewServer constructs the HTTP handler with a single index route.
func NewServer() http.Handler {
    t := mustLoadTemplates()
    s := &Server{router: chi.NewRouter(), t: t}
    s.routes()
    return s
}

func (s *Server) routes() {
    s.router.Get("/", s.handleIndex)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}
