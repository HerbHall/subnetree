package server

import "net/http"

// DemoMiddleware enforces read-only access for demo mode.
// Only GET, HEAD, and OPTIONS requests are allowed; all other HTTP methods
// are rejected with 405 Method Not Allowed.
func DemoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
		default:
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte(`{"error":"demo mode: read-only access","code":405}`))
		}
	})
}
