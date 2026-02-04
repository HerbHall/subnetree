package dashboard

import (
	"io/fs"
	"net/http"
	"strings"
)

// Handler returns an http.Handler that serves the built React SPA.
// For any request that doesn't match a static file and isn't an API route,
// it serves index.html so React Router can handle client-side routing.
func Handler() http.Handler {
	if distFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "dashboard not available (dev mode)", http.StatusNotFound)
		})
	}

	subFS, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic("dashboard: failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(subFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't serve SPA for API routes, health endpoints, or metrics
		if strings.HasPrefix(r.URL.Path, "/api/") ||
			r.URL.Path == "/healthz" ||
			r.URL.Path == "/readyz" ||
			r.URL.Path == "/metrics" {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if the file exists in the embedded FS
		f, err := subFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found -- serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
