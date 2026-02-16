package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/HerbHall/subnetree/internal/recon"
	"github.com/HerbHall/subnetree/pkg/models"
	"github.com/google/uuid"
)

// SeedDemoNetwork populates the database with a realistic 20-device home
// network. It is idempotent: UpsertDevice matches on MAC address, so
// re-running is safe.
func SeedDemoNetwork(ctx context.Context, reconStore *recon.ReconStore) error {
	now := time.Now().UTC()

	devices := demoDevices(now)
	deviceIDs := make(map[string]string, len(devices)) // hostname -> ID

	for i := range devices {
		if _, err := reconStore.UpsertDevice(ctx, &devices[i]); err != nil {
			return fmt.Errorf("seed device %s: %w", devices[i].Hostname, err)
		}
		deviceIDs[devices[i].Hostname] = devices[i].ID
	}

	if err := seedTopologyLinks(ctx, reconStore, deviceIDs, now); err != nil {
		return fmt.Errorf("seed topology: %w", err)
	}

	if err := seedHierarchy(ctx, reconStore, deviceIDs); err != nil {
		return fmt.Errorf("seed hierarchy: %w", err)
	}

	if err := seedScanHistory(ctx, reconStore, len(devices), now); err != nil {
		return fmt.Errorf("seed scans: %w", err)
	}

	return nil
}

