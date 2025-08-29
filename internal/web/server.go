package web

import (
    "log"
    "net/http"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
)

// Server wires a chi router with minimal routes.
type Server struct {
    router chi.Router
    t      *templates
    svc    *app.Service
}

// NewServer constructs the HTTP handler with a single index route.
func NewServer(svc *app.Service) http.Handler {
    t := mustLoadTemplates()
    s := &Server{router: chi.NewRouter(), t: t, svc: svc}
    s.routes()
    return s
}

func (s *Server) routes() {
    s.router.Use(requestLogger())
    s.router.Get("/", s.handleIndex)
    s.router.Get("/game", s.handleGame)
    s.router.Post("/game", s.handleCreateGame)
    s.router.Get("/game/{id}/lobby", s.handleLobby)
    s.router.Get("/game/{id}/lobby/status", s.handleLobbyStatus)
    s.router.Post("/game/{id}/join", s.handleJoin)
    s.router.Get("/game/{id}", s.handleGameID)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}

// requestLogger logs method, path, status, bytes written, and duration.
func requestLogger() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            sr := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
            start := time.Now()
            next.ServeHTTP(sr, r)
            dur := time.Since(start)
            log.Printf("%s %s -> %d (%dB) in %s", r.Method, r.URL.Path, sr.status, sr.bytes, dur)
        })
    }
}

type statusRecorder struct {
    http.ResponseWriter
    status int
    bytes  int
}

func (sr *statusRecorder) WriteHeader(code int) {
    sr.status = code
    sr.ResponseWriter.WriteHeader(code)
}

func (sr *statusRecorder) Write(p []byte) (int, error) {
    n, err := sr.ResponseWriter.Write(p)
    sr.bytes += n
    return n, err
}
