package web

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
)

// NewServer wires routes and returns an http.Handler.
func NewServer(s *app.Service) http.Handler {
    r := chi.NewRouter()
    h := &handlers{svc: s, tpl: loadTemplates()}
    r.Get("/", h.index)
    r.Post("/game", h.create)
    r.Route("/game/{id}", func(r chi.Router) {
        r.Get("/", h.view)
        r.Post("/join", h.join)
        r.Post("/play", h.play)
        r.Get("/events", h.events)
    })
    return r
}