// demoDevices returns 20 devices representing a typical home network.
func demoDevices(now time.Time) []models.Device {
	return []models.Device{
		// Router
		{
			ID: uuid.New().String(), Hostname: "ubiquiti-gateway",
			IPAddresses: []string{"192.168.1.1"}, MACAddress: "24:5A:4C:01:00:01",
			Manufacturer: "Ubiquiti Inc.", DeviceType: models.DeviceTypeRouter,
			OS: "EdgeOS 2.0", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverySNMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Network Closet", Category: "infrastructure", PrimaryRole: "gateway",
			ClassificationConfidence: 75, ClassificationSource: "snmp_sysservices",
			ClassificationSignals: `{"snmp_sysservices":30,"oui":15,"port_profile":15,"ttl":10}`,
		},
		// Managed switch
		{
			ID: uuid.New().String(), Hostname: "cisco-switch-01",
			IPAddresses: []string{"192.168.1.2"}, MACAddress: "00:1A:A1:02:00:01",
			Manufacturer: "Cisco Systems", DeviceType: models.DeviceTypeSwitch,
			OS: "IOS 15.2", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverySNMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Network Closet", Category: "infrastructure", PrimaryRole: "core-switch",
			ClassificationConfidence: 85, ClassificationSource: "snmp_bridge_mib",
			ClassificationSignals: `{"snmp_bridge_mib":35,"oui":15,"port_profile":15,"lldp":40}`,
		},
		// Unmanaged switch
		{
			ID: uuid.New().String(), Hostname: "tp-link-switch",
			IPAddresses: []string{"192.168.1.3"}, MACAddress: "50:C7:BF:03:00:01",
			Manufacturer: "TP-Link Technologies", DeviceType: models.DeviceTypeSwitch,
			Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-5 * 24 * time.Hour), LastSeen: now,
			Location: "Living Room", Category: "infrastructure", PrimaryRole: "edge-switch",
			ClassificationConfidence: 25, ClassificationSource: "heuristic",
			ClassificationSignals: `{"oui":15,"ttl":10}`,
		},
		// Access point
		{
			ID: uuid.New().String(), Hostname: "unifi-ap-lr",
			IPAddresses: []string{"192.168.1.5"}, MACAddress: "24:5A:4C:05:00:01",
			Manufacturer: "Ubiquiti Inc.", DeviceType: models.DeviceTypeAccessPoint,
			OS: "UniFi 7.1", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverySNMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Hallway", Category: "infrastructure", PrimaryRole: "wireless-ap",
			ClassificationConfidence: 70, ClassificationSource: "lldp",
			ClassificationSignals: `{"lldp":40,"oui":15,"port_profile":15}`,
		},
		// Server 1
		{
			ID: uuid.New().String(), Hostname: "proxmox-host",
			IPAddresses: []string{"192.168.1.10"}, MACAddress: "D4:BE:D9:10:00:01",
			Manufacturer: "Dell Inc.", DeviceType: models.DeviceTypeServer,
			OS: "Proxmox VE 8.1", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryAgent,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Server Rack", Category: "compute", PrimaryRole: "hypervisor",
			Tags: []string{"homelab", "proxmox"},
			ClassificationConfidence: 90, ClassificationSource: "agent",
		},
		// Server 2
		{
			ID: uuid.New().String(), Hostname: "docker-host",
			IPAddresses: []string{"192.168.1.11"}, MACAddress: "D4:BE:D9:11:00:01",
			Manufacturer: "Dell Inc.", DeviceType: models.DeviceTypeServer,
			OS: "Ubuntu 24.04 LTS", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryAgent,
			FirstSeen: now.Add(-6 * 24 * time.Hour), LastSeen: now,
			Location: "Server Rack", Category: "compute", PrimaryRole: "container-host",
			Tags: []string{"homelab", "docker"},
			ClassificationConfidence: 90, ClassificationSource: "agent",
		},
		// Desktop 1
		{
			ID: uuid.New().String(), Hostname: "gaming-pc",
			IPAddresses: []string{"192.168.1.20"}, MACAddress: "A8:A1:59:20:00:01",
			Manufacturer: "ASUSTek Computer", DeviceType: models.DeviceTypeDesktop,
			OS: "Windows 11 Pro", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Office", Category: "endpoint", PrimaryRole: "workstation",
			ClassificationConfidence: 55, ClassificationSource: "composite",
			ClassificationSignals: `{"oui":15,"port_profile":15,"ttl":10}`,
		},
		// Desktop 2
		{
			ID: uuid.New().String(), Hostname: "work-desktop",
			IPAddresses: []string{"192.168.1.21"}, MACAddress: "3C:7C:3F:21:00:01",
			Manufacturer: "ASUSTek Computer", DeviceType: models.DeviceTypeDesktop,
			OS: "Windows 11 Pro", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-5 * 24 * time.Hour), LastSeen: now,
			Location: "Office", Category: "endpoint", PrimaryRole: "workstation",
			ClassificationConfidence: 50, ClassificationSource: "composite",
		},
		// Desktop 3
		{
			ID: uuid.New().String(), Hostname: "media-center",
			IPAddresses: []string{"192.168.1.22"}, MACAddress: "B4:2E:99:22:00:01",
			Manufacturer: "Intel Corporate", DeviceType: models.DeviceTypeDesktop,
			OS: "Ubuntu 22.04", Status: models.DeviceStatusOffline,
			DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now.Add(-2 * 24 * time.Hour),
			Location: "Living Room", Category: "media", PrimaryRole: "htpc",
			ClassificationConfidence: 45, ClassificationSource: "composite",
		},
		// Laptop 1
		{
			ID: uuid.New().String(), Hostname: "macbook-pro",
			IPAddresses: []string{"192.168.1.30"}, MACAddress: "A4:83:E7:30:00:01",
			Manufacturer: "Apple, Inc.", DeviceType: models.DeviceTypeLaptop,
			OS: "macOS 15.2", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverymDNS,
			FirstSeen: now.Add(-3 * 24 * time.Hour), LastSeen: now,
			Location: "Office", Category: "endpoint", Owner: "admin",
			ClassificationConfidence: 60, ClassificationSource: "mdns",
		},
		// Laptop 2
		{
			ID: uuid.New().String(), Hostname: "thinkpad-t14",
			IPAddresses: []string{"192.168.1.31"}, MACAddress: "8C:8C:AA:31:00:01",
			Manufacturer: "Lenovo", DeviceType: models.DeviceTypeLaptop,
			OS: "Fedora 41", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-4 * 24 * time.Hour), LastSeen: now,
			Location: "Office", Category: "endpoint", Owner: "admin",
			ClassificationConfidence: 50, ClassificationSource: "composite",
		},
		// NAS
		{
			ID: uuid.New().String(), Hostname: "synology-nas",
			IPAddresses: []string{"192.168.1.40"}, MACAddress: "00:11:32:40:00:01",
			Manufacturer: "Synology Incorporated", DeviceType: models.DeviceTypeNAS,
			OS: "DSM 7.2", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverySNMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Server Rack", Category: "storage", PrimaryRole: "file-server",
			Tags: []string{"backup", "media"},
			ClassificationConfidence: 80, ClassificationSource: "snmp_sysservices",
		},
		// Printer
		{
			ID: uuid.New().String(), Hostname: "hp-laserjet",
			IPAddresses: []string{"192.168.1.50"}, MACAddress: "3C:D9:2B:50:00:01",
			Manufacturer: "HP Inc.", DeviceType: models.DeviceTypePrinter,
			OS: "FutureSmart 5.6", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverymDNS,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Office", Category: "peripheral", PrimaryRole: "printer",
			ClassificationConfidence: 70, ClassificationSource: "mdns",
		},
		// IoT 1 - Smart plug
		{
			ID: uuid.New().String(), Hostname: "smart-plug-living",
			IPAddresses: []string{"192.168.1.60"}, MACAddress: "68:57:2D:60:00:01",
			Manufacturer: "TP-Link Technologies", DeviceType: models.DeviceTypeIoT,
			Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-6 * 24 * time.Hour), LastSeen: now,
			Location: "Living Room", Category: "iot", PrimaryRole: "smart-plug",
			ClassificationConfidence: 35, ClassificationSource: "oui",
		},
		// IoT 2 - Camera
		{
			ID: uuid.New().String(), Hostname: "cam-front-door",
			IPAddresses: []string{"192.168.1.61"}, MACAddress: "9C:8E:CD:61:00:01",
			Manufacturer: "Reolink Innovation", DeviceType: models.DeviceTypeCamera,
			Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Front Porch", Category: "security", PrimaryRole: "camera",
			ClassificationConfidence: 40, ClassificationSource: "oui",
		},
		// Phone 1
		{
			ID: uuid.New().String(), Hostname: "iphone-15-pro",
			IPAddresses: []string{"192.168.1.70"}, MACAddress: "F8:4D:89:70:00:01",
			Manufacturer: "Apple, Inc.", DeviceType: models.DeviceTypePhone,
			Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoverymDNS,
			FirstSeen: now.Add(-2 * 24 * time.Hour), LastSeen: now,
			Category: "mobile", Owner: "admin",
			ClassificationConfidence: 55, ClassificationSource: "mdns",
		},
		// Phone 2
		{
			ID: uuid.New().String(), Hostname: "pixel-8",
			IPAddresses: []string{"192.168.1.71"}, MACAddress: "DC:E5:5B:71:00:01",
			Manufacturer: "Google, Inc.", DeviceType: models.DeviceTypePhone,
			Status: models.DeviceStatusOffline, DiscoveryMethod: models.DiscoverymDNS,
			FirstSeen: now.Add(-1 * 24 * time.Hour), LastSeen: now.Add(-6 * time.Hour),
			Category: "mobile",
			ClassificationConfidence: 55, ClassificationSource: "mdns",
		},
		// Tablet
		{
			ID: uuid.New().String(), Hostname: "galaxy-tab-s9",
			IPAddresses: []string{"192.168.1.72"}, MACAddress: "C0:A8:E0:72:00:01",
			Manufacturer: "Samsung Electronics", DeviceType: models.DeviceTypeTablet,
			Status: models.DeviceStatusOffline, DiscoveryMethod: models.DiscoverymDNS,
			FirstSeen: now.Add(-3 * 24 * time.Hour), LastSeen: now.Add(-12 * time.Hour),
			Category: "mobile",
			ClassificationConfidence: 50, ClassificationSource: "mdns",
		},
		// Firewall
		{
			ID: uuid.New().String(), Hostname: "pfsense-fw",
			IPAddresses: []string{"192.168.1.254"}, MACAddress: "00:08:A2:FE:00:01",
			Manufacturer: "Netgate", DeviceType: models.DeviceTypeFirewall,
			OS: "pfSense CE 2.7", Status: models.DeviceStatusOnline,
			DiscoveryMethod: models.DiscoverySNMP,
			FirstSeen: now.Add(-7 * 24 * time.Hour), LastSeen: now,
			Location: "Network Closet", Category: "security", PrimaryRole: "firewall",
			ClassificationConfidence: 80, ClassificationSource: "snmp_sysservices",
			ClassificationSignals: `{"snmp_sysservices":30,"oui":15,"port_profile":15,"ttl":10}`,
		},
		// Unknown device
		{
			ID: uuid.New().String(), Hostname: "",
			IPAddresses: []string{"192.168.1.99"}, MACAddress: "AA:BB:CC:99:00:01",
			DeviceType: models.DeviceTypeUnknown,
			Status: models.DeviceStatusOnline, DiscoveryMethod: models.DiscoveryICMP,
			FirstSeen: now.Add(-1 * 24 * time.Hour), LastSeen: now,
			ClassificationConfidence: 0, ClassificationSource: "none",
		},
	}
}

