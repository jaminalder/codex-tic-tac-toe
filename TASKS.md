# TASKS.md — Test-Driven Plan

1) Write domain tests (wins/draw/illegal) — completed
2) Implement domain Game/Board/Play — completed
3) Write app service tests suite — completed
   - create/get
   - join seats + rejoin keeps seat
   - turn enforcement + play updates
   - subscribe/broadcast fan-out; drop slow subs
4) Implement service (UUID, seats, mutex, snapshot fan-out) — completed
5) Write web handler tests (SSR/HTMX) — completed
   - statuses, fragment rendering, cookie/auto-claim
6) Implement HTTP server, routes, templates — completed
7) Implement SSE events + heartbeats — completed
8) Render board fragment + inline errors — completed
9) Run race detector and iterate — completed
10) Add server entrypoint (cmd/ttt-server) — completed

Notes
- Keep this file updated as tasks progress (pending → in_progress → completed).
- Source of truth mirrors the in-tool plan.
