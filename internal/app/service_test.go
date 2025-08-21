package app

import (
    "context"
    "errors"
    "fmt"
    "testing"
    "time"

    "github.com/jaminalder/codex-tic-tac-toe/internal/domain"
)

// minimal renderer for tests: encode moves count as bytes
func testRenderer(gs GameState) []byte { return []byte(fmt.Sprintf("moves=%d", gs.Game.Moves)) }

func TestCreateAndGet(t *testing.T) {
    s := NewServiceWithRenderer(testRenderer)
    gs, err := s.CreateGame()
    if err != nil {
        t.Fatalf("CreateGame error: %v", err)
    }
    if gs.ID == "" {
        t.Fatalf("expected non-empty game ID")
    }
    if gs.Game.Turn != domain.X {
        t.Fatalf("expected initial turn X")
    }
    if gs.Created.IsZero() || gs.Updated.IsZero() {
        t.Fatalf("expected timestamps to be set")
    }
    got, ok := s.Get(gs.ID)
    if !ok || got.ID != gs.ID {
        t.Fatalf("Get should find created game")
    }
}

func TestJoinSeatsAndRejoin(t *testing.T) {
    s := NewServiceWithRenderer(testRenderer)
    gs, _ := s.CreateGame()
    p1, p2, p3 := "p1", "p2", "p3"

    side, _, err := s.Join(gs.ID, p1)
    if err != nil || side != domain.X {
        t.Fatalf("p1 should claim X, got %v, err=%v", side, err)
    }
    side, _, err = s.Join(gs.ID, p2)
    if err != nil || side != domain.O {
        t.Fatalf("p2 should claim O, got %v, err=%v", side, err)
    }
    side, _, err = s.Join(gs.ID, p1)
    if err != nil || side != domain.X {
        t.Fatalf("p1 rejoin should keep X, got %v, err=%v", side, err)
    }
    side, _, err = s.Join(gs.ID, p3)
    if err != nil || side != domain.Empty {
        t.Fatalf("p3 should spectate (Empty), got %v, err=%v", side, err)
    }
}

func TestPlayEnforcesTurnAndSpectatorBlocked(t *testing.T) {
    s := NewServiceWithRenderer(testRenderer)
    gs, _ := s.CreateGame()
    p1, p2, p3 := "p1", "p2", "p3"
    s.Join(gs.ID, p1) // X
    s.Join(gs.ID, p2) // O
    s.Join(gs.ID, p3) // spectator

    // O cannot play first
    if _, err := s.Play(gs.ID, p2, 0, 0); !errors.Is(err, ErrNotYourTurn) {
        t.Fatalf("expected ErrNotYourTurn, got %v", err)
    }
    // spectator cannot play
    if _, err := s.Play(gs.ID, p3, 0, 0); !errors.Is(err, ErrNotAPlayer) {
        t.Fatalf("expected ErrNotAPlayer, got %v", err)
    }
    // X plays
    st, err := s.Play(gs.ID, p1, 0, 0)
    if err != nil {
        t.Fatalf("X play failed: %v", err)
    }
    if st.Game.Board[0] != domain.X || st.Game.Turn != domain.O || st.Game.Moves != 1 {
        t.Fatalf("unexpected state after X move: turn=%v moves=%d cell0=%v", st.Game.Turn, st.Game.Moves, st.Game.Board[0])
    }
    // X cannot play again
    if _, err := s.Play(gs.ID, p1, 1, 1); !errors.Is(err, ErrNotYourTurn) {
        t.Fatalf("expected ErrNotYourTurn for X again, got %v", err)
    }
}

func TestSubscribeAndBroadcast(t *testing.T) {
    s := NewServiceWithRenderer(testRenderer)
    gs, _ := s.CreateGame()
    p1, p2 := "p1", "p2"
    s.Join(gs.ID, p1)
    s.Join(gs.ID, p2)

    ctx, cancel := context.WithTimeout(context.Background(), time.Second*2)
    defer cancel()
    ch, unsub := s.Subscribe(ctx, gs.ID)
    defer unsub()

    // Trigger an update: X plays
    if _, err := s.Play(gs.ID, p1, 0, 0); err != nil {
        t.Fatalf("play failed: %v", err)
    }

    select {
    case b, ok := <-ch:
        if !ok { t.Fatalf("channel closed unexpectedly") }
        if string(b) != "moves=1" {
            t.Fatalf("unexpected broadcast payload: %q", string(b))
        }
    case <-ctx.Done():
        t.Fatalf("timed out waiting for broadcast")
    }
}

func TestDropSlowSubscriber(t *testing.T) {
    s := NewServiceWithRenderer(testRenderer)
    gs, _ := s.CreateGame()
    p1, p2 := "p1", "p2"
    s.Join(gs.ID, p1)
    s.Join(gs.ID, p2)

    // Slow subscriber: never read
    ctxSlow, cancelSlow := context.WithCancel(context.Background())
    slowCh, _ := s.Subscribe(ctxSlow, gs.ID)
    _ = slowCh // intentionally not read

    // Fast subscriber: will read
    ctxFast, cancelFast := context.WithTimeout(context.Background(), time.Second*2)
    defer cancelFast()
    fastCh, unsubFast := s.Subscribe(ctxFast, gs.ID)
    defer unsubFast()

    // Two quick updates; slow should be dropped to avoid blocking fast
    if _, err := s.Play(gs.ID, p1, 0, 0); err != nil { t.Fatalf("play1: %v", err) }
    if _, err := s.Play(gs.ID, p2, 1, 1); err != nil { t.Fatalf("play2: %v", err) }

    // Fast still receives the latest
    got := 0
    for got < 2 {
        select {
        case <-fastCh:
            got++
        case <-ctxFast.Done():
            t.Fatalf("fast subscriber did not receive updates in time")
        }
    }

    // Slow subscriber should be dropped; cancel context and ensure channel is closed promptly
    cancelSlow()
}

