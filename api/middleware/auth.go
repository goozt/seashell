package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/service"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

// JWT returns middleware that validates the Bearer token and injects claims into the context.
func JWT(authSvc *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" || !strings.HasPrefix(header, "Bearer ") {
				response.Unauthorized(w)
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")
			claims, err := authSvc.ValidateAccessToken(tokenStr)
			if err != nil {
				response.Unauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns middleware that rejects requests whose token role is not in the allowed list.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := ClaimsFromContext(r.Context())
			if claims == nil || !allowed[claims.Role] {
				response.Forbidden(w)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthorityOwner rejects requests where the user is not an authority owner.
func RequireAuthorityOwner(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := ClaimsFromContext(r.Context())
		if claims == nil || claims.AuthorityRole != "owner" || claims.AuthorityID == "" {
			response.Forbidden(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ClaimsFromContext extracts the AccessClaims from the request context.
// Returns nil if not present.
func ClaimsFromContext(ctx context.Context) *service.AccessClaims {
	c, _ := ctx.Value(UserClaimsKey).(*service.AccessClaims)
	return c
}
