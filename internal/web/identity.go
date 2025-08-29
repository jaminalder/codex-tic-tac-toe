package web

import (
    "net/http"
    "time"

    "github.com/google/uuid"
)

const playerCookie = "player_id"

func ensurePlayerID(w http.ResponseWriter, r *http.Request) string {
    if c, err := r.Cookie(playerCookie); err == nil && c.Value != "" {
        return c.Value
    }
    id := uuid.NewString()
    ck := &http.Cookie{
        Name:     playerCookie,
        Value:    id,
        Path:     "/",
        HttpOnly: true,
        SameSite: http.SameSiteLaxMode,
        Expires:  time.Now().Add(365 * 24 * time.Hour),
    }
    // In real deployments behind HTTPS, set Secure when X-Forwarded-Proto=https.
    http.SetCookie(w, ck)
    return id
}

