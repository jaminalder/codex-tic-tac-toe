package web

import (
    "bytes"
    "html/template"
    "net/http"

    "github.com/jaminalder/codex-tic-tac-toe/internal/domain"
    "github.com/google/uuid"
)

type templates struct {
    base   *template.Template
    game   *template.Template
    board  *template.Template
    index  *template.Template
}

func funcs() template.FuncMap {
    return template.FuncMap{
        "iter": func(n int) []int { a := make([]int, n); for i := range a { a[i] = i }; return a },
        "cellSymbol": func(c domain.Cell) string {
            switch c { case domain.X: return "X"; case domain.O: return "O"; default: return "" }
        },
        "eq": func(a, b any) bool { return a == b },
        "add": func(a, b int) int { return a + b },
        "mul": func(a, b int) int { return a * b },
    }
}

func loadTemplates() *templates {
    // Minimal inline templates to satisfy tests; can be replaced by file loading later.
    base := template.Must(template.New("base").Funcs(funcs()).Parse(`<!doctype html><html><head>
<meta charset="utf-8"/>
<script src="https://unpkg.com/htmx.org@1.9.12"></script>
<script src="https://unpkg.com/htmx.org/dist/ext/sse.js"></script>
</head><body>{{template "content" .}}</body></html>`))
    // Define the board template within the same set so game can include it
    template.Must(base.New("board").Funcs(funcs()).Parse(boardTemplate))
    index := template.Must(template.Must(base.Clone()).New("content").Parse(`<h1>TicTacToe</h1><form action="/game" method="post"><button>Create</button></form>`))
    game := template.Must(template.Must(base.Clone()).New("content").Parse(`
<div hx-ext="sse" hx-sse="connect:/game/{{.Game.ID}}/events">
  <div id="board" hx-sse="swap:board">{{template "board" .}}</div>
</div>`))
    // Standalone board template used for fragment rendering
    board := template.Must(template.New("board_only").Funcs(funcs()).Parse(boardTemplate))
    return &templates{base: base, game: game, board: board, index: index}
}

func renderTemplate(t *template.Template, name string, data any) []byte {
    var buf bytes.Buffer
    if name == "" {
        _ = t.Execute(&buf, data)
    } else {
        _ = t.ExecuteTemplate(&buf, name, data)
    }
    return buf.Bytes()
}

const boardTemplate = `
<div id="board">
  {{if .Error}}
  <div class="alert">{{.Error}}</div>
  {{end}}
  {{/* 3x3 grid */}}
  {{range $r := iter 3}}
  <div class="row">
    {{range $c := iter 3}}
      <form hx-post="/game/{{.ID}}/play" hx-target="#board" hx-swap="outerHTML" method="post">
        <input type="hidden" name="r" value="{{$r}}">
        <input type="hidden" name="c" value="{{$c}}">
        <button type="submit">{{cellSymbol (index .Game.Board (add (mul $r 3) $c))}}</button>
      </form>
    {{end}}
  </div>
  {{end}}
</div>
`

// Data models for templates
type pageData struct {
    ID    string
    Game  any
}

// Helper to set cookie
func ensurePlayerCookie(w http.ResponseWriter, r *http.Request) string {
    if c, err := r.Cookie("player_id"); err == nil && c.Value != "" {
        return c.Value
    }
    // Generate UUIDv4 for player ID
    v := uuid.NewString()
    http.SetCookie(w, &http.Cookie{Name: "player_id", Value: v, Path: "/"})
    return v
}
