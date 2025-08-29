package web

import (
    "html/template"
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
    ))
    return &templates{t: t}
}

func (t *templates) render(w http.ResponseWriter, name string, data any) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    if err := t.t.ExecuteTemplate(w, name, data); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
}

