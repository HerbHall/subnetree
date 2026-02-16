package recon

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"net/http"
	"runtime"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"go.uber.org/zap"

	// models imported for swagger annotation resolution.
	_ "github.com/HerbHall/subnetree/pkg/models"
)

// diagSemaphore limits concurrent diagnostic operations to 3.
var diagSemaphore = make(chan struct{}, 3)

// acquireDiagSlot attempts to acquire a diagnostic execution slot.
// Returns false if all slots are occupied.
func acquireDiagSlot() bool {
	select {
	case diagSemaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

// releaseDiagSlot releases a diagnostic execution slot.
func releaseDiagSlot() {
	<-diagSemaphore
}

// ============================================================================
// Ping Diagnostic
// ============================================================================

// DiagPingRequest is the request body for the ping diagnostic.
type DiagPingRequest struct {
	Target  string `json:"target" example:"192.168.1.1"`
	Count   int    `json:"count,omitempty" example:"4"`
	Timeout int    `json:"timeout_ms,omitempty" example:"2000"`
}

// DiagPingResult holds ping diagnostic results.
type DiagPingResult struct {
	Target     string  `json:"target"`
	Sent       int     `json:"sent"`
	Received   int     `json:"received"`
	PacketLoss float64 `json:"packet_loss"`
	MinRTT     float64 `json:"min_rtt_ms"`
	AvgRTT     float64 `json:"avg_rtt_ms"`
	MaxRTT     float64 `json:"max_rtt_ms"`
}

// handleDiagPing performs an ICMP ping diagnostic to a target host.
//
//	@Summary		Ping a target host
//	@Description	Sends ICMP echo requests to the target and returns RTT statistics
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Param			request	body		DiagPingRequest	true	"Ping parameters"
//	@Success		200		{object}	DiagPingResult
//	@Failure		400		{object}	models.APIProblem
//	@Failure		429		{object}	models.APIProblem	"Too many concurrent diagnostics"
//	@Failure		500		{object}	models.APIProblem
//	@Security		BearerAuth
//	@Router			/recon/diag/ping [post]
func (m *Module) handleDiagPing(w http.ResponseWriter, r *http.Request) {
	if !acquireDiagSlot() {
		writeError(w, http.StatusTooManyRequests, "too many concurrent diagnostic operations, please wait")
		return
	}
	defer releaseDiagSlot()

	var req DiagPingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	if !isValidTarget(req.Target) {
		writeError(w, http.StatusBadRequest, "target must be a valid IP address or hostname")
		return
	}

	count := req.Count
	if count <= 0 || count > 10 {
		count = 4
	}
	timeoutMs := req.Timeout
	if timeoutMs <= 0 || timeoutMs > 10000 {
		timeoutMs = 2000
	}

	totalTimeout := time.Duration(count) * time.Duration(timeoutMs) * time.Millisecond
	ctx, cancel := context.WithTimeout(r.Context(), totalTimeout+5*time.Second)
	defer cancel()

	result, err := runDiagPing(ctx, req.Target, count, timeoutMs, m.logger.Named("diag-ping"))
	if err != nil {
		m.logger.Error("diagnostic ping failed",
			zap.String("target", req.Target),
			zap.Error(err),
		)
		writeError(w, http.StatusInternalServerError, "ping failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// runDiagPing executes an ICMP ping using the pro-bing library.
func runDiagPing(ctx context.Context, target string, count, timeoutMs int, logger *zap.Logger) (*DiagPingResult, error) {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return nil, fmt.Errorf("create pinger: %w", err)
	}

	pinger.Count = count
	pinger.Timeout = time.Duration(count) * time.Duration(timeoutMs) * time.Millisecond
	pinger.SetPrivileged(runtime.GOOS == "windows")

	done := make(chan struct{})
	go func() {
		defer close(done)
		if runErr := pinger.Run(); runErr != nil {
			logger.Debug("ping run error", zap.String("target", target), zap.Error(runErr))
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
		pinger.Stop()
		return nil, ctx.Err()
	}

	stats := pinger.Statistics()
	return &DiagPingResult{
		Target:     target,
		Sent:       stats.PacketsSent,
		Received:   stats.PacketsRecv,
		PacketLoss: stats.PacketLoss,
		MinRTT:     float64(stats.MinRtt.Microseconds()) / 1000.0,
		AvgRTT:     float64(stats.AvgRtt.Microseconds()) / 1000.0,
		MaxRTT:     float64(stats.MaxRtt.Microseconds()) / 1000.0,
	}, nil
}

// ============================================================================
// DNS Diagnostic
// ============================================================================

// DiagDNSRequest is the request body for DNS lookup.
type DiagDNSRequest struct {
	Target string `json:"target" example:"192.168.1.1"`
}

// DiagDNSResult holds DNS lookup results.
type DiagDNSResult struct {
	Target     string   `json:"target"`
	Hostnames  []string `json:"hostnames,omitempty"`
	IPs        []string `json:"ips,omitempty"`
	LookupType string   `json:"lookup_type"`
	DurationMs float64  `json:"duration_ms"`
}

// handleDiagDNS performs a DNS lookup for a target (reverse if IP, forward if hostname).
//
//	@Summary		DNS lookup
//	@Description	Performs a reverse DNS lookup for IP addresses or forward lookup for hostnames
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Param			request	body		DiagDNSRequest	true	"DNS lookup parameters"
//	@Success		200		{object}	DiagDNSResult
//	@Failure		400		{object}	models.APIProblem
//	@Failure		429		{object}	models.APIProblem	"Too many concurrent diagnostics"
//	@Failure		500		{object}	models.APIProblem
//	@Security		BearerAuth
//	@Router			/recon/diag/dns [post]
func (m *Module) handleDiagDNS(w http.ResponseWriter, r *http.Request) {
	if !acquireDiagSlot() {
		writeError(w, http.StatusTooManyRequests, "too many concurrent diagnostic operations, please wait")
		return
	}
	defer releaseDiagSlot()

	var req DiagDNSRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	result := &DiagDNSResult{Target: req.Target}

	// Determine if target is an IP or hostname.
	if ip := net.ParseIP(req.Target); ip != nil {
		// Reverse lookup.
		result.LookupType = "reverse"
		names, err := net.DefaultResolver.LookupAddr(ctx, req.Target)
		if err != nil {
			result.DurationMs = float64(time.Since(start).Microseconds()) / 1000.0
			// Return the result with empty hostnames rather than error,
			// since "no PTR record" is a valid result.
			m.logger.Debug("reverse DNS lookup returned no results",
				zap.String("target", req.Target),
				zap.Error(err),
			)
			writeJSON(w, http.StatusOK, result)
			return
		}
		// Strip trailing dots from FQDNs.
		hostnames := make([]string, 0, len(names))
		for _, name := range names {
			if name != "" && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}
			hostnames = append(hostnames, name)
		}
		result.Hostnames = hostnames
	} else {
		// Forward lookup.
		result.LookupType = "forward"
		addrs, err := net.DefaultResolver.LookupHost(ctx, req.Target)
		if err != nil {
			m.logger.Debug("forward DNS lookup failed",
				zap.String("target", req.Target),
				zap.Error(err),
			)
			writeError(w, http.StatusInternalServerError, "DNS lookup failed: "+err.Error())
			return
		}
		result.IPs = addrs
	}

	result.DurationMs = float64(time.Since(start).Microseconds()) / 1000.0
	writeJSON(w, http.StatusOK, result)
}

// ============================================================================
// Port Check Diagnostic
// ============================================================================

// DiagPortCheckRequest is the request body for TCP port check.
type DiagPortCheckRequest struct {
	Target  string `json:"target" example:"192.168.1.1"`
	Ports   []int  `json:"ports" example:"22,80,443"`
	Timeout int    `json:"timeout_ms,omitempty" example:"2000"`
}

// DiagPortResult holds a single port check result.
type DiagPortResult struct {
	Port   int    `json:"port"`
	Open   bool   `json:"open"`
	Banner string `json:"banner,omitempty"`
}

// DiagPortCheckResult holds all port check results.
type DiagPortCheckResult struct {
	Target     string           `json:"target"`
	Ports      []DiagPortResult `json:"ports"`
	DurationMs float64          `json:"duration_ms"`
}

// handleDiagPortCheck performs TCP port checks on a target host.
//
//	@Summary		TCP port check
//	@Description	Tests TCP connectivity to specified ports on the target host, with optional banner grabbing
//	@Tags			recon
//	@Accept			json
//	@Produce		json
//	@Param			request	body		DiagPortCheckRequest	true	"Port check parameters"
//	@Success		200		{object}	DiagPortCheckResult
//	@Failure		400		{object}	models.APIProblem
//	@Failure		429		{object}	models.APIProblem	"Too many concurrent diagnostics"
//	@Failure		500		{object}	models.APIProblem
//	@Security		BearerAuth
//	@Router			/recon/diag/port-check [post]
func (m *Module) handleDiagPortCheck(w http.ResponseWriter, r *http.Request) {
	if !acquireDiagSlot() {
		writeError(w, http.StatusTooManyRequests, "too many concurrent diagnostic operations, please wait")
		return
	}
	defer releaseDiagSlot()

	var req DiagPortCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Target == "" {
		writeError(w, http.StatusBadRequest, "target is required")
		return
	}
	if len(req.Ports) == 0 {
		writeError(w, http.StatusBadRequest, "at least one port is required")
		return
	}
	if len(req.Ports) > 20 {
		writeError(w, http.StatusBadRequest, "maximum 20 ports per request")
		return
	}

	if !isValidTarget(req.Target) {
		writeError(w, http.StatusBadRequest, "target must be a valid IP address or hostname")
		return
	}

	// Validate port range.
	for _, port := range req.Ports {
		if port < 1 || port > 65535 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid port %d: must be 1-65535", port))
			return
		}
	}

	timeoutMs := req.Timeout
	if timeoutMs <= 0 || timeoutMs > 10000 {
		timeoutMs = 2000
	}
	timeout := time.Duration(timeoutMs) * time.Millisecond

	start := time.Now()
	results := make([]DiagPortResult, 0, len(req.Ports))

	for _, port := range req.Ports {
		pr := checkPort(req.Target, port, timeout)
		results = append(results, pr)
	}

	writeJSON(w, http.StatusOK, DiagPortCheckResult{
		Target:     req.Target,
		Ports:      results,
		DurationMs: float64(time.Since(start).Microseconds()) / 1000.0,
	})
}

// checkPort tests TCP connectivity to a single port and tries to grab a banner.
func checkPort(target string, port int, timeout time.Duration) DiagPortResult {
	addr := net.JoinHostPort(target, strconv.Itoa(port))
	result := DiagPortResult{Port: port}

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return result
	}
	defer conn.Close()

	result.Open = true

	// Try to read a banner with a short timeout.
	_ = conn.SetReadDeadline(time.Now().Add(1 * time.Second))
	buf := make([]byte, 256)
	n, err := conn.Read(buf)
	if err == nil && n > 0 {
		banner := strings.TrimSpace(string(buf[:n]))
		// Only include printable ASCII banners.
		if isPrintable(banner) {
			result.Banner = banner
		}
	}

	return result
}

// ============================================================================
// Helpers
// ============================================================================

// isValidTarget checks if the target is a valid IP address or hostname.
func isValidTarget(target string) bool {
	if net.ParseIP(target) != nil {
		return true
	}
	// Basic hostname validation: must have at least one dot or be a single label,
	// no spaces, reasonable length.
	if len(target) > 253 || target == "" {
		return false
	}
	if strings.ContainsAny(target, " \t\n\r") {
		return false
	}
	return true
}

// isPrintable checks if a string contains only printable ASCII characters.
func isPrintable(s string) bool {
	for _, c := range s {
		if c < 32 || c > 126 {
			if c != '\n' && c != '\r' && c != '\t' {
				return false
			}
		}
	}
	return true
}
