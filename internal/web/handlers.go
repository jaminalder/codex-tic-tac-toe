package web

import (
    "net/http"
)

// handleIndex renders the landing page.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
    s.t.render(w, "index.html.tmpl", map[string]any{
        "Title": "Tic-Tac-Toe",
    })
}

// handleGame renders a static game page (no server logic yet).
func (s *Server) handleGame(w http.ResponseWriter, r *http.Request) {
    s.t.render(w, "game.html.tmpl", map[string]any{
        "Title": "Game",
    })
}
