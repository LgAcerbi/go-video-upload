package middleware

import (
	"net/http"
	"strings"
)

const headerXAPIKey = "X-Api-Key"

// RequireAPIKey returns a middleware that rejects requests whose X-Api-Key header
// does not match the expected token. The token must be non-empty.
func RequireAPIKey(expectedToken string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimSpace(r.Header.Get(headerXAPIKey))
			if key == "" {
				http.Error(w, "missing X-Api-Key header", http.StatusUnauthorized)
				return
			}
			if key != expectedToken {
				http.Error(w, "invalid X-Api-Key", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
