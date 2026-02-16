package recon

import (
	"context"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
)

// PortScanResult holds the results of a port scan for a single host.
type PortScanResult struct {
	IP        string
	OpenPorts []int
}

// InfrastructurePorts are TCP ports commonly found on network infrastructure devices.
var InfrastructurePorts = []int{
	22,   // SSH
	23,   // Telnet
	80,   // HTTP management UI
	161,  // SNMP (checked via TCP; UDP check would require gosnmp)
	443,  // HTTPS management UI
	8080, // HTTP alt (Ubiquiti inform, some Netgear)
	8291, // Winbox (MikroTik RouterOS)
	8443, // HTTPS alt (Ubiquiti UniFi)
}

// PortScanner performs targeted TCP port scans on network devices.
type PortScanner struct {
	timeout     time.Duration
	concurrency int
	logger      *zap.Logger
}

// NewPortScanner creates a new port scanner.
func NewPortScanner(timeout time.Duration, concurrency int, logger *zap.Logger) *PortScanner {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	if concurrency <= 0 {
		concurrency = 10
	}
	return &PortScanner{
		timeout:     timeout,
		concurrency: concurrency,
		logger:      logger,
	}
}

// ScanPorts checks which of the given ports are open on the target IP.
func (s *PortScanner) ScanPorts(ctx context.Context, ip string, ports []int) *PortScanResult {
	result := &PortScanResult{IP: ip}

	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.concurrency)

	for _, port := range ports {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(p int) {
			defer wg.Done()
			defer func() { <-sem }()

			if s.isPortOpen(ctx, ip, p) {
				mu.Lock()
				result.OpenPorts = append(result.OpenPorts, p)
				mu.Unlock()
			}
		}(port)
	}
	wg.Wait()

	// Sort for deterministic output.
	sort.Ints(result.OpenPorts)

	s.logger.Debug("port scan complete",
		zap.String("ip", ip),
		zap.Ints("open", result.OpenPorts),
	)

	return result
}

// isPortOpen attempts a TCP connection to the given port.
func (s *PortScanner) isPortOpen(ctx context.Context, ip string, port int) bool {
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	d := net.Dialer{Timeout: s.timeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}
