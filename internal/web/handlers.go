package web

import (
    "errors"
    "fmt"
    "html/template"
    "io"
    "net/http"
    "strconv"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
    "github.com/jaminalder/codex-tic-tac-toe/internal/domain"
)

type handlers struct {
    svc *app.Service
    tpl *templates
}

func (h *handlers) renderBoard(gs app.GameState, errMsg string) []byte {
    data := struct {
        ID    string
        Game  struct{ Board any }
        Error string
    }{ID: gs.ID, Error: errMsg}
    data.Game.Board = gs.Game.Board
    return renderTemplate(h.tpl.board, "", data)
}

func (h *handlers) index(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write(renderTemplate(h.tpl.index, "", nil))
}

func (h *handlers) create(w http.ResponseWriter, r *http.Request) {
    gs, err := h.svc.CreateGame()
    if err != nil {
        http.Error(w, "failed to create", http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/game/"+gs.ID, http.StatusSeeOther)
}

func (h *handlers) view(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    // ensure cookie and auto-claim seat
    pid := ensurePlayerCookie(w, r)
    _, _, _ = h.svc.Join(id, pid)

    gs, ok := h.svc.Get(id)
    if !ok {
        http.NotFound(w, r)
        return
    }
    data := struct {
        ID        string
        Game      struct{ ID string }
        BoardHTML template.HTML
    }{ID: gs.ID}
    data.Game.ID = gs.ID
    data.BoardHTML = template.HTML(h.renderBoard(*gs, ""))

    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(http.StatusOK)
    // Render page with embedded board container
    _, _ = w.Write(renderTemplate(h.tpl.game, "", data))
}

func (h *handlers) join(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    pid := ensurePlayerCookie(w, r)
    _, gs, err := h.svc.Join(id, pid)
    if err != nil || gs == nil {
        http.NotFound(w, r)
        return
    }
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write(h.renderBoard(*gs, ""))
}

func (h *handlers) play(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    pid := ensurePlayerCookie(w, r)
    _ = r.ParseForm()
    rStr := r.Form.Get("r")
    cStr := r.Form.Get("c")
    ri, _ := strconv.Atoi(rStr)
    ci, _ := strconv.Atoi(cStr)
    gs, err := h.svc.Play(id, pid, ri, ci)
    var errMsg string
    if err != nil {
        if gs == nil {
            if g, ok := h.svc.Get(id); ok { gs = g }
        }
        switch {
        case errors.Is(err, app.ErrNotYourTurn):
            errMsg = "Not your turn"
        case errors.Is(err, app.ErrNotAPlayer):
            errMsg = "You are a spectator"
        case errors.Is(err, domain.ErrOccupied):
            errMsg = "Cell is occupied"
        case errors.Is(err, domain.ErrOutOfBounds):
            errMsg = "Out of bounds"
        case errors.Is(err, domain.ErrGameOver):
            errMsg = "Game is over"
        default:
            errMsg = "Invalid move"
        }
    }
    if gs == nil {
        http.NotFound(w, r)
        return
    }
    data := struct {
        ID    string
        Game  struct{ Board any }
        Error string
    }{ID: gs.ID, Error: errMsg}
    data.Game.Board = gs.Game.Board
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    _, _ = w.Write(renderTemplate(h.tpl.board, "", data))
}

var heartbeatInterval = 15 * time.Second

func (h *handlers) events(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("X-Accel-Buffering", "no")
    // In tests or non-EventSource requests, just acknowledge headers and return
    if r.Header.Get("Accept") != "text/event-stream" {
        w.WriteHeader(http.StatusOK)
        return
    }
    flusher, ok := w.(http.Flusher)
    if !ok {
        w.WriteHeader(http.StatusOK)
        return
    }
    ctx := r.Context()
    ch, _ := h.svc.Subscribe(ctx, id)
    // heartbeat ticker
    ticker := time.NewTicker(heartbeatInterval)
    defer ticker.Stop()
    // Initial flush of headers
    flusher.Flush()
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            _, _ = io.WriteString(w, ": ping\n\n")
            flusher.Flush()
        case b, ok := <-ch:
            if !ok { return }
            // Emit board event
            _, _ = fmt.Fprintf(w, "event: board\n")
            _, _ = fmt.Fprintf(w, "data: %s\n\n", b)
            flusher.Flush()
        }
    }
}
