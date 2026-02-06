// Package server provides the main HTTP server for SubNetree.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/HerbHall/subnetree/internal/version"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

// PluginSource provides the server with plugin metadata and routes.
// Defined here (consumer-side) rather than importing the concrete registry.
type PluginSource interface {
	AllRoutes() map[string][]plugin.Route
	All() []plugin.Plugin
}

// ReadinessChecker verifies that the server is ready to serve traffic.
// Returns nil if ready, an error describing why not otherwise.
type ReadinessChecker func(ctx context.Context) error

// RouteRegistrar allows external packages to register routes and middleware
// on the server without creating import cycles (consumer-side interface).
type RouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
	Middleware() func(http.Handler) http.Handler
}

// Server is the main SubNetree HTTP server.
type Server struct {
	httpServer *http.Server
	plugins    PluginSource
	logger     *zap.Logger
	mux        *http.ServeMux
	ready      ReadinessChecker
}

// SimpleRouteRegistrar can register routes without middleware.
type SimpleRouteRegistrar interface {
	RegisterRoutes(mux *http.ServeMux)
}

// New creates a new Server with middleware and routes.
// The auth parameter is optional; pass nil to disable authentication.
// The dashboard parameter is optional; pass nil to disable dashboard serving.
// When devMode is true, Swagger UI is served at /swagger/.
// Additional route registrars can be passed to register extra API routes.
func New(addr string, plugins PluginSource, logger *zap.Logger, ready ReadinessChecker, auth RouteRegistrar, dashboard http.Handler, devMode bool, extraRoutes ...SimpleRouteRegistrar) *Server {
	mux := http.NewServeMux()

	s := &Server{
		plugins: plugins,
		logger:  logger,
		mux:     mux,
		ready:   ready,
	}

	s.registerRoutes()
	if auth != nil {
		auth.RegisterRoutes(mux)
	}
	for _, r := range extraRoutes {
		r.RegisterRoutes(mux)
	}
	s.mountPluginRoutes()

	if devMode {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))
		logger.Info("swagger UI enabled (dev_mode)", zap.String("path", "/swagger/"))
	}

	// Mount dashboard last as a catch-all for SPA routing
	if dashboard != nil {
		mux.Handle("/", dashboard)
	}

	// Middleware chain: outermost listed first.
	middlewares := []Middleware{
		RecoveryMiddleware(logger),
		RequestIDMiddleware,
		LoggingMiddleware(logger),
		SecurityHeadersMiddleware,
		VersionHeaderMiddleware,
		RateLimitMiddleware(100, 200, []string{"/healthz", "/readyz", "/metrics"}),
	}
	if auth != nil {
		middlewares = append(middlewares, auth.Middleware())
	}

	handler := Chain(mux, middlewares...)

	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// registerRoutes sets up all core routes.
func (s *Server) registerRoutes() {
	// Unversioned operational endpoints.
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)
	s.mux.HandleFunc("GET /readyz", s.handleReadyz)
	s.mux.Handle("GET /metrics", promhttp.Handler())

	// Versioned API endpoints.
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/v1/plugins", s.handlePlugins)
}

// mountPluginRoutes registers all plugin routes under /api/v1/{plugin}/.
func (s *Server) mountPluginRoutes() {
	allRoutes := s.plugins.AllRoutes()
	for pluginName, routes := range allRoutes {
		for _, route := range routes {
			pattern := fmt.Sprintf("%s /api/v1/%s%s", route.Method, pluginName, route.Path)
			s.mux.HandleFunc(pattern, route.Handler)
			s.logger.Debug("mounted route",
				zap.String("plugin", pluginName),
				zap.String("pattern", pattern),
			)
		}
	}
}

// Start begins serving HTTP requests.
func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", zap.String("addr", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("HTTP server error: %w", err)
	}
	return nil
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	return s.httpServer.Shutdown(ctx)
}

// handleHealthz is a liveness probe -- returns 200 if the process is running.
func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
}

// handleReadyz checks readiness -- returns 200 if the server can serve traffic.
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if s.ready != nil {
		if err := s.ready(r.Context()); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"status": "not ready",
				"error":  err.Error(),
			})
			return
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
}

// HealthResponse is the response for GET /health.
type HealthResponse struct {
	Status  string            `json:"status" example:"ok"`
	Service string            `json:"service" example:"subnetree"`
	Version map[string]string `json:"version"`
}

// PluginResponse describes a registered plugin.
type PluginResponse struct {
	Name        string `json:"name" example:"recon"`
	Version     string `json:"version" example:"0.1.0"`
	Description string `json:"description" example:"Network discovery and scanning"`
}

// handleHealth returns detailed health information (versioned API endpoint).
//
//	@Summary		Health check
//	@Description	Returns service health status with version information.
//	@Tags			system
//	@Produce		json
//	@Success		200	{object}	HealthResponse
//	@Router			/health [get]
func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(HealthResponse{
		Status:  "ok",
		Service: "subnetree",
		Version: version.Map(),
	})
}

// handlePlugins returns the list of registered plugins.
//
//	@Summary		List plugins
//	@Description	Returns all registered plugins with their metadata.
//	@Tags			system
//	@Produce		json
//	@Success		200	{array}	PluginResponse
//	@Router			/plugins [get]
func (s *Server) handlePlugins(w http.ResponseWriter, _ *http.Request) {
	plugins := s.plugins.All()
	info := make([]PluginResponse, 0, len(plugins))
	for _, p := range plugins {
		pi := p.Info()
		info = append(info, PluginResponse{
			Name:        pi.Name,
			Version:     pi.Version,
			Description: pi.Description,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(info)
}
