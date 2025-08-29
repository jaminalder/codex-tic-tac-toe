package web

import (
    "io"
    "net/http"
    "net/http/httptest"
    "net/url"
    "regexp"
    "strings"
    "testing"

    "github.com/jaminalder/codex-tic-tac-toe/internal/app"
)

func TestInitFlow_CreateLobby_Join_RedirectToGame(t *testing.T) {
    h := NewServer(app.NewService())

    // Step 1: creator posts /game with icon
    form := url.Values{"icon": {"😺"}}
    req := httptest.NewRequest(http.MethodPost, "/game", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusSeeOther && rr.Code != http.StatusFound {
        t.Fatalf("create: expected redirect, got %d", rr.Code)
    }
    loc := rr.Header().Get("Location")
    if !strings.Contains(loc, "/lobby") {
        t.Fatalf("create: expected redirect to lobby, got %q", loc)
    }
    // capture player1 cookie
    var p1Cookie *http.Cookie
    for _, c := range rr.Result().Cookies() {
        if c.Name == playerCookie { p1Cookie = c; break }
    }
    if p1Cookie == nil { t.Fatalf("expected player_id cookie for creator") }

    // extract id from /game/{id}/lobby
    re := regexp.MustCompile(`/game/([^/]+)/lobby`)
    m := re.FindStringSubmatch(loc)
    if len(m) != 2 { t.Fatalf("could not extract game id from %q", loc) }
    id := m[1]

    // Step 2: GET lobby as p1, should show waiting and icon X
    req2 := httptest.NewRequest(http.MethodGet, "/game/"+id+"/lobby", nil)
    req2.AddCookie(p1Cookie)
    rr2 := httptest.NewRecorder()
    h.ServeHTTP(rr2, req2)
    if rr2.Code != http.StatusOK { t.Fatalf("lobby: expected 200, got %d", rr2.Code) }
    body2 := rr2.Body.String()
    if !strings.Contains(body2, "Waiting for opponent") && !strings.Contains(body2, "hx-get=\"/game/"+id+"/lobby/status\"") {
        t.Fatalf("lobby: expected waiting indicator, got: %q", body2)
    }
    if !strings.Contains(body2, "Icons — X: 😺") {
        t.Fatalf("lobby: expected creator icon shown, got: %q", body2)
    }
    // Share URL should be absolute with host and include /lobby
    expectShare := "http://" + req2.Host + "/game/" + id + "/lobby"
    if !strings.Contains(body2, expectShare) {
        t.Fatalf("lobby: expected absolute share url %q, got body: %q", expectShare, body2)
    }

    // Step 3: lobby status should be 204 while waiting
    req3 := httptest.NewRequest(http.MethodGet, "/game/"+id+"/lobby/status", nil)
    req3.AddCookie(p1Cookie)
    rr3 := httptest.NewRecorder()
    h.ServeHTTP(rr3, req3)
    if rr3.Code != http.StatusNoContent { t.Fatalf("status: expected 204 while waiting, got %d", rr3.Code) }

    // Step 4: second player joins with icon 🐶
    formJ := url.Values{"icon": {"🐶"}}
    reqJ := httptest.NewRequest(http.MethodPost, "/game/"+id+"/join", strings.NewReader(formJ.Encode()))
    reqJ.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    rrJ := httptest.NewRecorder()
    h.ServeHTTP(rrJ, reqJ)
    if rrJ.Code != http.StatusSeeOther && rrJ.Code != http.StatusFound {
        t.Fatalf("join: expected redirect to game, got %d", rrJ.Code)
    }

    // Step 5: lobby status should now send HX-Redirect
    req4 := httptest.NewRequest(http.MethodGet, "/game/"+id+"/lobby/status", nil)
    req4.AddCookie(p1Cookie)
    rr4 := httptest.NewRecorder()
    h.ServeHTTP(rr4, req4)
    if rr4.Code != http.StatusOK { t.Fatalf("status after join: expected 200, got %d", rr4.Code) }
    if rr4.Header().Get("HX-Redirect") != "/game/"+id {
        t.Fatalf("status after join: expected HX-Redirect to game, got %q", rr4.Header().Get("HX-Redirect"))
    }

    // Step 6: GET game page as p1, should have my icon in data attribute
    reqG1 := httptest.NewRequest(http.MethodGet, "/game/"+id, nil)
    reqG1.AddCookie(p1Cookie)
    rrG1 := httptest.NewRecorder()
    h.ServeHTTP(rrG1, reqG1)
    if rrG1.Code != http.StatusOK { t.Fatalf("game (p1): expected 200, got %d", rrG1.Code) }
    bodyG1 := rrG1.Body.String()
    if !strings.Contains(bodyG1, "data-my-icon=\"😺\"") {
        t.Fatalf("game (p1): expected my icon 😺 in data attribute, got body: %q", bodyG1)
    }

    // Step 7: GET game page as p2, ensure p2 cookie from join response or generate by calling ensure again
    // Extract p2 cookie from join response (if present)
    var p2Cookie *http.Cookie
    for _, c := range rrJ.Result().Cookies() {
        if c.Name == playerCookie { p2Cookie = c; break }
    }
    if p2Cookie == nil {
        // If not set due to test server behavior, simulate a new cookie value
        // by calling /game/{id} without cookie to trigger ensurePlayerID
        reqInit := httptest.NewRequest(http.MethodGet, "/game/"+id, nil)
        rrInit := httptest.NewRecorder()
        h.ServeHTTP(rrInit, reqInit)
        for _, c := range rrInit.Result().Cookies() {
            if c.Name == playerCookie { p2Cookie = c; break }
        }
        // We don't strictly need p2 check beyond ensuring handler serves the page
        _, _ = io.ReadAll(rrInit.Result().Body)
    }
    reqG2 := httptest.NewRequest(http.MethodGet, "/game/"+id, nil)
    if p2Cookie != nil { reqG2.AddCookie(p2Cookie) }
    rrG2 := httptest.NewRecorder()
    h.ServeHTTP(rrG2, reqG2)
    if rrG2.Code != http.StatusOK { t.Fatalf("game (p2): expected 200, got %d", rrG2.Code) }
    bodyG2 := rrG2.Body.String()
    if !strings.Contains(bodyG2, "data-my-icon=\"🐶\"") {
        t.Fatalf("game (p2): expected my icon 🐶 in data attribute, got body: %q", bodyG2)
    }
}

func TestLobbyShowsJoinFormForNonSeatedUser(t *testing.T) {
    svc := app.NewService()
    h := NewServer(svc)

    // Creator makes a game and is X
    form := url.Values{"icon": {"😺"}}
    req := httptest.NewRequest(http.MethodPost, "/game", strings.NewReader(form.Encode()))
    req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
    rr := httptest.NewRecorder()
    h.ServeHTTP(rr, req)
    if rr.Code != http.StatusSeeOther && rr.Code != http.StatusFound {
        t.Fatalf("create: expected redirect, got %d", rr.Code)
    }
    loc := rr.Header().Get("Location")
    re := regexp.MustCompile(`/game/([^/]+)/lobby`)
    m := re.FindStringSubmatch(loc)
    if len(m) != 2 { t.Fatalf("could not extract game id from %q", loc) }
    id := m[1]

    // Before visiting as second user, ensure O is empty
    st, ok := svc.Get(id)
    if !ok { t.Fatalf("game not found") }
    if st.O != "" { t.Fatalf("expected O seat empty before second user visits, got %q", st.O) }

    // Second user visits lobby without cookie: should see join form and not claim seat yet
    req2 := httptest.NewRequest(http.MethodGet, "/game/"+id+"/lobby", nil)
    rr2 := httptest.NewRecorder()
    h.ServeHTTP(rr2, req2)
    if rr2.Code != http.StatusOK { t.Fatalf("lobby (p2): expected 200, got %d", rr2.Code) }
    body := rr2.Body.String()
    if !strings.Contains(body, "/game/"+id+"/join") || !strings.Contains(body, "Join Game") {
        t.Fatalf("lobby (p2): expected join form, got body: %q", body)
    }
    // Ensure seat O still empty after GET lobby
    st2, ok := svc.Get(id)
    if !ok { t.Fatalf("game not found") }
    if st2.O != "" { t.Fatalf("expected O seat to remain empty after lobby GET, got %q", st2.O) }
}
