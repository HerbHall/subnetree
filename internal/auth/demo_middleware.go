package auth

import (
	"context"
	"net/http"
	"strings"
)

// DemoAuthMiddleware skips JWT validation and injects synthetic viewer claims.
// This allows all API endpoints to work without login in demo mode.
func DemoAuthMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only inject claims for API paths (same logic as AuthMiddleware).
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			demoClaims := &Claims{
				UserID:   "demo-user",
				Username: "demo",
				Role:     "viewer",
			}
			ctx := context.WithValue(r.Context(), authUserKey{}, demoClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
