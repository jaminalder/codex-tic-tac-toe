# AGENTS.md — Implementation Guide

## Mission
Build a two-player Tic‑Tac‑Toe web app in Go with server-rendered HTML and realtime updates via Server‑Sent Events (SSE). Shareable game URL, no persistence beyond memory, clean layering.

## Architecture
- Two layers only:
  1) Domain (pure Go): board, rules, turns, win/draw. No HTTP/HTML/logging/time.
  2) Web/App (SSR + HTMX + SSE): routing, handlers, templates, identity, in‑memory state, pub/sub.
- Concrete types, meaningful zero values. Avoid interfaces unless a consumer needs one.
- Concurrency in app layer: Guard state with a mutex. When broadcasting, take a snapshot inside the lock, then unlock and fan‑out.

## Tech
- Language: Go
- Router: chi (`github.com/go-chi/chi/v5`)
- Templates: `html/template`
- UI: SSR with HTMX and HTMX SSE extension (via CDN)
- Realtime: SSE (no WebSockets)
- State: in‑memory service
- Identity: random `player_id` cookie; first visitor is X, second is O, extras spectate

## Layout
```
ttt/
├─ cmd/ttt-server/main.go
├─ internal/domain/            # pure logic
│  ├─ game.go                  # Board, Game, Play(), win/draw
│  └─ game_test.go
├─ internal/app/               # state + concurrency + pub/sub
│  ├─ service.go               # Create/Get/Join/Play/Subscribe
│  └─ ids.go                   # short IDs
├─ internal/web/               # HTTP boundary
│  ├─ server.go                # chi router + middleware
│  ├─ handlers.go              # index/create/view/play/join/sse
│  └─ templates.go             # load + helper funcs (iter, cell, symbol)
├─ web/templates/
│  ├─ base.html.tmpl
│  ├─ index.html.tmpl
│  ├─ game.html.tmpl           # page w/ SSE hookup
│  └─ _board.html.tmpl         # swappable fragment
└─ web/static/                 # optional local css/assets (HTMX via CDN)
```

## Domain Layer (internal/domain)
- Types:
  - `type Cell uint8` with `const (Empty Cell = iota; X; O)`
  - `type Board [9]Cell` (row-major; index = r*3 + c)
  - `type Game struct { Board; Turn Cell; Winner Cell; Over bool; Moves int }`
- Construction: `func New() Game` returns zero-valid `Game` with `Turn: X`.
- Behavior:
  - `func (g *Game) Play(r, c int) error`
    - Validates bounds (0..2) and empty cell.
    - Writes current `Turn` to cell, increments `Moves`.
    - Checks win across 8 lines; sets `Winner` and `Over` when found.
    - If no win and `Moves == 9`, sets `Over = true` and `Winner = Empty` (draw).
    - Flips `Turn` (X↔O) only if not `Over`.
- Errors: exported sentinels: `ErrOutOfBounds`, `ErrOccupied`, `ErrGameOver`.
- Purity: stdlib only, no I/O, no time, no HTTP.

## App/Service Layer (internal/app)
- State:
  - `games: map[string]*GameState`
  - `subs: map[string]map[*subscriber]struct{}`
- GameState:
  - `ID string`, `Game domain.Game`, `X string`, `O string`, `Created time.Time`, `Updated time.Time`.
- Subscriber:
  - `ch chan []byte`, `done <-chan struct{}`; per-game set for broadcast.
- Mutex discipline: Single `sync.Mutex` (or `RWMutex` used as needed). For any mutation or read+publish:
  - Lock → read/modify → render snapshot data and copy subscriber list → unlock → fan‑out to copied list with non‑blocking send or short timeout; drop slow subscribers.
- Methods:
  - `CreateGame() (*GameState, error)`
  - `Get(id string) (*GameState, bool)`
  - `Join(id, playerID string) (side domain.Cell, gs *GameState, err error)`
  - `Play(id, playerID string, r, c int) (*GameState, error)`
  - `Subscribe(ctx context.Context, id string) (<-chan []byte, func())` // returns channel and unsubscribe
- Seat rules:
  - First unique `playerID` to visit claims X; second claims O; returning visitors keep their seat; others spectate.
  - `Play` validates that `playerID` matches current turn’s seat.
- IDs: `ids.go` provides UUIDv4 strings for game IDs and player IDs; no collision handling required.
- Cleanup (optional): periodic scan drops games idle past TTL (e.g., 30–60 minutes).

