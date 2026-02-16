package recon

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"go.uber.org/zap"
)

// HostResult holds the result of probing a single host.
type HostResult struct {
	IP     string
	RTT    time.Duration
	Alive  bool
	Method string // "icmp" or "arp"
	TTL    int    // IP TTL from response (0 if unknown)
}

// ICMPScanner pings hosts in a subnet using ICMP.
type ICMPScanner struct {
	pingTimeout time.Duration
	pingCount   int
	concurrency int
	logger      *zap.Logger
}

// NewICMPScanner creates a new ICMP scanner.
func NewICMPScanner(cfg ReconConfig, logger *zap.Logger) *ICMPScanner {
	return &ICMPScanner{
		pingTimeout: cfg.PingTimeout,
		pingCount:   cfg.PingCount,
		concurrency: cfg.Concurrency,
		logger:      logger,
	}
}

// Scan pings all hosts in the given subnet and sends alive hosts to results.
// The caller must close the results channel after Scan returns.
func (s *ICMPScanner) Scan(ctx context.Context, subnet *net.IPNet, results chan<- HostResult) error {
	hosts := expandSubnet(subnet)
	if len(hosts) == 0 {
		return fmt.Errorf("no hosts in subnet %s", subnet)
	}

	s.logger.Info("starting ICMP scan",
		zap.String("subnet", subnet.String()),
		zap.Int("hosts", len(hosts)),
		zap.Int("concurrency", s.concurrency),
	)

	// Semaphore for bounded concurrency.
	sem := make(chan struct{}, s.concurrency)
	errCh := make(chan error, 1)

	// Determine if we need privileged mode.
	privileged := runtime.GOOS == "windows"

	for _, ip := range hosts {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}

		go func(ip string) {
			defer func() { <-sem }()

			alive, rtt, ttl := s.pingHost(ctx, ip, privileged)
			if alive {
				select {
				case results <- HostResult{IP: ip, RTT: rtt, Alive: true, Method: "icmp", TTL: ttl}:
				case <-ctx.Done():
				}
			}
		}(ip)
	}

	// Wait for all goroutines to finish.
	for i := 0; i < s.concurrency; i++ {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

// pingHost pings a single host and returns whether it is alive.
func (s *ICMPScanner) pingHost(ctx context.Context, ip string, privileged bool) (alive bool, rtt time.Duration, ttl int) {
	pinger, err := probing.NewPinger(ip)
	if err != nil {
		s.logger.Debug("failed to create pinger", zap.String("ip", ip), zap.Error(err))
		return false, 0, 0
	}

	pinger.Count = s.pingCount
	pinger.Timeout = s.pingTimeout
	pinger.SetPrivileged(privileged)

	// Capture TTL from first received packet.
	var receivedTTL int
	pinger.OnRecv = func(pkt *probing.Packet) {
		if receivedTTL == 0 {
			receivedTTL = pkt.TTL
		}
	}

	// Run with context for cancellation support.
	done := make(chan struct{})
	go func() {
		defer close(done)
		if runErr := pinger.Run(); runErr != nil {
			s.logger.Debug("ping failed", zap.String("ip", ip), zap.Error(runErr))
		}
	}()

	select {
	case <-done:
	case <-ctx.Done():
		pinger.Stop()
		return false, 0, 0
	}

	stats := pinger.Statistics()
	if stats.PacketsRecv > 0 {
		return true, stats.AvgRtt, receivedTTL
	}
	return false, 0, 0
}

// InferOSFromTTL returns an OS hint based on the ICMP TTL value.
// TTL values are decremented at each hop, so we check ranges.
func InferOSFromTTL(ttl int) string {
	switch {
	case ttl == 0:
		return ""
	case ttl >= 225: // 255 minus up to 30 hops
		return "network_equipment" // Cisco IOS, Juniper JUNOS, most switches/routers
	case ttl >= 110 && ttl <= 128:
		return "windows"
	case ttl >= 35 && ttl <= 64:
		return "linux" // Also macOS, FreeBSD, Linux-based switches (Ubiquiti)
	default:
		return ""
	}
}

// expandSubnet returns all host IPs in a subnet (excluding network and broadcast).
func expandSubnet(subnet *net.IPNet) []string {
	ones, bits := subnet.Mask.Size()
	if ones == 0 && bits == 0 {
		return nil
	}

	// Limit to /16 (65534 hosts) to prevent accidental huge scans.
	hostBits := bits - ones
	if hostBits > 16 {
		return nil
	}

	var hosts []string
	totalHosts := 1 << hostBits

	ip := make(net.IP, len(subnet.IP))
	copy(ip, subnet.IP)

	for i := 1; i < totalHosts-1; i++ {
		next := incrementIP(subnet.IP, i)
		if subnet.Contains(next) {
			hosts = append(hosts, next.String())
		}
	}

	return hosts
}

// incrementIP adds offset to a base IP.
func incrementIP(base net.IP, offset int) net.IP {
	ip := make(net.IP, len(base))
	copy(ip, base)

	// Work with 4-byte IP for simplicity.
	ip = ip.To4()
	if ip == nil {
		return nil
	}

	carry := offset
	for i := 3; i >= 0; i-- {
		val := int(ip[i]) + carry
		ip[i] = byte(val % 256)
		carry = val / 256
		if carry == 0 {
			break
		}
	}
	return ip
}
