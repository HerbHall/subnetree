package auth

import (
	"context"
	"net/http"
	"strings"
)

// authUserKey is a context key for the authenticated user.
type authUserKey struct{}

// UserFromContext returns the authenticated user from the request context.
// Returns nil if the request is not authenticated.
func UserFromContext(ctx context.Context) *Claims {
	if c, ok := ctx.Value(authUserKey{}).(*Claims); ok {
		return c
	}
	return nil
}

// Public paths that don't require authentication.
var publicPaths = map[string]bool{
	"/api/v1/auth/login":        true,
	"/api/v1/auth/refresh":      true,
	"/api/v1/auth/logout":       true,
	"/api/v1/auth/setup":        true,
	"/api/v1/auth/setup/status": true,
}

// AuthMiddleware validates JWT access tokens on API routes.
// Public paths and non-API paths (healthz, readyz, metrics) are skipped.
func AuthMiddleware(tokens *TokenService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip non-API paths (healthz, readyz, metrics, etc.).
			if !strings.HasPrefix(r.URL.Path, "/api/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip WebSocket paths (auth handled by WS handler via query param).
			if strings.HasPrefix(r.URL.Path, "/api/v1/ws/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip public auth paths.
			if publicPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Extract Bearer token from Authorization header.
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeAuthError(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := tokens.ValidateAccessToken(tokenString)
			if err != nil {
				writeAuthError(w, http.StatusUnauthorized, "invalid or expired access token")
				return
			}

			// Set claims in context for downstream handlers.
			ctx := context.WithValue(r.Context(), authUserKey{}, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