## Web Layer (internal/web)
- Server: chi router, listen on `:8080` by default.
- Static: serve `web/static` at `/static/` (if present; HTMX/SSE loaded via CDN in base template).
- Templates: loaded once at startup; helper funcs:
  - `iter(n int) []int` for loops; `cellSymbol(c Cell) string` returns "", "X", "O"; `eq(a,b any) bool`.
- Cookie: name `player_id`, long-lived (e.g., 1 year), `Path=/`, `SameSite=Lax`. Prefer `Secure` when behind HTTPS.
- Endpoints:
  - `GET /` → landing page with create form.
  - `POST /game` → create game → redirect to `/game/{id}`.
  - `GET /game/{id}` → ensures `player_id` cookie; auto-claim seat if available; renders page; SSE connects.
  - `POST /game/{id}/join` → explicit join (optional); returns `_board` fragment.
  - `POST /game/{id}/play` → HTMX post; form fields: `r`, `c` (0..2), `side` (client hint). Server trusts cookie and seat, not `side`. Returns `_board` fragment.
  - `GET /game/{id}/events` → SSE; headers `Content-Type: text/event-stream`, `Cache-Control: no-cache`, `X-Accel-Buffering: no`. Send heartbeats.
- HTMX SSE wiring in `game.html.tmpl`:
  - Root: `hx-ext="sse" hx-sse="connect:/game/{{.Game.ID}}/events"`
  - Board container: `<div id="board" hx-sse="swap:board">…</div>`
- Fragment contract (`_board.html.tmpl`):
  - 3×3 buttons with:
    - `hx-post="/game/{{.ID}}/play"`
    - `hx-target="#board" hx-swap="outerHTML"`
    - hidden inputs: `name="r"`, `name="c"`, client-provided `side` (not trusted).
  - Disabled buttons for occupied cells or when game over or not your turn.
  - Shows small alert region for last error (turn/occupied/out-of-bounds/game-over).

## SSE
- Event name: `event: board` for updates; data is rendered `_board` HTML.
- Heartbeat: every ~15s send comment `: ping\n\n`.
- Lifecycle: on `Subscribe`, create buffered channel (e.g., 1–2). On client disconnect, cancel ctx → unsubscribe.
- Broadcast: render HTML while holding lock (or copy data then render outside if templates are pure) → send to subscribers after unlock.

## Validation & Errors
- Turn enforcement: server checks `player_id` matches seat of `Game.Turn`.
- Spectators: `Play` is no-op with error; return fragment with alert.
- HTTP status: return 200 with fragment, even on invalid moves; present error in alert region to keep HTMX happy.

- Domain: table-driven wins across 8 lines, draw, illegal moves (occupied/out-of-bounds), game-over blocks moves.
- App: seat assignment, rejoin keeps seat, spectators blocked, turn enforcement, subscribe/broadcast fan-out (drop slow subs).
- Web: handlers return expected statuses; fragment renders with correct disabled/enabled states; SSE endpoint streams with `board` events and heartbeats.
- Race detector: `go test -race ./...` and `go run -race ./cmd/ttt-server`.
- Single test: `go test -run TestName ./...`.

## Definition of Done
- Shareable `/game/{id}` URL.
- Two browsers auto-claim X/O; spectators allowed.
- Moves update the mover immediately; the other tab updates via SSE quickly.
- Game ends correctly (win/draw); further moves blocked.
- `go test -race` clean.

## Style & Conventions
- API: tiny, concrete types; zero values meaningful.
- Receivers: pointer when mutating or large structs.
- Errors: return `error`; no panics for control flow. Use sentinel errors in domain and wrap with context in app/web.
- Handlers: thin; push logic into service/domain. Centralize template rendering helpers.
- Imports: stdlib, then external, then internal groups. `gofmt` formatting.

## Finalized Decisions
- IDs: use UUIDv4 for both game and player IDs; ignore collision risk.
- Assets: load HTMX and the SSE extension from a CDN in templates.
- Cookie: 1 year TTL, `SameSite=Lax`, set `Secure` when `X-Forwarded-Proto=https`.
- SSE: heartbeat every ~15s.
- Cleanup: none for MVP (no TTL eviction); keep it simple.
- Port/config: default `:8080`, allow `PORT` env override.
- Error UX: brief inline alert region in the board fragment.

## Quick Run
```sh
go run ./cmd/ttt-server
# open http://localhost:8080
```
