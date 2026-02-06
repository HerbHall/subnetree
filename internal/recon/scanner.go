package recon

import (
	"context"
	"net"
	"time"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
	"go.uber.org/zap"
)

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

		device := &models.Device{
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
