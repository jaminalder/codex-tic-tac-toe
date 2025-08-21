package web

import (
    "io"
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"

    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
)

func newTestServer(t *testing.T) (*app.Service, http.Handler) {
    t.Helper()
    s := app.NewService()
    h := NewServer(s)
    return s, h
}

func TestIndexPage(t *testing.T) {
    _, h := newTestServer(t)
    req := httptest.NewRequest("GET", "/", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    body := rr.Body.String()
    if !strings.Contains(body, "<form") || !strings.Contains(body, "action=\"/game\"") {
        t.Fatalf("index should contain create form; got body: %q", body)
    }
}

func TestCreateRedirectsToGame(t *testing.T) {
    _, h := newTestServer(t)
    req := httptest.NewRequest("POST", "/game", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusSeeOther && rr.Code != http.StatusFound {
        t.Fatalf("expected redirect, got %d", rr.Code)
    }
    loc := rr.Result().Header.Get("Location")
    if !strings.HasPrefix(loc, "/game/") {
        t.Fatalf("expected redirect to /game/{id}, got %q", loc)
    }
}

func TestGamePageSetsCookieAndAutoClaims(t *testing.T) {
    svc, h := newTestServer(t)
    // Create a game via service to know ID
    gs, _ := svc.CreateGame()

    req := httptest.NewRequest("GET", "/game/"+url.PathEscape(gs.ID), nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    // Cookie set
    cookies := rr.Result().Cookies()
    var playerID string
    for _, c := range cookies {
        if c.Name == "player_id" {
            playerID = c.Value
            break
        }
    }
    if playerID == "" {
        t.Fatalf("expected player_id cookie to be set")
    }
    // Auto-claimed seat
    latest, ok := svc.Get(gs.ID)
    if !ok || (latest.X != playerID && latest.O != playerID) {
        t.Fatalf("expected auto-claim X or O; have X=%q O=%q pid=%q", latest.X, latest.O, playerID)
    }
    // SSE wiring present
    body := rr.Body.String()
    if !strings.Contains(body, "hx-ext=\"sse\"") || !strings.Contains(body, "/game/"+gs.ID+"/events") {
        t.Fatalf("expected SSE wiring in page; got body: %q", body)
    }
}

func TestJoinEndpointReturnsBoardFragment(t *testing.T) {
    svc, h := newTestServer(t)
    gs, _ := svc.CreateGame()
    // First GET to auto-claim X for p1
    req1 := httptest.NewRequest("GET", "/game/"+gs.ID, nil)
    rr1 := httptest.NewRecorder()
    h.ServeHTTP(rr1, req1)
    // Extract cookie for second player
    p2 := &http.Cookie{Name: "player_id", Value: "p2"}
    form := url.Values{}
    req := httptest.NewRequest("POST", "/game/"+gs.ID+"/join", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(p2)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    if !strings.Contains(rr.Body.String(), "id=\"board\"") {
        t.Fatalf("expected board fragment, got %q", rr.Body.String())
    }
    latest, _ := svc.Get(gs.ID)
    if latest.O != "p2" && latest.X != "p2" { // allow if X was free
        t.Fatalf("expected seat for p2, got X=%q O=%q", latest.X, latest.O)
    }
}

func TestPlayEndpointUpdatesStateAndReturnsFragment(t *testing.T) {
    svc, h := newTestServer(t)
    gs, _ := svc.CreateGame()
    // Assign X and O
    svc.Join(gs.ID, "p1")
    svc.Join(gs.ID, "p2")

    form := url.Values{"r": {"0"}, "c": {"0"}, "side": {"X"}}
    req := httptest.NewRequest("POST", "/game/"+gs.ID+"/play", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{Name: "player_id", Value: "p1"})
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    if !strings.Contains(rr.Body.String(), "id=\"board\"") {
        t.Fatalf("expected board fragment, got %q", rr.Body.String())
    }
    latest, _ := svc.Get(gs.ID)
    if latest.Game.Moves != 1 {
        t.Fatalf("expected move applied, moves=%d", latest.Game.Moves)
    }
}

func TestEventsEndpointSSEHeaders(t *testing.T) {
    _, h := newTestServer(t)
    // create a game via POST
    reqCreate := httptest.NewRequest("POST", "/game", nil)
    rrCreate := httptest.NewRecorder()
    h.ServeHTTP(rrCreate, reqCreate)
    loc := rrCreate.Result().Header.Get("Location")
    if loc == "" {
        t.Fatalf("missing redirect location")
    }
    // Request SSE
    req := httptest.NewRequest("GET", loc+"/events", nil)
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    ct := rr.Result().Header.Get("Content-Type")
    if !strings.HasPrefix(ct, "text/event-stream") {
        io.Copy(io.Discard, rr.Result().Body)
        t.Fatalf("expected text/event-stream, got %q", ct)
    }
}

