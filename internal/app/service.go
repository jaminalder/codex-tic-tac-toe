package app

import (
    "context"
    "errors"
    "sync"
    "time"

    "github.com/jaminalder/codex-tic-tac-toe/internal/domain"
    "github.com/google/uuid"
)

// Errors exposed by the service layer.
var (
    ErrNotFound    = errors.New("game not found")
    ErrNotYourTurn = errors.New("not your turn")
    ErrNotAPlayer  = errors.New("not a player")
)

// GameState is the in-memory state tracked per game.
type GameState struct {
    ID      string
    Game    domain.Game
    X       string
    O       string
    Created time.Time
    Updated time.Time
}

type subscriber struct {
    ch       chan []byte
    closeOnce sync.Once
}

func (s *subscriber) close() { s.closeOnce.Do(func() { close(s.ch) }) }

// Service manages games and subscribers.
type Service struct {
    mu     sync.Mutex
    games  map[string]*GameState
    subs   map[string]map[*subscriber]struct{}
    render func(GameState) []byte
}

// NewService creates a service with a default renderer (encodes nothing useful).
func NewService() *Service { return NewServiceWithRenderer(func(gs GameState) []byte { return nil }) }

// NewServiceWithRenderer allows injecting a renderer for broadcast payloads.
func NewServiceWithRenderer(renderer func(GameState) []byte) *Service {
    if renderer == nil {
        renderer = func(gs GameState) []byte { return nil }
    }
    return &Service{
        games:  make(map[string]*GameState),
        subs:   make(map[string]map[*subscriber]struct{}),
        render: renderer,
    }
}

// SetRenderer replaces the broadcast renderer function.
func (s *Service) SetRenderer(renderer func(GameState) []byte) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if renderer == nil {
        s.render = func(gs GameState) []byte { return nil }
        return
    }
    s.render = renderer
}

// CreateGame creates and registers a new game.
func (s *Service) CreateGame() (*GameState, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    id := uuid.NewString()
    now := time.Now()
    gs := &GameState{ID: id, Game: domain.New(), Created: now, Updated: now}
    s.games[id] = gs
    cp := *gs
    return &cp, nil
}

// Get returns a copy of the game state if present.
func (s *Service) Get(id string) (*GameState, bool) {
    s.mu.Lock()
    defer s.mu.Unlock()
    gs, ok := s.games[id]
    if !ok {
        return nil, false
    }
    cp := *gs
    return &cp, true
}

// Join assigns a seat to the player if available; returns Empty for spectators.
func (s *Service) Join(id, playerID string) (domain.Cell, *GameState, error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    gs, ok := s.games[id]
    if !ok {
        return domain.Empty, nil, ErrNotFound
    }
    side := domain.Empty
    if gs.X == "" || gs.X == playerID {
        gs.X = playerID
        side = domain.X
    } else if gs.O == "" || gs.O == playerID {
        gs.O = playerID
        side = domain.O
    }
    gs.Updated = time.Now()
    cp := *gs
    return side, &cp, nil
}

// Play validates seat and turn, applies a move, updates timestamps, and broadcasts.
func (s *Service) Play(id, playerID string, r, c int) (*GameState, error) {
    var payload []byte
    var cp GameState
    var toDrop []*subscriber

    s.mu.Lock()
    gs, ok := s.games[id]
    if !ok {
        s.mu.Unlock()
        return nil, ErrNotFound
    }
    // Validate player is seated
    var seat domain.Cell
    if gs.X == playerID {
        seat = domain.X
    } else if gs.O == playerID {
        seat = domain.O
    } else {
        s.mu.Unlock()
        return nil, ErrNotAPlayer
    }
    // Validate turn
    if seat != gs.Game.Turn {
        s.mu.Unlock()
        return nil, ErrNotYourTurn
    }
    // Apply move
    if err := gs.Game.Play(r, c); err != nil {
        s.mu.Unlock()
        return nil, err
    }
    gs.Updated = time.Now()

    // Snapshot state and subscribers
    cp = *gs
    subs := s.copySubsLocked(id)
    payload = s.render(cp)
    s.mu.Unlock()

    // Fan-out; drop slow subscribers by closing and marking for deletion
    for sub := range subs {
        select {
        case sub.ch <- payload:
        default:
            // drop slow subscriber
            sub.close()
            toDrop = append(toDrop, sub)
        }
    }
    if len(toDrop) > 0 {
        s.mu.Lock()
        for _, sub := range toDrop {
            if set, ok := s.subs[id]; ok {
                delete(set, sub)
            }
        }
        s.mu.Unlock()
    }
    return &cp, nil
}

// Subscribe registers a subscriber for a game. Returns a channel and an unsubscribe func.
func (s *Service) Subscribe(ctx context.Context, id string) (<-chan []byte, func()) {
    s.mu.Lock()
    defer s.mu.Unlock()
    if _, ok := s.games[id]; !ok {
        // create lazily to allow subscriptions before CreateGame in some flows
        s.games[id] = &GameState{ID: id, Game: domain.New(), Created: time.Now(), Updated: time.Now()}
    }
    set := s.subs[id]
    if set == nil {
        set = make(map[*subscriber]struct{})
        s.subs[id] = set
    }
    sub := &subscriber{ch: make(chan []byte, 1)}
    set[sub] = struct{}{}

    unsubOnce := &sync.Once{}
    unsub := func() {
        unsubOnce.Do(func() {
            s.mu.Lock()
            if set, ok := s.subs[id]; ok {
                delete(set, sub)
            }
            s.mu.Unlock()
            sub.close()
        })
    }
    go func() {
        <-ctx.Done()
        unsub()
    }()
    return sub.ch, unsub
}

func (s *Service) copySubsLocked(id string) map[*subscriber]struct{} {
    out := make(map[*subscriber]struct{})
    if set, ok := s.subs[id]; ok {
        for k := range set {
            out[k] = struct{}{}
        }
    }
    return out
}
