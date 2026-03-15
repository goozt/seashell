package middleware

import (
	"net/http"

	"github.com/goozt/seashell/api/response"
)

// RequireNodeSecret validates the X-Node-Secret header for P2P routes.
// Requests without the correct secret are rejected with 401.
func RequireNodeSecret(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Node-Secret") != secret {
				response.Unauthorized(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
