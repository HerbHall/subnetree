package recon

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

// dnsTimeout is the maximum time to wait for a reverse DNS lookup.
const dnsTimeout = 500 * time.Millisecond

// PingScanner probes hosts via ICMP and sends results to a channel.
type PingScanner interface {
	Scan(ctx context.Context, subnet *net.IPNet, results chan<- HostResult) error
}

// ARPTableReader reads the system ARP table.
type ARPTableReader interface {
	ReadTable(ctx context.Context) map[string]string
}

// OUIResolver resolves MAC address prefixes to manufacturer names.
type OUIResolver interface {
	Lookup(mac string) string
}

// ScanOrchestrator coordinates ICMP scanning, ARP enrichment, OUI lookup,
// device persistence, and event publishing.
type ScanOrchestrator struct {
	store  *ReconStore
	bus    plugin.EventBus
	oui    OUIResolver
	pinger PingScanner
	arp    ARPTableReader
	logger *zap.Logger
}

// NewScanOrchestrator creates a new orchestrator.
func NewScanOrchestrator(
	store *ReconStore,
	bus plugin.EventBus,
	oui OUIResolver,
	pinger PingScanner,
	arp ARPTableReader,
	logger *zap.Logger,
) *ScanOrchestrator {
	return &ScanOrchestrator{
		store:  store,
		bus:    bus,
		oui:    oui,
		pinger: pinger,
		arp:    arp,
		logger: logger,
	}
}

// RunScan executes a full network scan for the given subnet.
func (o *ScanOrchestrator) RunScan(ctx context.Context, scanID, subnet string) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		o.logger.Error("invalid subnet", zap.String("subnet", subnet), zap.Error(err))
		_ = o.store.UpdateScanError(ctx, scanID, "invalid subnet: "+err.Error())
		return
	}

	// Emit scan started event.
	o.publishEvent(ctx, TopicScanStarted, &models.ScanResult{
		ID: scanID, Subnet: subnet, Status: "running",
		StartedAt: time.Now().UTC().Format(time.RFC3339),
	})

	// Run ICMP scan.
	results := make(chan HostResult, 256)
	scanDone := make(chan error, 1)
	go func() {
		scanDone <- o.pinger.Scan(ctx, ipNet, results)
		close(results)
	}()

	// Collect alive hosts.
	var alive []HostResult
	for r := range results {
		if r.Alive {
			alive = append(alive, r)
		}
	}

	// Emit scan progress event (ping phase complete).
	ones, bits := ipNet.Mask.Size()
	subnetSize := 1<<(bits-ones) - 2 // subtract network + broadcast
	if subnetSize < 1 {
		subnetSize = 1
	}
	o.publishEvent(ctx, TopicScanProgress, &ScanProgressEvent{
		ScanID:     scanID,
		HostsAlive: len(alive),
		SubnetSize: subnetSize,
	})

	// Check for scan error. Use background context for DB cleanup since
	// the scan context may already be cancelled.
	cleanupCtx := context.Background()
	if scanErr := <-scanDone; scanErr != nil {
		if ctx.Err() != nil {
			o.logger.Info("scan cancelled", zap.String("scan_id", scanID))
			_ = o.store.UpdateScanError(cleanupCtx, scanID, "cancelled")
			return
		}
		o.logger.Error("ICMP scan error", zap.Error(scanErr))
		_ = o.store.UpdateScanError(cleanupCtx, scanID, scanErr.Error())
		return
	}

	// Read ARP table for MAC addresses.
	arpTable := map[string]string{}
	if o.arp != nil {
		arpTable = o.arp.ReadTable(ctx)
	}

	// Process each alive host.
	var onlineCount int
	var totalCount int
	for _, host := range alive {
		if ctx.Err() != nil {
			_ = o.store.UpdateScanError(cleanupCtx, scanID, "cancelled")
			return
		}

		totalCount++
		onlineCount++

		mac := arpTable[host.IP]
		manufacturer := ""
		discoveryMethod := models.DiscoveryICMP
		if mac != "" {
			manufacturer = o.oui.Lookup(mac)
			discoveryMethod = models.DiscoveryARP
		}

		hostname := o.resolveHostname(host.IP)

		device := &models.Device{
			Hostname:        hostname,
			IPAddresses:     []string{host.IP},
			MACAddress:      mac,
			Manufacturer:    manufacturer,
			DeviceType:      models.DeviceTypeUnknown,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: discoveryMethod,
		}

		created, err := o.store.UpsertDevice(ctx, device)
		if err != nil {
			o.logger.Error("failed to upsert device", zap.String("ip", host.IP), zap.Error(err))
			continue
		}

		// Link device to scan.
		if err := o.store.LinkScanDevice(ctx, scanID, device.ID); err != nil {
			o.logger.Error("failed to link scan device", zap.Error(err))
		}

		// Emit device event with scan ID.
		devEvent := &DeviceEvent{ScanID: scanID, Device: device}
		if created {
			o.publishEvent(ctx, TopicDeviceDiscovered, devEvent)
		} else {
			o.publishEvent(ctx, TopicDeviceUpdated, devEvent)
		}
	}

	// Infer topology links from ARP data.
	o.inferTopologyLinks(ctx, subnet, alive)

	// Detect service movements between scans.
	o.detectAndPublishServiceMovements(ctx, alive)

	// Update scan record.
	scan := &models.ScanResult{
		ID:      scanID,
		Subnet:  subnet,
		Status:  "completed",
		EndedAt: time.Now().UTC().Format(time.RFC3339),
		Total:   totalCount,
		Online:  onlineCount,
	}
	if err := o.store.UpdateScan(ctx, scan); err != nil {
		o.logger.Error("failed to update scan", zap.Error(err))
	}

	o.publishEvent(ctx, TopicScanCompleted, scan)
	o.logger.Info("scan completed",
		zap.String("scan_id", scanID),
		zap.Int("total", totalCount),
		zap.Int("online", onlineCount),
	)
}

