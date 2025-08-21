package web

import (
    "bytes"
    "context"
    "io"
    "net/http"
    "net/http/httptest"
    "net/url"
    "strings"
    "testing"
    "time"

    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
    "github.com/go-chi/chi/v5"
)

func newTestServer(t *testing.T) (*app.Service, http.Handler) {
    t.Helper()
    // Use a simple renderer so SSE emits a predictable payload
    s := app.NewServiceWithRenderer(func(gs app.GameState) []byte { return []byte("board") })
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

// flushRecorder is a ResponseWriter that supports Flusher and captures writes.
type flushRecorder struct {
    header http.Header
    code   int
    buf    bytes.Buffer
}

func (f *flushRecorder) Header() http.Header         { return f.header }
func (f *flushRecorder) WriteHeader(code int)        { f.code = code }
func (f *flushRecorder) Write(p []byte) (int, error) { return f.buf.Write(p) }
func (f *flushRecorder) Flush()                      {}

func TestEventsBroadcastsBoardOnPlay(t *testing.T) {
    svc, _ := newTestServer(t)
    // Build handlers directly to call events method
    h := &handlers{svc: svc, tpl: loadTemplates()}
    gs, _ := svc.CreateGame()
    svc.Join(gs.ID, "p1")
    svc.Join(gs.ID, "p2")

    // Prepare request with route param and Accept header
    req := httptest.NewRequest("GET", "/game/"+gs.ID+"/events", nil)
    rc := chi.NewRouteContext()
    rc.URLParams.Add("id", gs.ID)
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
    req.Header.Set("Accept", "text/event-stream")

    rw := &flushRecorder{header: make(http.Header)}

    // Run handler in goroutine and trigger a move shortly after
    done := make(chan struct{})
    go func() {
        defer close(done)
        h.events(rw, req)
    }()

    // Small delay to allow subscription to register
    time.Sleep(20 * time.Millisecond)
    // Trigger a move to cause a broadcast
    if _, err := svc.Play(gs.ID, "p1", 0, 0); err != nil {
        t.Fatalf("play failed: %v", err)
    }

    // Poll buffer for an event
    deadline := time.Now().Add(2 * time.Second)
    for time.Now().Before(deadline) {
        if strings.Contains(rw.buf.String(), "event: board") && strings.Contains(rw.buf.String(), "data: board") {
            break
        }
        time.Sleep(10 * time.Millisecond)
    }
    if !strings.Contains(rw.buf.String(), "event: board") {
        t.Fatalf("expected board event, got: %q", rw.buf.String())
    }
}

func TestEventsHeartbeat(t *testing.T) {
    svc, _ := newTestServer(t)
    h := &handlers{svc: svc, tpl: loadTemplates()}
    gs, _ := svc.CreateGame()
    req := httptest.NewRequest("GET", "/game/"+gs.ID+"/events", nil)
    rc := chi.NewRouteContext()
    rc.URLParams.Add("id", gs.ID)
    req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
    req.Header.Set("Accept", "text/event-stream")
    rw := &flushRecorder{header: make(http.Header)}

    // Speed up heartbeats for test
    old := heartbeatInterval
    heartbeatInterval = 20 * time.Millisecond
    defer func() { heartbeatInterval = old }()

    // Run handler and cancel after first ping observed
    ctx, cancel := context.WithCancel(req.Context())
    req = req.WithContext(ctx)
    go h.events(rw, req)

    deadline := time.Now().Add(500 * time.Millisecond)
    for time.Now().Before(deadline) {
        if strings.Contains(rw.buf.String(), ": ping") {
            break
        }
        time.Sleep(10 * time.Millisecond)
    }
    cancel()
    if !strings.Contains(rw.buf.String(), ": ping") {
        t.Fatalf("expected heartbeat ping, got: %q", rw.buf.String())
    }
}

func TestPlayEndpointRendersErrorAlert(t *testing.T) {
    svc, h := newTestServer(t)
    gs, _ := svc.CreateGame()
    // Assign both seats
    svc.Join(gs.ID, "p1")
    svc.Join(gs.ID, "p2")
    // O tries to play first (not your turn)
    form := url.Values{"r": {"0"}, "c": {"0"}, "side": {"O"}}
    req := httptest.NewRequest("POST", "/game/"+gs.ID+"/play", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    req.AddCookie(&http.Cookie{Name: "player_id", Value: "p2"})
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rr.Code)
    }
    if !strings.Contains(strings.ToLower(rr.Body.String()), "not your turn") {
        t.Fatalf("expected inline error alert, got: %q", rr.Body.String())
    }
}
