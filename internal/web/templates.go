package web

import (
    "html/template"
    "log"
    "net/http"
    "os"
    "path/filepath"
)

type templates struct {
    index *template.Template
    game  *template.Template
    lobby *template.Template
}

func mustLoadTemplates() *templates {
    funcs := template.FuncMap{}
    // Determine templates directory relative to current working directory (works in tests and at runtime).
    candidates := []string{
        "web/templates",
        "../web/templates",
        "../../web/templates",
    }
    var dir string
    for _, c := range candidates {
        if _, err := os.Stat(filepath.Join(c, "base.html.tmpl")); err == nil {
            dir = c
            break
        }
    }
    if dir == "" {
        log.Fatal("templates directory not found; looked in ", candidates)
    }
    basePath := filepath.Join(dir, "base.html.tmpl")
    // Parse base, then clone for each page to avoid define name collisions.
    base := template.Must(template.New("base.html.tmpl").Funcs(funcs).ParseFiles(basePath))
    indexT := template.Must(template.Must(base.Clone()).ParseFiles(filepath.Join(dir, "index.html.tmpl")))
    gameT := template.Must(template.Must(base.Clone()).ParseFiles(filepath.Join(dir, "game.html.tmpl")))
    lobbyT := template.Must(template.Must(base.Clone()).ParseFiles(filepath.Join(dir, "lobby.html.tmpl")))
    return &templates{index: indexT, game: gameT, lobby: lobbyT}
}

func (t *templates) render(w http.ResponseWriter, name string, data any) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    log.Printf("render template=%s", name)
    var err error
    switch name {
    case "index.html.tmpl":
        err = t.index.ExecuteTemplate(w, name, data)
    case "game.html.tmpl":
        err = t.game.ExecuteTemplate(w, name, data)
    case "lobby.html.tmpl":
        err = t.lobby.ExecuteTemplate(w, name, data)
    default:
        http.Error(w, "template not found", http.StatusInternalServerError)
        return
    }
    if err != nil {
        log.Printf("template error name=%s err=%v", name, err)
        http.Error(w, "template render error", http.StatusInternalServerError)
        return
    }
}