// resolveHostname performs a reverse DNS lookup for the given IP address.
// Returns an empty string if the lookup fails or times out.
func (o *ScanOrchestrator) resolveHostname(ip string) string {
	ctx, cancel := context.WithTimeout(context.Background(), dnsTimeout)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

// inferTopologyLinks creates topology edges between discovered devices and the
// subnet gateway. The gateway is assumed to be the first usable IP in the CIDR.
func (o *ScanOrchestrator) inferTopologyLinks(ctx context.Context, subnet string, hosts []HostResult) {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return
	}
	gatewayIP := firstUsableIP(ipNet)

	gateway, err := o.store.GetDeviceByIP(ctx, gatewayIP)
	if err != nil || gateway == nil {
		o.logger.Debug("gateway device not found, skipping link inference",
			zap.String("gateway_ip", gatewayIP))
		return
	}

	var linkCount int
	for _, host := range hosts {
		if host.IP == gatewayIP {
			continue
		}
		device, err := o.store.GetDeviceByIP(ctx, host.IP)
		if err != nil || device == nil {
			continue
		}
		link := &TopologyLink{
			SourceDeviceID: device.ID,
			TargetDeviceID: gateway.ID,
			LinkType:       "arp",
		}
		if err := o.store.UpsertTopologyLink(ctx, link); err != nil {
			o.logger.Error("failed to upsert topology link", zap.Error(err))
			continue
		}
		linkCount++
	}
	o.logger.Info("topology links inferred",
		zap.Int("count", linkCount),
		zap.String("gateway", gatewayIP))
}

// firstUsableIP returns the first usable host address in a subnet
// (network address + 1).
func firstUsableIP(ipNet *net.IPNet) string {
	ip := make(net.IP, len(ipNet.IP))
	copy(ip, ipNet.IP)
	ip = ip.To4()
	if ip == nil {
		return ""
	}
	ip[len(ip)-1]++
	return ip.String()
}

// detectAndPublishServiceMovements compares the current scan's service map
// against the previous scan and publishes events for any detected movements.
func (o *ScanOrchestrator) detectAndPublishServiceMovements(ctx context.Context, alive []HostResult) {
	previous, err := o.store.GetPreviousServiceMap(ctx)
	if err != nil {
		o.logger.Error("failed to get previous service map", zap.Error(err))
		return
	}

	// Build current service map from alive hosts.
	// Note: Until port scanning is implemented in the scan pipeline, the
	// current map will also be empty. This wiring is ready for when port
	// data becomes available.
	current := make(map[string][]int)
	for _, host := range alive {
		device, devErr := o.store.GetDeviceByIP(ctx, host.IP)
		if devErr != nil || device == nil {
			continue
		}
		// Ports will be populated once the scan pipeline includes port scanning.
		current[device.ID] = []int{}
	}

	movements := detectServiceMovements(previous, current)

	for i := range movements {
		if saveErr := o.store.SaveServiceMovement(ctx, movements[i]); saveErr != nil {
			o.logger.Error("failed to save service movement",
				zap.Int("port", movements[i].Port),
				zap.Error(saveErr),
			)
			continue
		}
		o.publishEvent(ctx, TopicServiceMoved, ServiceMovedEvent{Movement: movements[i]})
		o.logger.Info("service movement detected",
			zap.Int("port", movements[i].Port),
			zap.String("from", movements[i].FromDevice),
			zap.String("to", movements[i].ToDevice),
			zap.String("service", movements[i].ServiceName),
		)
	}
}

func (o *ScanOrchestrator) publishEvent(ctx context.Context, topic string, payload any) {
	if o.bus == nil {
		return
	}
	o.bus.PublishAsync(ctx, plugin.Event{
		Topic:     topic,
		Source:    "recon",
		Timestamp: time.Now(),
		Payload:   payload,
	})
}
