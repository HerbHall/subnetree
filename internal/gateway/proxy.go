package gateway

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"go.uber.org/zap"
)

// ReverseProxyManager manages httputil.ReverseProxy instances keyed by session ID.
type ReverseProxyManager struct {
	mu      sync.RWMutex
	proxies map[string]*httputil.ReverseProxy
	logger  *zap.Logger
}

// NewReverseProxyManager creates a new proxy manager.
func NewReverseProxyManager(logger *zap.Logger) *ReverseProxyManager {
	return &ReverseProxyManager{
		proxies: make(map[string]*httputil.ReverseProxy),
		logger:  logger,
	}
}

// CreateProxy creates and stores a reverse proxy for the given session.
// The proxy targets http://{host}:{port} using the session's ProxyTarget.
func (pm *ReverseProxyManager) CreateProxy(session *Session, scheme string) error {
	if scheme == "" {
		scheme = "http"
	}

	targetURL, err := url.Parse(fmt.Sprintf("%s://%s:%d", scheme, session.Target.Host, session.Target.Port))
	if err != nil {
		return fmt.Errorf("parse proxy target URL: %w", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// Custom error handler that logs via zap instead of writing to stderr.
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, proxyErr error) {
		pm.logger.Warn("reverse proxy error",
			zap.String("session_id", session.ID),
			zap.String("target", targetURL.String()),
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Error(proxyErr),
		)
		w.WriteHeader(http.StatusBadGateway)
	}

	pm.mu.Lock()
	pm.proxies[session.ID] = proxy
	pm.mu.Unlock()

	pm.logger.Debug("proxy created",
		zap.String("session_id", session.ID),
		zap.String("target", targetURL.String()),
	)
	return nil
}

// ServeProxy serves an HTTP request through the proxy associated with the given session ID.
// Returns an error if no proxy exists for the session.
func (pm *ReverseProxyManager) ServeProxy(sessionID string, w http.ResponseWriter, r *http.Request) error {
	pm.mu.RLock()
	proxy, ok := pm.proxies[sessionID]
	pm.mu.RUnlock()

	if !ok {
		return fmt.Errorf("no proxy for session %q", sessionID)
	}

	proxy.ServeHTTP(w, r)
	return nil
}

// RemoveProxy removes and cleans up the proxy for the given session ID.
func (pm *ReverseProxyManager) RemoveProxy(sessionID string) {
	pm.mu.Lock()
	delete(pm.proxies, sessionID)
	pm.mu.Unlock()

	pm.logger.Debug("proxy removed", zap.String("session_id", sessionID))
}

// CloseAll removes all active proxies.
func (pm *ReverseProxyManager) CloseAll() {
	pm.mu.Lock()
	for id := range pm.proxies {
		delete(pm.proxies, id)
	}
	pm.mu.Unlock()

	pm.logger.Debug("all proxies closed")
}

// Count returns the number of active proxies.
func (pm *ReverseProxyManager) Count() int {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return len(pm.proxies)
}
