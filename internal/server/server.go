package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/HerbHall/netvantage/internal/version"
	"github.com/HerbHall/netvantage/pkg/plugin"
	"go.uber.org/zap"
)

// PluginSource provides the server with plugin metadata and routes.
// Defined here (consumer-side) rather than importing the concrete registry.
type PluginSource interface {
	AllRoutes() map[string][]plugin.Route
	All() []plugin.Plugin
}

// Server is the main NetVantage server.
type Server struct {
	httpServer *http.Server
	plugins    PluginSource
	logger     *zap.Logger
	mux        *http.ServeMux
}

// New creates a new Server instance.
func New(addr string, plugins PluginSource, logger *zap.Logger) *Server {
	mux := http.NewServeMux()

	s := &Server{
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      mux,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		plugins: plugins,
		logger:  logger,
		mux:     mux,
	}

	s.registerCoreRoutes()
	s.mountPluginRoutes()

	return s
}

// registerCoreRoutes sets up routes that are always available.
func (s *Server) registerCoreRoutes() {
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

// handleHealth returns the server health status.
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-NetVantage-Version", version.Short())
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"service": "netvantage",
		"version": version.Map(),
	})
}

// handlePlugins returns the list of registered plugins.
func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	plugins := s.plugins.All()
	type pluginResponse struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
	}
	info := make([]pluginResponse, 0, len(plugins))
	for _, p := range plugins {
		pi := p.Info()
		info = append(info, pluginResponse{
			Name:        pi.Name,
			Version:     pi.Version,
			Description: pi.Description,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-NetVantage-Version", version.Short())
	_ = json.NewEncoder(w).Encode(info)
}