// seedTopologyLinks creates a hierarchical network topology:
// firewall <-> router <-> managed switch <-> downstream devices.
func seedTopologyLinks(ctx context.Context, store *recon.ReconStore, ids map[string]string, now time.Time) error {
	links := []recon.TopologyLink{
		// Firewall -> Router
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["pfsense-fw"], TargetDeviceID: ids["ubiquiti-gateway"],
			SourcePort: "LAN", TargetPort: "WAN",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
		// Router -> Managed switch
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["ubiquiti-gateway"], TargetDeviceID: ids["cisco-switch-01"],
			SourcePort: "LAN1", TargetPort: "Gi0/1",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
		// Managed switch -> Access point
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["cisco-switch-01"], TargetDeviceID: ids["unifi-ap-lr"],
			SourcePort: "Gi0/2", TargetPort: "ETH0",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
		// Managed switch -> Unmanaged switch
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["cisco-switch-01"], TargetDeviceID: ids["tp-link-switch"],
			SourcePort: "Gi0/3", TargetPort: "Port 1",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-5 * 24 * time.Hour), LastConfirmed: now,
		},
		// Managed switch -> Server 1
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["cisco-switch-01"], TargetDeviceID: ids["proxmox-host"],
			SourcePort: "Gi0/10", TargetPort: "eth0",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
		// Managed switch -> Server 2
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["cisco-switch-01"], TargetDeviceID: ids["docker-host"],
			SourcePort: "Gi0/11", TargetPort: "eth0",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-6 * 24 * time.Hour), LastConfirmed: now,
		},
		// Managed switch -> NAS
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["cisco-switch-01"], TargetDeviceID: ids["synology-nas"],
			SourcePort: "Gi0/12", TargetPort: "LAN 1",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
		// Unmanaged switch -> Desktops
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["tp-link-switch"], TargetDeviceID: ids["gaming-pc"],
			SourcePort: "Port 2", TargetPort: "eth0",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["tp-link-switch"], TargetDeviceID: ids["work-desktop"],
			SourcePort: "Port 3", TargetPort: "eth0",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-5 * 24 * time.Hour), LastConfirmed: now,
		},
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["tp-link-switch"], TargetDeviceID: ids["media-center"],
			SourcePort: "Port 4", TargetPort: "eth0",
			LinkType: "ethernet", Speed: 1000,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now.Add(-2 * 24 * time.Hour),
		},
		// Managed switch -> Printer
		{
			ID: uuid.New().String(),
			SourceDeviceID: ids["cisco-switch-01"], TargetDeviceID: ids["hp-laserjet"],
			SourcePort: "Gi0/20", TargetPort: "eth0",
			LinkType: "ethernet", Speed: 100,
			DiscoveredAt: now.Add(-7 * 24 * time.Hour), LastConfirmed: now,
		},
	}

	for i := range links {
		if err := store.UpsertTopologyLink(ctx, &links[i]); err != nil {
			return fmt.Errorf("link %s->%s: %w", links[i].SourcePort, links[i].TargetPort, err)
		}
	}

	return nil
}

