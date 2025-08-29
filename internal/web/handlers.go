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
