package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/HerbHall/subnetree/internal/version"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Prometheus HTTP metrics.
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests.",
		},
		[]string{"method", "path", "status"},
	)
	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
}

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware in order (first argument is outermost).
func Chain(handler http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		handler = mw[i](handler)
	}
	return handler
}

// requestIDKey is a context key for the request ID.
type requestIDKey struct{}

// RequestID returns the request ID from the context.
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(requestIDKey{}).(string); ok {
		return id
	}
	return ""
}

// RequestIDMiddleware generates or propagates X-Request-ID headers.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = generateID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey{}, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoggingMiddleware logs each HTTP request with duration and status, and
// records Prometheus metrics (request count and duration histogram).
// Paths in skipPaths are excluded from logging but still recorded in metrics.
func LoggingMiddleware(logger *zap.Logger, skipPaths []string) Middleware {
	skip := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(sw, r)

			duration := time.Since(start)

			if !skip[r.URL.Path] {
				logger.Info("http request",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", sw.status),
					zap.Duration("duration", duration),
					zap.String("remote", r.RemoteAddr),
					zap.String("request_id", RequestID(r.Context())),
				)
			}

			httpRequestsTotal.WithLabelValues(
				r.Method, r.URL.Path, strconv.Itoa(sw.status),
			).Inc()
			httpRequestDuration.WithLabelValues(
				r.Method, r.URL.Path,
			).Observe(duration.Seconds())
		})
	}
}

// SecurityHeadersMiddleware adds standard security headers to all responses.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		// CSP allows inline styles for Tailwind/shadcn-ui, data: URIs for inline SVGs
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// VersionHeaderMiddleware adds X-SubNetree-Version to all responses.
func VersionHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-SubNetree-Version", version.Short())
		next.ServeHTTP(w, r)
	})
}

// RecoveryMiddleware catches panics and returns a 500 problem response.
func RecoveryMiddleware(logger *zap.Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						zap.Any("panic", rec),
						zap.String("path", r.URL.Path),
						zap.String("request_id", RequestID(r.Context())),
					)
					InternalError(w, "an unexpected error occurred", r.URL.Path)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitMiddleware enforces per-IP rate limiting.
// Requests to paths in skipPaths are not rate limited.
func RateLimitMiddleware(rps float64, burst int, skipPaths []string) Middleware {
	rl := &ipRateLimiter{
		rateVal: rate.Limit(rps),
		burst:   burst,
	}
	skip := make(map[string]bool, len(skipPaths))
	for _, p := range skipPaths {
		skip[p] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			if !rl.allow(ip) {
				RateLimited(w, "rate limit exceeded", r.URL.Path)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ipRateLimiter tracks per-IP token-bucket rate limiters.
type ipRateLimiter struct {
	mu       sync.Mutex
	limiters map[string]*rateLimitEntry
	rateVal  rate.Limit
	burst    int
}

type rateLimitEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func (l *ipRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.limiters == nil {
		l.limiters = make(map[string]*rateLimitEntry)
	}

	e, ok := l.limiters[ip]
	if !ok {
		if len(l.limiters) >= 10000 {
			l.cleanup()
		}
		e = &rateLimitEntry{limiter: rate.NewLimiter(l.rateVal, l.burst)}
		l.limiters[ip] = e
	}
	e.lastSeen = time.Now()

	return e.limiter.Allow()
}

// cleanup removes entries not seen in the last 10 minutes.
// Must be called with l.mu held.
func (l *ipRateLimiter) cleanup() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for ip, e := range l.limiters {
		if e.lastSeen.Before(cutoff) {
			delete(l.limiters, ip)
		}
	}
}

// clientIP extracts the client IP from the request.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if parts := strings.SplitN(xff, ",", 2); len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// statusWriter wraps ResponseWriter to capture the status code.
type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.status = code
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.wroteHeader = true
	}
	return w.ResponseWriter.Write(b)
}

// generateID creates a random 32-character hex string for request IDs.
func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
