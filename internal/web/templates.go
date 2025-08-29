package web

import (
    "html/template"
    "log"
    "net/http"
)

type templates struct {
    t *template.Template
}

func mustLoadTemplates() *templates {
    funcs := template.FuncMap{}
    // Load base first, then pages.
    t := template.Must(template.New("base.html.tmpl").Funcs(funcs).ParseFiles(
        "web/templates/base.html.tmpl",
        "web/templates/index.html.tmpl",
        "web/templates/game.html.tmpl",
    ))
    return &templates{t: t}
}

func (t *templates) render(w http.ResponseWriter, name string, data any) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    log.Printf("render template=%s", name)
    if err := t.t.ExecuteTemplate(w, name, data); err != nil {
        log.Printf("template error name=%s err=%v", name, err)
        http.Error(w, "template render error", http.StatusInternalServerError)
        return
    }
}
