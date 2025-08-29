package web

import (
    "fmt"
    "net/http"
    "strings"

    "github.com/go-chi/chi/v5"
    "github.com/jaminalder/codex-tic-tac-toe/internal/domain"
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

// handleCreateGame handles POST /game from index with selected icon.
func (s *Server) handleCreateGame(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad form", http.StatusBadRequest)
        return
    }
    icon := strings.TrimSpace(r.Form.Get("icon"))
    if icon == "" {
        http.Error(w, "icon required", http.StatusBadRequest)
        return
    }
    pid := ensurePlayerID(w, r)
    gs, err := s.svc.CreateGame()
    if err != nil {
        http.Error(w, "create failed", http.StatusInternalServerError)
        return
    }
    if _, _, err := s.svc.Join(gs.ID, pid); err != nil {
        http.Error(w, "join failed", http.StatusInternalServerError)
        return
    }
    // Set icon for X
    _ = s.svc.SetIcon(gs.ID, domain.X, icon)
    http.Redirect(w, r, fmt.Sprintf("/game/%s/lobby", gs.ID), http.StatusSeeOther)
}

// handleLobby shows waiting/share or join UI.
func (s *Server) handleLobby(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    pid := ensurePlayerID(w, r)
    st, ok := s.svc.Get(id)
    if !ok {
        http.NotFound(w, r)
        return
    }
    // self seat
    seat := ""
    if st.X == pid {
        seat = "X"
    } else if st.O == pid {
        seat = "O"
    }
    if st.X != "" && st.O != "" {
        // Both seats filled; if viewer is a player, go to game page
        http.Redirect(w, r, "/game/"+id, http.StatusSeeOther)
        return
    }
    s.t.render(w, "lobby.html.tmpl", map[string]any{
        "Title":   "Lobby",
        "ID":      id,
        "Seat":    seat,
        "HasO":    st.O != "",
        "IconX":   st.IconX,
        "IconO":   st.IconO,
        "ShareURL": absoluteURL(r, "/game/"+id+"/lobby"),
    })
}

// handleLobbyStatus polls for opponent readiness; HX-Redirect to game when ready.
func (s *Server) handleLobbyStatus(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    st, ok := s.svc.Get(id)
    if !ok {
        http.NotFound(w, r)
        return
    }
    if st.X != "" && st.O != "" {
        w.Header().Set("HX-Redirect", "/game/"+id)
        w.WriteHeader(http.StatusOK)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

// handleJoin handles second player join with icon.
func (s *Server) handleJoin(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    if err := r.ParseForm(); err != nil {
        http.Error(w, "bad form", http.StatusBadRequest)
        return
    }
    icon := strings.TrimSpace(r.Form.Get("icon"))
    if icon == "" {
        http.Error(w, "icon required", http.StatusBadRequest)
        return
    }
    pid := ensurePlayerID(w, r)
    side, _, err := s.svc.Join(id, pid)
    if err != nil {
        http.NotFound(w, r)
        return
    }
    if side == domain.O {
        _ = s.svc.SetIcon(id, domain.O, icon)
    }
    http.Redirect(w, r, "/game/"+id, http.StatusSeeOther)
}

// handleGameID renders the actual game page for /game/{id}
func (s *Server) handleGameID(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    pid := ensurePlayerID(w, r)
    st, ok := s.svc.Get(id)
    if !ok {
        http.NotFound(w, r)
        return
    }
    seat := ""
    myIcon := ""
    oppIcon := ""
    if st.X == pid {
        seat = "X"
        myIcon = st.IconX
        oppIcon = st.IconO
    } else if st.O == pid {
        seat = "O"
        myIcon = st.IconO
        oppIcon = st.IconX
    }
    s.t.render(w, "game.html.tmpl", map[string]any{
        "Title":        "Game",
        "ID":           id,
        "Seat":         seat,
        "MyIcon":       myIcon,
        "OpponentIcon": oppIcon,
    })
}

// no extra helpers
