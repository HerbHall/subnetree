package recon

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/huin/goupnp"
	"go.uber.org/zap"

	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/HerbHall/subnetree/pkg/plugin"
)

// UPNPDiscoverer discovers devices via UPnP/SSDP multicast queries.
// Pattern: same as MDNSListener in mdns.go.
type UPNPDiscoverer struct {
	store    *ReconStore
	bus      plugin.EventBus
	logger   *zap.Logger
	interval time.Duration

	mu   sync.Mutex
	seen map[string]time.Time // UDN -> last seen time (deduplication)
}

// NewUPNPDiscoverer creates a new UPnP discoverer that periodically queries for
// UPnP/SSDP devices and upserts discovered devices into the store.
func NewUPNPDiscoverer(store *ReconStore, bus plugin.EventBus, logger *zap.Logger, interval time.Duration) *UPNPDiscoverer {
	return &UPNPDiscoverer{
		store:    store,
		bus:      bus,
		logger:   logger,
		interval: interval,
		seen:     make(map[string]time.Time),
	}
}

// Run starts the periodic UPnP discovery loop. It blocks until ctx is cancelled.
// The caller is responsible for running this in a goroutine and calling wg.Done.
func (d *UPNPDiscoverer) Run(ctx context.Context) {
	d.logger.Info("UPnP discoverer started",
		zap.Duration("interval", d.interval),
	)

	// Run an initial scan immediately, then on a ticker.
	d.discoverAll(ctx)

	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.logger.Info("UPnP discoverer stopped")
			return
		case <-ticker.C:
			d.discoverAll(ctx)
		}
	}
}

// discoverAll performs a full UPnP/SSDP discovery sweep.
func (d *UPNPDiscoverer) discoverAll(ctx context.Context) {
	d.logger.Debug("UPnP scan starting")

	devices, err := goupnp.DiscoverDevicesCtx(ctx, "ssdp:all")
	if err != nil {
		// Context cancellation is expected during shutdown.
		if ctx.Err() != nil {
			return
		}
		d.logger.Warn("UPnP discovery failed", zap.Error(err))
		return
	}

	var discovered int
	for i := range devices {
		if ctx.Err() != nil {
			return
		}
		if d.processDevice(ctx, &devices[i]) {
			discovered++
		}
	}

	d.logger.Debug("UPnP scan complete", zap.Int("devices_found", discovered))
	d.cleanSeen()
}

// processDevice converts a UPnP discovery result into a device and upserts it.
// Returns true if the device was new or updated (not deduplicated).
func (d *UPNPDiscoverer) processDevice(ctx context.Context, maybe *goupnp.MaybeRootDevice) bool {
	if maybe.Err != nil {
		d.logger.Debug("UPnP device probe error",
			zap.String("usn", maybe.USN),
			zap.Error(maybe.Err),
		)
		return false
	}

	if maybe.Root == nil {
		return false
	}

	dev := &maybe.Root.Device
	udn := dev.UDN
	if udn == "" {
		// Fall back to USN if UDN is not available.
		udn = maybe.USN
	}
	if udn == "" {
		return false
	}

	// Deduplicate: skip if we've seen this UDN within the current interval.
	if d.recentlySeen(udn) {
		return false
	}
	d.markSeen(udn)

	// Extract the IP from the device location URL.
	ip := ""
	if maybe.Location != nil {
		ip = maybe.Location.Hostname()
	}
	if ip == "" {
		return false
	}

	hostname := dev.FriendlyName
	if hostname == "" {
		hostname = dev.ModelName
	}

	device := &models.Device{
		Hostname:        hostname,
		IPAddresses:     []string{ip},
		Status:          models.DeviceStatusOnline,
		DiscoveryMethod: models.DiscoveryUPnP,
		DeviceType:      inferDeviceTypeFromUPnP(dev.DeviceType),
		Manufacturer:    dev.Manufacturer,
	}

	created, err := d.store.UpsertDevice(ctx, device)
	if err != nil {
		d.logger.Warn("UPnP device upsert failed",
			zap.String("ip", ip),
			zap.String("hostname", hostname),
			zap.Error(err),
		)
		return false
	}

	topic := TopicDeviceUpdated
	if created {
		topic = TopicDeviceDiscovered
	}
	d.publishEvent(ctx, topic, DeviceEvent{
		Device: device,
	})

	d.logger.Info("UPnP device discovered",
		zap.String("ip", ip),
		zap.String("hostname", hostname),
		zap.String("udn", udn),
		zap.String("manufacturer", dev.Manufacturer),
		zap.String("model", dev.ModelName),
		zap.Bool("new", created),
	)

	return true
}

// recentlySeen returns true if the UDN was seen within the current scan interval.
func (d *UPNPDiscoverer) recentlySeen(udn string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	lastSeen, ok := d.seen[udn]
	if !ok {
		return false
	}
	return time.Since(lastSeen) < d.interval
}

// markSeen records the UDN as recently seen.
func (d *UPNPDiscoverer) markSeen(udn string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.seen[udn] = time.Now()
}

// cleanSeen removes entries older than 2x the scan interval.
func (d *UPNPDiscoverer) cleanSeen() {
	d.mu.Lock()
	defer d.mu.Unlock()
	cutoff := time.Now().Add(-2 * d.interval)
	for udn, t := range d.seen {
		if t.Before(cutoff) {
			delete(d.seen, udn)
		}
	}
}

// publishEvent publishes an event to the event bus.
func (d *UPNPDiscoverer) publishEvent(ctx context.Context, topic string, payload any) {
	if d.bus == nil {
		return
	}
	d.bus.PublishAsync(ctx, plugin.Event{
		Topic:     topic,
		Source:    "recon",
		Timestamp: time.Now(),
		Payload:   payload,
	})
}

// inferDeviceTypeFromUPnP guesses the SubNetree device type from the UPnP device type URN.
func inferDeviceTypeFromUPnP(deviceType string) models.DeviceType {
	dt := strings.ToLower(deviceType)

	switch {
	case strings.Contains(dt, "mediarenderer") ||
		strings.Contains(dt, "mediaserver"):
		return models.DeviceTypeNAS

	case strings.Contains(dt, "printer"):
		return models.DeviceTypePrinter

	case strings.Contains(dt, "internetgateway") ||
		strings.Contains(dt, "wandevice") ||
		strings.Contains(dt, "wanconnectiondevice"):
		return models.DeviceTypeRouter

	case strings.Contains(dt, "wlanaccess"):
		return models.DeviceTypeAccessPoint

	case strings.Contains(dt, "digitalSecuritycamera") ||
		strings.Contains(dt, "digitalsecuritycamera"):
		return models.DeviceTypeCamera

	case strings.Contains(dt, "lightingcontrols") ||
		strings.Contains(dt, "binarylight") ||
		strings.Contains(dt, "dimmablelight") ||
		strings.Contains(dt, "hvac") ||
		strings.Contains(dt, "sensormanagement"):
		return models.DeviceTypeIoT

	default:
		return models.DeviceTypeUnknown
	}
}
