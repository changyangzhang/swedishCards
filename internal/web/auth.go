package web

import (
	"crypto/subtle"
	"net/http"
)

// basicAuth wraps next with HTTP Basic Auth. When both user and pass are empty,
// auth is disabled — the middleware becomes a no-op so local dev with
// docker-compose still works without configuring credentials.
func basicAuth(user, pass string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if user == "" && pass == "" {
			return next
		}
		userB := []byte(user)
		passB := []byte(pass)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok ||
				subtle.ConstantTimeCompare([]byte(u), userB) != 1 ||
				subtle.ConstantTimeCompare([]byte(p), passB) != 1 {
				w.Header().Set("WWW-Authenticate", `Basic realm="Swedish Cards", charset="UTF-8"`)
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