// seedHierarchy assigns network layers and parent device IDs to seeded devices.
// This runs after devices and topology links are created.
func seedHierarchy(ctx context.Context, store *recon.ReconStore, ids map[string]string) error {
	type assignment struct {
		hostname string
		parent   string // hostname of parent (empty for root)
		layer    int
	}

	assignments := []assignment{
		// Gateway layer (1): routers, firewalls
		{"pfsense-fw", "ubiquiti-gateway", models.NetworkLayerGateway},
		{"ubiquiti-gateway", "", models.NetworkLayerGateway},

		// Distribution layer (2): core/managed switch
		{"cisco-switch-01", "ubiquiti-gateway", models.NetworkLayerDistribution},

		// Access layer (3): edge switch, AP
		{"tp-link-switch", "cisco-switch-01", models.NetworkLayerAccess},
		{"unifi-ap-lr", "cisco-switch-01", models.NetworkLayerAccess},

		// Endpoint layer (4): everything else
		{"proxmox-host", "cisco-switch-01", models.NetworkLayerEndpoint},
		{"docker-host", "cisco-switch-01", models.NetworkLayerEndpoint},
		{"synology-nas", "cisco-switch-01", models.NetworkLayerEndpoint},
		{"hp-laserjet", "cisco-switch-01", models.NetworkLayerEndpoint},
		{"gaming-pc", "tp-link-switch", models.NetworkLayerEndpoint},
		{"work-desktop", "tp-link-switch", models.NetworkLayerEndpoint},
		{"media-center", "tp-link-switch", models.NetworkLayerEndpoint},
		{"macbook-pro", "unifi-ap-lr", models.NetworkLayerEndpoint},
		{"thinkpad-t14", "unifi-ap-lr", models.NetworkLayerEndpoint},
		{"iphone-15-pro", "unifi-ap-lr", models.NetworkLayerEndpoint},
		{"pixel-8", "unifi-ap-lr", models.NetworkLayerEndpoint},
		{"galaxy-tab-s9", "unifi-ap-lr", models.NetworkLayerEndpoint},
		{"smart-plug-living", "ubiquiti-gateway", models.NetworkLayerEndpoint},
		{"cam-front-door", "ubiquiti-gateway", models.NetworkLayerEndpoint},
	}

	for _, a := range assignments {
		deviceID, ok := ids[a.hostname]
		if !ok {
			continue
		}
		parentID := ""
		if a.parent != "" {
			parentID = ids[a.parent]
		}
		if err := store.UpdateDeviceHierarchy(ctx, deviceID, parentID, a.layer); err != nil {
			return fmt.Errorf("hierarchy %s: %w", a.hostname, err)
		}
	}

	return nil
}

