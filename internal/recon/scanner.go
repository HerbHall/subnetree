package recon

import (
	"context"
	"encoding/json"
	"fmt"
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

// SNMPWalker walks FDB tables on SNMP-enabled switches.
type SNMPWalker interface {
	WalkFDB(ctx context.Context, target string, cred CredentialAccessor, credID string) ([]FDBEntry, error)
}

// CredentialLookup finds SNMP credentials for a device.
type CredentialLookup interface {
	FindSNMPCredentialForDevice(ctx context.Context, deviceID string) (credID string, err error)
}

// ScanOrchestrator coordinates ICMP scanning, ARP enrichment, OUI lookup,
// device persistence, and event publishing.
type ScanOrchestrator struct {
	store       *ReconStore
	bus         plugin.EventBus
	oui         OUIResolver
	pinger      PingScanner
	arp         ARPTableReader
	snmpWalker  SNMPWalker
	wifiScanner WifiScanner
	credLookup  CredentialLookup
	credAccess  CredentialAccessor
	logger      *zap.Logger
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

// SetSNMPWalker configures the SNMP FDB walker used during scan post-processing.
func (o *ScanOrchestrator) SetSNMPWalker(w SNMPWalker) {
	o.snmpWalker = w
}

// SetWifiScanner configures the WiFi scanner used to discover nearby access points.
func (o *ScanOrchestrator) SetWifiScanner(ws WifiScanner) {
	o.wifiScanner = ws
}

// SetCredentialLookup configures the credential lookup used for FDB walks.
func (o *ScanOrchestrator) SetCredentialLookup(cl CredentialLookup) {
	o.credLookup = cl
}

// SetCredentialAccessor configures the credential accessor used for FDB walks.
func (o *ScanOrchestrator) SetCredentialAccessor(ca CredentialAccessor) {
	o.credAccess = ca
}

// scanStage represents a named post-scan processing stage.
type scanStage struct {
	name string
	run  func(ctx context.Context)
}

// runStages executes stages sequentially, checking for cancellation between each.
func (o *ScanOrchestrator) runStages(ctx context.Context, stages []scanStage) {
	for _, stage := range stages {
		if ctx.Err() != nil {
			o.logger.Warn("scan cancelled, skipping remaining stages",
				zap.String("skipped", stage.name))
			return
		}
		stage.run(ctx)
	}
}

// RunScan executes a full network scan for the given subnet.
func (o *ScanOrchestrator) RunScan(ctx context.Context, scanID, subnet string) {
	scanStart := time.Now()

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

	// Calculate subnet size for progress reporting.
	ones, bits := ipNet.Mask.Size()
	subnetSize := 1<<(bits-ones) - 2 // subtract network + broadcast
	if subnetSize < 1 {
		subnetSize = 1
	}

	// Read ARP table upfront so we can enrich devices as they arrive.
	arpTable := map[string]string{}
	if o.arp != nil {
		arpTable = o.arp.ReadTable(ctx)
	}

	// Run ICMP scan.
	results := make(chan HostResult, 256)
	scanDone := make(chan error, 1)
	go func() {
		scanDone <- o.pinger.Scan(ctx, ipNet, results)
		close(results)
	}()

	// Process each alive host as it arrives, streaming device events in
	// real time rather than waiting for the full sweep to complete.
	alive := make([]HostResult, 0, subnetSize)
	var onlineCount int
	var totalCount int
	var devicesCreated int
	var devicesUpdated int
	for r := range results {
		if !r.Alive {
			continue
		}
		alive = append(alive, r)

		if ctx.Err() != nil {
			continue // drain channel but skip processing
		}

		totalCount++
		onlineCount++

		mac := arpTable[r.IP]
		manufacturer := ""
		discoveryMethod := models.DiscoveryICMP
		if mac != "" {
			manufacturer = o.oui.Lookup(mac)
			discoveryMethod = models.DiscoveryARP
		}

		hostname := o.resolveHostname(r.IP)

		deviceType := models.DeviceTypeUnknown
		if manufacturer != "" {
			deviceType = ClassifyByManufacturer(manufacturer)
		}

		device := &models.Device{
			Hostname:        hostname,
			IPAddresses:     []string{r.IP},
			MACAddress:      mac,
			Manufacturer:    manufacturer,
			DeviceType:      deviceType,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: discoveryMethod,
		}

		created, upsertErr := o.store.UpsertDevice(ctx, device)
		if upsertErr != nil {
			o.logger.Error("failed to upsert device", zap.String("ip", r.IP), zap.Error(upsertErr))
			continue
		}

		if created {
			devicesCreated++
		} else {
			devicesUpdated++
		}

		// Link device to scan.
		if linkErr := o.store.LinkScanDevice(ctx, scanID, device.ID); linkErr != nil {
			o.logger.Error("failed to link scan device", zap.Error(linkErr))
		}

		// Emit device event with scan ID immediately.
		devEvent := &DeviceEvent{ScanID: scanID, Device: device}
		if created {
			o.publishEvent(ctx, TopicDeviceDiscovered, devEvent)
		} else {
			o.publishEvent(ctx, TopicDeviceUpdated, devEvent)
		}

		// Emit incremental progress so the UI can show a running count.
		o.publishEvent(ctx, TopicScanProgress, &ScanProgressEvent{
			ScanID:     scanID,
			HostsAlive: len(alive),
			SubnetSize: subnetSize,
		})
	}

	// Ping + enrichment happen together in the streaming loop above.
	enrichDone := time.Now()

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

	// Run post-scan processing stages.
	o.runStages(ctx, []scanStage{
		{"wifi-scan", func(ctx context.Context) { o.scanWifiNetworks(ctx) }},
		{"port-scan", func(ctx context.Context) { o.portScanInfraDevices(ctx, alive, arpTable) }},
		{"classify", func(ctx context.Context) { o.classifyDevices(ctx, alive, arpTable) }},
		{"unmanaged-switch", func(ctx context.Context) { o.detectUnmanagedSwitches(ctx, alive, arpTable) }},
		{"fdb-walk", func(ctx context.Context) { o.walkSwitchFDBTables(ctx) }},
		{"wifi-heuristic", func(ctx context.Context) { o.analyzeWiFiConnections(ctx) }},
		{"topology-links", func(ctx context.Context) { o.inferTopologyLinks(ctx, subnet, alive) }},
		{"hierarchy", func(ctx context.Context) { o.inferHierarchy(ctx) }},
		{"service-movements", func(ctx context.Context) { o.detectAndPublishServiceMovements(ctx, alive) }},
	})

	postDone := time.Now()

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

	// Save scan metrics. Ping and enrichment are combined in the streaming
	// model, so pingPhaseMs equals the full streaming loop duration.
	metrics := &models.ScanMetrics{
		ScanID:         scanID,
		DurationMs:     time.Since(scanStart).Milliseconds(),
		PingPhaseMs:    enrichDone.Sub(scanStart).Milliseconds(),
		EnrichPhaseMs:  0,
		PostProcessMs:  postDone.Sub(enrichDone).Milliseconds(),
		HostsScanned:   subnetSize,
		HostsAlive:     len(alive),
		DevicesCreated: devicesCreated,
		DevicesUpdated: devicesUpdated,
	}
	if saveErr := o.store.SaveScanMetrics(ctx, metrics); saveErr != nil {
		o.logger.Error("failed to save scan metrics", zap.Error(saveErr))
	}

	o.publishEvent(ctx, TopicScanCompleted, scan)
	o.logger.Info("scan completed",
		zap.String("scan_id", scanID),
		zap.Int("total", totalCount),
		zap.Int("online", onlineCount),
		zap.Int64("duration_ms", metrics.DurationMs),
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

// portScanInfraDevices performs targeted port scanning on devices identified
// as potential infrastructure by OUI classification.
func (o *ScanOrchestrator) portScanInfraDevices(ctx context.Context, alive []HostResult, arpTable map[string]string) {
	scanner := NewPortScanner(2*time.Second, 10, o.logger)

	var scannedCount int
	for _, host := range alive {
		if ctx.Err() != nil {
			return
		}

		// Only scan devices with infrastructure OUI classification.
		mac := arpTable[host.IP]
		if mac == "" {
			continue
		}
		manufacturer := ""
		if o.oui != nil {
			manufacturer = o.oui.Lookup(mac)
		}
		ouiType := ClassifyByManufacturer(manufacturer)
		if !IsInfrastructureOUI(ouiType) {
			continue
		}

		result := scanner.ScanPorts(ctx, host.IP, InfrastructurePorts)
		if len(result.OpenPorts) == 0 {
			continue
		}

		portType := ClassifyByPorts(result.OpenPorts)
		if portType == models.DeviceTypeUnknown {
			continue
		}

		// Update device type if port fingerprinting gives a more specific result.
		device, err := o.store.GetDeviceByIP(ctx, host.IP)
		if err != nil || device == nil {
			continue
		}

		// Only upgrade from unknown or generic OUI classification.
		if device.DeviceType == models.DeviceTypeUnknown || device.DeviceType == ouiType {
			if updateErr := o.store.UpdateDeviceType(ctx, device.ID, portType); updateErr != nil {
				o.logger.Error("failed to update device type from port scan",
					zap.String("device_id", device.ID),
					zap.Error(updateErr))
				continue
			}
			scannedCount++
		}
	}

	if scannedCount > 0 {
		o.logger.Info("port fingerprinting updated devices",
			zap.Int("count", scannedCount))
	}
}

// classifyDevices runs the composite classifier on all discovered devices,
// combining OUI, SNMP, port fingerprint, and TTL signals.
func (o *ScanOrchestrator) classifyDevices(ctx context.Context, alive []HostResult, arpTable map[string]string) {
	var classifiedCount int
	for _, host := range alive {
		if ctx.Err() != nil {
			return
		}

		device, err := o.store.GetDeviceByIP(ctx, host.IP)
		if err != nil || device == nil {
			continue
		}

		signals := &DeviceSignals{
			TTL: host.TTL,
		}

		// Populate OUI signal.
		mac := arpTable[host.IP]
		if mac != "" && o.oui != nil {
			signals.Manufacturer = o.oui.Lookup(mac)
			signals.OUIDeviceType = ClassifyByManufacturer(signals.Manufacturer)
		}

		// TTL OS hint.
		if host.TTL > 0 {
			signals.OSHint = InferOSFromTTL(host.TTL)
		}

		result := Classify(signals)

		// Only update if classifier found something and it's better than current.
		if result.DeviceType == models.DeviceTypeUnknown {
			continue
		}
		if result.Confidence < 25 {
			continue
		}

		// Don't downgrade from a more specific manual or SNMP classification.
		if device.DeviceType != models.DeviceTypeUnknown && device.DeviceType != result.DeviceType {
			// Even if we don't change the type, update classification metadata
			// if the new confidence is higher.
			if result.Confidence > device.ClassificationConfidence {
				signalsJSON, _ := json.Marshal(result.Signals)
				_ = o.store.UpdateDeviceClassification(ctx, device.ID, device.DeviceType, result.Confidence, result.Source, string(signalsJSON))
			}
			continue
		}

		signalsJSON, _ := json.Marshal(result.Signals)
		if device.DeviceType == models.DeviceTypeUnknown {
			if updateErr := o.store.UpdateDeviceClassification(ctx, device.ID, result.DeviceType, result.Confidence, result.Source, string(signalsJSON)); updateErr != nil {
				o.logger.Error("failed to update device type from classifier",
					zap.String("device_id", device.ID),
					zap.Error(updateErr))
				continue
			}
			classifiedCount++
		}
	}

	if classifiedCount > 0 {
		o.logger.Info("composite classifier updated devices",
			zap.Int("count", classifiedCount))
	}
}

// detectUnmanagedSwitches infers potential unmanaged switches from ARP/MAC
// patterns after classification. Devices with infrastructure vendor OUIs that
// have no SNMP, no open ports, and remain unclassified are candidates.
func (o *ScanOrchestrator) detectUnmanagedSwitches(ctx context.Context, alive []HostResult, arpTable map[string]string) {
	infos := make([]UnmanagedDeviceInfo, 0, len(alive))

	for _, host := range alive {
		if ctx.Err() != nil {
			return
		}

		device, err := o.store.GetDeviceByIP(ctx, host.IP)
		if err != nil || device == nil {
			continue
		}

		mac := arpTable[host.IP]
		manufacturer := ""
		if mac != "" && o.oui != nil {
			manufacturer = o.oui.Lookup(mac)
		}

		infos = append(infos, UnmanagedDeviceInfo{
			DeviceID:     device.ID,
			IP:           host.IP,
			MAC:          mac,
			Manufacturer: manufacturer,
			DeviceType:   device.DeviceType,
			HasSNMP:      device.DiscoveryMethod == models.DiscoverySNMP,
			HasOpenPorts: false, // Port scan results stored on device are not yet exposed; rely on DeviceType.
		})
	}

	candidates := DetectUnmanagedSwitches(infos)

	var updatedCount int
	for i := range candidates {
		if candidates[i].Confidence < 15 {
			continue
		}

		if updateErr := o.store.UpdateDeviceType(ctx, candidates[i].DeviceID, models.DeviceTypeSwitch); updateErr != nil {
			o.logger.Error("failed to update device type for unmanaged switch candidate",
				zap.String("device_id", candidates[i].DeviceID),
				zap.Error(updateErr))
			continue
		}

		// Re-fetch device for the event payload.
		device, err := o.store.GetDeviceByIP(ctx, candidates[i].IP)
		if err == nil && device != nil {
			o.publishEvent(ctx, TopicDeviceUpdated, &DeviceEvent{Device: device})
		}

		o.logger.Debug("inferred unmanaged switch",
			zap.String("device_id", candidates[i].DeviceID),
			zap.String("ip", candidates[i].IP),
			zap.String("reason", candidates[i].Reason),
			zap.Int("confidence", candidates[i].Confidence),
		)
		updatedCount++
	}

	if updatedCount > 0 {
		o.logger.Info("unmanaged switch detection updated devices",
			zap.Int("count", updatedCount))
	}
}

// walkSwitchFDBTables queries all classified switches for their BRIDGE-MIB
// forwarding database and creates topology links from FDB entries.
func (o *ScanOrchestrator) walkSwitchFDBTables(ctx context.Context) {
	if o.snmpWalker == nil || o.credLookup == nil || o.credAccess == nil {
		return
	}

	switches, _, err := o.store.ListDevices(ctx, ListDevicesOptions{DeviceType: "switch", Limit: 500})
	if err != nil {
		o.logger.Error("failed to list switches for FDB walk", zap.Error(err))
		return
	}

	var totalLinks int
	for i := range switches {
		if ctx.Err() != nil {
			return
		}

		sw := &switches[i]
		if sw.ClassificationConfidence < 50 {
			continue
		}
		if len(sw.IPAddresses) == 0 {
			continue
		}

		credID, credErr := o.credLookup.FindSNMPCredentialForDevice(ctx, sw.ID)
		if credErr != nil || credID == "" {
			o.logger.Debug("no SNMP credential for switch, skipping FDB walk",
				zap.String("device_id", sw.ID),
				zap.String("ip", sw.IPAddresses[0]),
			)
			continue
		}

		entries, walkErr := o.snmpWalker.WalkFDB(ctx, sw.IPAddresses[0], o.credAccess, credID)
		if walkErr != nil {
			o.logger.Warn("FDB walk failed for switch",
				zap.String("device_id", sw.ID),
				zap.String("ip", sw.IPAddresses[0]),
				zap.Error(walkErr),
			)
			continue
		}

		if len(entries) == 0 {
			continue
		}

		// Remove stale FDB links for this switch before inserting fresh ones.
		if removeErr := o.store.RemoveFDBLinksForDevice(ctx, sw.ID); removeErr != nil {
			o.logger.Error("failed to remove old FDB links",
				zap.String("device_id", sw.ID),
				zap.Error(removeErr),
			)
		}

		for j := range entries {
			targetDevice, devErr := o.store.GetDeviceByMAC(ctx, entries[j].MACAddress)
			if devErr != nil || targetDevice == nil {
				continue
			}
			if targetDevice.ID == sw.ID {
				continue
			}

			portName := entries[j].IfName
			if portName == "" && entries[j].IfIndex > 0 {
				portName = fmt.Sprintf("ifIndex:%d", entries[j].IfIndex)
			}

			link := &TopologyLink{
				SourceDeviceID: sw.ID,
				TargetDeviceID: targetDevice.ID,
				SourcePort:     portName,
				LinkType:       "fdb",
			}
			if upsertErr := o.store.UpsertTopologyLink(ctx, link); upsertErr != nil {
				o.logger.Error("failed to upsert FDB topology link",
					zap.String("switch", sw.ID),
					zap.String("target", targetDevice.ID),
					zap.Error(upsertErr),
				)
				continue
			}
			totalLinks++
		}

		o.logger.Debug("FDB walk created topology links",
			zap.String("switch_id", sw.ID),
			zap.Int("fdb_entries", len(entries)),
		)
	}

	if totalLinks > 0 {
		o.logger.Info("FDB topology links created",
			zap.Int("total_links", totalLinks),
		)
	}
}

// inferHierarchy runs network hierarchy inference after topology links are built.
func (o *ScanOrchestrator) inferHierarchy(ctx context.Context) {
	inferrer := NewHierarchyInferrer(o.store, o.logger)
	if err := inferrer.InferHierarchy(ctx); err != nil {
		o.logger.Error("hierarchy inference failed", zap.Error(err))
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

// analyzeWiFiConnections runs WiFi heuristic analysis on all devices after
// FDB data is available. Sets ConnectionType to "wired" for devices found in
// switch FDB tables and infers "wifi" for devices matching wireless heuristics.
func (o *ScanOrchestrator) analyzeWiFiConnections(ctx context.Context) {
	analyzer := NewWiFiHeuristicAnalyzer(o.store, o.logger.Named("wifi-heuristic"))
	analyzer.AnalyzeAll(ctx)
}

// scanWifiNetworks discovers nearby WiFi access points via OS APIs and upserts
// them as devices. Runs before port-scan so newly discovered APs are available
// for subsequent stages.
func (o *ScanOrchestrator) scanWifiNetworks(ctx context.Context) {
	if o.wifiScanner == nil || !o.wifiScanner.Available() {
		return
	}

	aps, err := o.wifiScanner.Scan(ctx)
	if err != nil {
		o.logger.Warn("wifi scan failed", zap.Error(err))
		return
	}

	for _, ap := range aps {
		if ap.BSSID == "" {
			continue
		}

		device := &models.Device{
			Hostname:        ap.SSID,
			MACAddress:      ap.BSSID,
			DeviceType:      models.DeviceTypeAccessPoint,
			Status:          models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryWiFi,
			ConnectionType:  models.ConnectionWiFi,
		}

		created, upsertErr := o.store.UpsertDevice(ctx, device)
		if upsertErr != nil {
			o.logger.Error("failed to upsert wifi AP",
				zap.String("bssid", ap.BSSID), zap.Error(upsertErr))
			continue
		}
		if created {
			o.publishEvent(ctx, TopicDeviceDiscovered, &DeviceEvent{Device: device})
		}
	}

	o.logger.Info("wifi scan complete", zap.Int("aps_found", len(aps)))
}
