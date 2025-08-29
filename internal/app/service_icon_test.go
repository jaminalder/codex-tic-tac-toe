package app

import (
    "testing"

    "github.com/jaminalder/codex-tic-tac-toe/internal/domain"
)

func TestSetIconSetsValues(t *testing.T) {
    s := NewService()
    gs, err := s.CreateGame()
    if err != nil { t.Fatal(err) }

    if err := s.SetIcon(gs.ID, domain.X, "😺"); err != nil { t.Fatal(err) }
    if err := s.SetIcon(gs.ID, domain.O, "🐶"); err != nil { t.Fatal(err) }

    got, ok := s.Get(gs.ID)
    if !ok { t.Fatalf("game not found after SetIcon") }
    if got.IconX != "😺" || got.IconO != "🐶" {
        t.Fatalf("unexpected icons: X=%q O=%q", got.IconX, got.IconO)
    }
}

func TestSetIconNotFound(t *testing.T) {
    s := NewService()
    if err := s.SetIcon("missing", domain.X, "😺"); err == nil {
        t.Fatalf("expected error for missing game id")
    }
}