// seedScanHistory creates 3 completed scan records spread over the last 7 days.
func seedScanHistory(ctx context.Context, store *recon.ReconStore, deviceCount int, now time.Time) error {
	scans := []struct {
		age       time.Duration
		duration  int64
		ping      int64
		enrich    int64
		postProc  int64
		alive     int
		created   int
		updated   int
	}{
		{7 * 24 * time.Hour, 42000, 18000, 20000, 4000, 15, 15, 0},
		{3 * 24 * time.Hour, 38000, 16000, 18000, 4000, 18, 3, 15},
		{6 * time.Hour, 35000, 15000, 16000, 4000, deviceCount - 3, 0, deviceCount - 3},
	}

	for _, s := range scans {
		scanTime := now.Add(-s.age)
		endTime := scanTime.Add(time.Duration(s.duration) * time.Millisecond)
		scanID := uuid.New().String()

		scan := &models.ScanResult{
			ID:        scanID,
			Subnet:    "192.168.1.0/24",
			StartedAt: scanTime.Format(time.RFC3339),
			EndedAt:   endTime.Format(time.RFC3339),
			Status:    "completed",
			Total:     deviceCount,
			Online:    s.alive,
		}
		if err := store.CreateScan(ctx, scan); err != nil {
			return fmt.Errorf("create scan: %w", err)
		}

		metrics := &models.ScanMetrics{
			ScanID:         scanID,
			DurationMs:     s.duration,
			PingPhaseMs:    s.ping,
			EnrichPhaseMs:  s.enrich,
			PostProcessMs:  s.postProc,
			HostsScanned:   254,
			HostsAlive:     s.alive,
			DevicesCreated: s.created,
			DevicesUpdated: s.updated,
			CreatedAt:      scanTime.Format(time.RFC3339),
		}
		if err := store.SaveScanMetrics(ctx, metrics); err != nil {
			return fmt.Errorf("save scan metrics: %w", err)
		}
	}

	return nil
}
