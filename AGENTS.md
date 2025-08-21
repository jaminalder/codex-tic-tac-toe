# CRUSH.md

## AGENT DIRECTIVE — READ FIRST
You are in a Go repo for a **two-player Tic‑Tac‑Toe** web app. Keep it simple and idiomatic:

- **Two layers only**
  1) **Domain** (pure Go): board, rules, turns, win/draw. No HTTP/HTML/logging.
  2) **Web** (SSR + HTMX): routing, handlers, templates, realtime push.
- Prefer **concrete types**; zero values must be meaningful. No interfaces unless a consumer needs one.
- Concurrency belongs in the web/app layer (registry, joins, turns, SSE). Use a mutex, snapshot then fan‑out after unlock.

## Goal
Server‑rendered Tic‑Tac‑Toe for two humans. Share a game URL. When one moves, the other browser updates immediately.

## Tech & Decisions
- **Language:** Go
- **Router:** chi (`github.com/go-chi/chi/v5`)
- **Templates:** `html/template`
- **UI:** SSR with **HTMX** (plus HTMX **SSE** extension)
- **Realtime push:** **SSE** (Server‑Sent Events)
- **State:** in‑memory service; per‑game pub/sub; `sync.RWMutex` for safety
- **Identity:** random `player_id` cookie; first visitor → **X**, second → **O**, extras spectate

## Project layout
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
│  └─ templates.go             # load + helper funcs (iter, cell, etc.)
├─ web/templates/
│  ├─ base.html.tmpl
│  ├─ index.html.tmpl
│  ├─ game.html.tmpl           # page w/ SSE hookup
│  └─ _board.html.tmpl         # swappable fragment
└─ web/static/                 # htmx + sse ext + css
```

## Domain (internal/domain)
- Types:
  - `type Cell uint8; const (Empty Cell = iota; X; O)`
  - `type Board [9]Cell`
  - `type Game struct { Board; Turn Cell; Winner Cell; Over bool; Moves int }`
- Behavior:
  - `New()` sets `Turn: X`
  - `Play(r,c)` validates bounds/occupancy, applies move, checks win/draw, flips turn
- **No** imports beyond stdlib. **No** I/O, HTTP, logs, or time here.

## App / Service (internal/app)
- Holds:
  - `games: map[string]*GameState`
  - `subs: map[string]set[*subscriber]`
- `GameState`: `ID`, `domain.Game`, `X`, `O`, `Created`, `Updated`.
- Methods: `CreateGame()`, `Get(id)`, `Join(id, playerID)`, `Play(id, player, r, c)`, `Subscribe(ctx, id)`.
- **Mutex discipline:** Lock → read/modify → copy a **snapshot** of state **and** subscriber list → unlock → fan‑out to copied list (drop slow subscribers).

## Web (internal/web)
- Endpoints:
  - `GET  /` → landing page
  - `POST /game` → create + redirect
  - `GET  /game/{id}` → full page (SSR) + ensure `player_id` cookie (auto‑claim seat if free)
  - `POST /game/{id}/join` → optional explicit join (returns `_board` fragment)
  - `POST /game/{id}/play` → HTMX post; returns updated `_board` fragment
  - `GET  /game/{id}/events` → **SSE** stream broadcasting `_board` HTML
- Templates:
  - `game.html.tmpl` wires HTMX SSE:  
    `hx-ext="sse" hx-sse="connect:/game/{{.Game.ID}}/events"`  
    `<div id="board" hx-sse="swap:board">…</div>`

## HTMX contract
- Each cell button in `_board.html.tmpl`:
  - `hx-post="/game/{{.ID}}/play"`
  - `hx-target="#board" hx-swap="outerHTML"`
  - send `r`, `c`, `side` (from player cookie/assignment)
- Errors (not your turn / occupied) return the same fragment with a small alert region.

## Realtime (SSE)
- Server renders `_board` to HTML and emits as SSE **`event: board`**.
- Clients subscribe once per page; HTMX SSE swaps into `#board` on event.
- Send periodic heartbeats (e.g., `: ping`) and disable proxy buffering on this route.

## Persistence & Cleanup
- In‑memory only; consider TTL cleanup by `Updated`.
- Future: swap to Redis (hash for state + pub/sub) inside `internal/app` without touching the domain.

## Testing
- **Domain:** table‑driven wins/draw/illegal moves.
- **App:** seat assignment, turn enforcement, duplicate joins, subscribe/broadcast.
- **Web:** handler tests via `httptest` for statuses + fragment rendering.
- Run race detector: `go test -race ./...` and `go run -race ./cmd/ttt-server`.
- Run a single test: `go test -run TestName ./...`

## Non‑goals (MVP)
- No bots/AI, no auth system, no database, no WebSockets (SSE is enough).

## Definition of Done
- Shareable `/game/{id}` URL.
- Two browsers auto‑claim X/O; spectators allowed.
- Move posts update the mover; **other tab updates via SSE** quickly.
- Game ends correctly (win/draw); further moves blocked.
- Race‑detector clean.

## Style & Conventions
- Concrete types; meaningful zero values; tiny exported API.
- Pointer receivers when mutating or for large structs.
- Return `error`; no panics for control flow.
- Keep handlers thin; push logic into service/domain.
- Use `gofmt` for formatting.
- Group imports in order: stdlib, external packages, internal packages.

## Quick run
```sh
go run ./cmd/ttt-server
# open http://localhost:8080
```
