package domain

import "errors"

// Cell represents a board cell state.
type Cell uint8

const (
    Empty Cell = iota
    X
    O
)

// Board is a fixed 3x3 board stored row-major.
type Board [9]Cell

// Game holds the current state of a Tic-Tac-Toe match.
type Game struct {
    Board  Board
    Turn   Cell
    Winner Cell
    Over   bool
    Moves  int
}

// Errors returned by domain operations.
var (
    ErrOutOfBounds = errors.New("out of bounds")
    ErrOccupied    = errors.New("cell occupied")
    ErrGameOver    = errors.New("game over")
)

// New returns a new game with X to move.
func New() Game {
    return Game{Turn: X}
}

// Play attempts to play the current turn at row r, column c (0..2).
func (g *Game) Play(r, c int) error {
    if g.Over {
        return ErrGameOver
    }
    if r < 0 || r > 2 || c < 0 || c > 2 {
        return ErrOutOfBounds
    }
    idx := r*3 + c
    if g.Board[idx] != Empty {
        return ErrOccupied
    }

    // Place the mark
    g.Board[idx] = g.Turn
    g.Moves++

    // Check for a win
    if hasWin(g.Board, g.Turn) {
        g.Winner = g.Turn
        g.Over = true
        return nil
    }

    // Check for draw
    if g.Moves == 9 {
        g.Winner = Empty
        g.Over = true
        return nil
    }

    // Flip turn
    if g.Turn == X {
        g.Turn = O
    } else {
        g.Turn = X
    }
    return nil
}

func hasWin(b Board, side Cell) bool {
    lines := [8][3]int{
        // rows
        {0, 1, 2}, {3, 4, 5}, {6, 7, 8},
        // cols
        {0, 3, 6}, {1, 4, 7}, {2, 5, 8},
        // diags
        {0, 4, 8}, {2, 4, 6},
    }
    for _, ln := range lines {
        if b[ln[0]] == side && b[ln[1]] == side && b[ln[2]] == side {
            return true
        }
    }
    return false
}

