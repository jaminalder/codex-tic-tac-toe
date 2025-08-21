package domain

import (
    "testing"
)

// helper to apply a sequence of moves
func playMoves(t *testing.T, g *Game, moves [][2]int) {
    t.Helper()
    for i, m := range moves {
        if err := g.Play(m[0], m[1]); err != nil {
            t.Fatalf("move %d (%v) failed: %v", i, m, err)
        }
    }
}

func TestNewGameInitialState(t *testing.T) {
    g := New()
    if g.Turn != X {
        t.Fatalf("expected initial turn X, got %v", g.Turn)
    }
    if g.Moves != 0 {
        t.Fatalf("expected 0 moves, got %d", g.Moves)
    }
    if g.Over {
        t.Fatalf("expected game not over")
    }
    if g.Winner != Empty {
        t.Fatalf("expected no winner, got %v", g.Winner)
    }
    for i, c := range g.Board {
        if c != Empty {
            t.Fatalf("expected empty board, cell %d = %v", i, c)
        }
    }
}

func TestPlayOutOfBounds(t *testing.T) {
    g := New()
    cases := [][2]int{{-1, 0}, {0, -1}, {3, 0}, {0, 3}, {5, 5}}
    for _, m := range cases {
        if err := g.Play(m[0], m[1]); err == nil || err != ErrOutOfBounds {
            t.Fatalf("expected ErrOutOfBounds for %v, got %v", m, err)
        }
    }
}

func TestPlayOccupied(t *testing.T) {
    g := New()
    if err := g.Play(0, 0); err != nil {
        t.Fatalf("first move failed: %v", err)
    }
    if err := g.Play(0, 0); err == nil || err != ErrOccupied {
        t.Fatalf("expected ErrOccupied on same cell, got %v", err)
    }
}

func TestTurnFlipsAfterValidMove(t *testing.T) {
    g := New()
    if g.Turn != X {
        t.Fatalf("expected X to start")
    }
    if err := g.Play(1, 1); err != nil {
        t.Fatalf("move failed: %v", err)
    }
    if g.Turn != O {
        t.Fatalf("expected turn to flip to O, got %v", g.Turn)
    }
}

func TestWinConditionsForX(t *testing.T) {
    winningLines := [][][2]int{
        // rows
        {{0, 0}, {0, 1}, {0, 2}},
        {{1, 0}, {1, 1}, {1, 2}},
        {{2, 0}, {2, 1}, {2, 2}},
        // cols
        {{0, 0}, {1, 0}, {2, 0}},
        {{0, 1}, {1, 1}, {2, 1}},
        {{0, 2}, {1, 2}, {2, 2}},
        // diags
        {{0, 0}, {1, 1}, {2, 2}},
        {{0, 2}, {1, 1}, {2, 0}},
    }
    filler := [][2]int{{1, 2}, {2, 1}, {1, 0}, {2, 0}, {0, 2}, {0, 1}}
    for _, line := range winningLines {
        g := New()
        seq := make([][2]int, 0, 5)
        // X, O, X, O, X on the line
        seq = append(seq, line[0])
        // choose O filler not on the line
        for _, f := range filler {
            if (f != line[0]) && (f != line[1]) && (f != line[2]) {
                seq = append(seq, f)
                break
            }
        }
        seq = append(seq, line[1])
        // another O filler not on the line and not same as previous filler
        for _, f := range filler {
            if (f != line[0]) && (f != line[1]) && (f != line[2]) && (f != seq[1]) {
                seq = append(seq, f)
                break
            }
        }
        seq = append(seq, line[2])

        playMoves(t, &g, seq)
        if !g.Over || g.Winner != X {
            t.Fatalf("expected X to win on line %v; over=%v winner=%v", line, g.Over, g.Winner)
        }
        if g.Moves != 5 {
            t.Fatalf("expected 5 moves to win, got %d", g.Moves)
        }
    }
}

func TestWinConditionsForO(t *testing.T) {
    winningLines := [][][2]int{
        // rows
        {{0, 0}, {0, 1}, {0, 2}},
        {{1, 0}, {1, 1}, {1, 2}},
        {{2, 0}, {2, 1}, {2, 2}},
        // cols
        {{0, 0}, {1, 0}, {2, 0}},
        {{0, 1}, {1, 1}, {2, 1}},
        {{0, 2}, {1, 2}, {2, 2}},
        // diags
        {{0, 0}, {1, 1}, {2, 2}},
        {{0, 2}, {1, 1}, {2, 0}},
    }
    // For O to win: X plays fillers, O plays the line cells.
    fillers := [][2]int{{1, 2}, {2, 1}, {1, 0}, {2, 0}, {0, 2}, {0, 1}, {2, 2}, {1, 1}}
    for _, line := range winningLines {
        g := New()
        seq := make([][2]int, 0, 6)
        // X filler not on line
        var f1, f2, f3 [2]int
        found := 0
        for _, f := range fillers {
            if (f != line[0]) && (f != line[1]) && (f != line[2]) {
                switch found {
                case 0:
                    f1 = f
                case 1:
                    if f != f1 { f2 = f }
                case 2:
                    if f != f1 && f != f2 { f3 = f }
                }
                found++
                if found == 3 { break }
            }
        }
        seq = append(seq, f1)      // X
        seq = append(seq, line[0]) // O
        seq = append(seq, f2)      // X
        seq = append(seq, line[1]) // O
        seq = append(seq, f3)      // X
        seq = append(seq, line[2]) // O wins

        playMoves(t, &g, seq)
        if !g.Over || g.Winner != O {
            t.Fatalf("expected O to win on line %v; over=%v winner=%v", line, g.Over, g.Winner)
        }
        if g.Moves != 6 {
            t.Fatalf("expected 6 moves to win for O, got %d", g.Moves)
        }
    }
}

func TestDrawNoWinner(t *testing.T) {
    g := New()
    // Draw pattern (no three in a row)
    seq := [][2]int{
        {0, 0}, {0, 1}, {0, 2},
        {1, 1}, {1, 0}, {1, 2},
        {2, 1}, {2, 0}, {2, 2},
    }
    playMoves(t, &g, seq)
    if !g.Over {
        t.Fatalf("expected game over on draw")
    }
    if g.Winner != Empty {
        t.Fatalf("expected no winner on draw, got %v", g.Winner)
    }
    if g.Moves != 9 {
        t.Fatalf("expected 9 moves on draw, got %d", g.Moves)
    }
}

func TestGameOverBlocksFurtherMoves(t *testing.T) {
    g := New()
    // X wins quickly on top row
    seq := [][2]int{{0, 0}, {1, 0}, {0, 1}, {1, 1}, {0, 2}}
    playMoves(t, &g, seq)
    if !g.Over || g.Winner != X {
        t.Fatalf("expected X win before extra move")
    }
    // Further move should be blocked
    if err := g.Play(2, 2); err == nil || err != ErrGameOver {
        t.Fatalf("expected ErrGameOver, got %v", err)
    }
}

