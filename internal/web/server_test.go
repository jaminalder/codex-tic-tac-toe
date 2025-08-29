package web

import (
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"

    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
)

func TestIndexRouteReturnsHTML(t *testing.T) {
    t.Parallel()
    h := NewServer(app.NewService())
    req := httptest.NewRequest(http.MethodGet, "/", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    ct := rr.Header().Get("Content-Type")
    if !strings.HasPrefix(ct, "text/html") {
        t.Fatalf("expected Content-Type text/html, got %q", ct)
    }
    body := rr.Body.String()
    if !strings.Contains(body, "<h1 class=\"title\">Tic-Tac-Toe</h1>") {
        t.Fatalf("expected index content to contain title, got body: %q", body)
    }
}

func TestGameRouteReturnsHTML(t *testing.T) {
    t.Parallel()
    h := NewServer(app.NewService())
    req := httptest.NewRequest(http.MethodGet, "/game", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)

    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    ct := rr.Header().Get("Content-Type")
    if !strings.HasPrefix(ct, "text/html") {
        t.Fatalf("expected Content-Type text/html, got %q", ct)
    }
    body := rr.Body.String()
    if !strings.Contains(body, "<h1 class=\"title\">Game</h1>") {
        t.Fatalf("expected game page header not found; body: %q", body)
    }
}
