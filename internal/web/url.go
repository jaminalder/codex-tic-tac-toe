package web

import (
    "net/http"
)

// absoluteURL builds an absolute URL using request scheme and host.
// Prefers X-Forwarded-Proto when present; falls back to TLS presence.
func absoluteURL(r *http.Request, path string) string {
    scheme := r.Header.Get("X-Forwarded-Proto")
    if scheme == "" {
        if r.TLS != nil {
            scheme = "https"
        } else {
            scheme = "http"
        }
    }
    host := r.Host
    if host == "" {
        host = "localhost"
    }
    return scheme + "://" + host + path
}

