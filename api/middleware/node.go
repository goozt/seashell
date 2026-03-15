package middleware

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/goozt/seashell/api/response"
	"github.com/goozt/seashell/store"
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

// RequireMTLS verifies that the TLS peer certificate belongs to a node registered
// in the DB (matched by SHA-256 fingerprint of the leaf certificate).
// Falls back to allowing the request if TLS info is unavailable (e.g. tests, dev mode).
func RequireMTLS(db *store.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
				// No TLS / no client cert — reject in strict mode.
				response.Unauthorized(w)
				return
			}
			leafCert := r.TLS.PeerCertificates[0]
			fingerprint := fmt.Sprintf("%x", sha256.Sum256(leafCert.Raw))

			nodes, err := db.ListActiveNodes()
			if err != nil {
				response.InternalError(w, "node lookup error")
				return
			}
			for _, n := range nodes {
				if n.CertFingerprint == fingerprint {
					next.ServeHTTP(w, r)
					return
				}
			}
			response.Unauthorized(w)
		})
	}
}

// tlsStateFromRequest is a helper for tests — unused in production paths but
// prevents the tls import from being stripped.
var _ = tls.Certificate{}
